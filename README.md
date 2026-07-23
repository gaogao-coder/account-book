# douyincloud-gin-demo

本项目是一个基于 Go + Gin 的抖音云托管示例服务，提供 MySQL 健康检查、抖音小程序登录和用户信息查询接口。
当前已实现接口的完整说明见 [docs/API.md](./docs/API.md)。

服务依赖以下外部组件：

- MySQL

## 当前能力

### MySQL 健康检查接口

- `GET /api/health/mysql`：检查 MySQL 连接状态，并返回连接是否正常和响应时间
- MySQL 连接失败或超时时返回 HTTP `503 Service Unavailable`

### 抖音小程序登录接口

- `POST /api/auth/douyin/login`

服务端会使用前端 `tt.login` 返回的 `code` 调用抖音开放平台官方 `jscode2session`，拿到 `openid`、`unionid`、`session_key` 后生成一个服务端 token，并把登录态写入 MySQL。

说明：

- 该接口依赖 `DOUYIN_APP_ID` 和 `DOUYIN_APP_SECRET`
- 登录接口固定请求抖音官方接口，不支持通过环境变量切换 mock 地址
- 登录态表 `douyin_users` 会在首次登录时自动创建或迁移
- 登录成功后会同步初始化 `users`、`user_profiles`、`user_settings` 基础用户数据
- OpenID 已存在时会刷新登录态并复用已有用户资料，允许用户重新登录
- `POST /api/auth/token/refresh` 可使用当前未过期 token 换取新的服务端 token

### 用户信息接口

- `POST /api/account/user_info`：校验服务端 token，并返回当前用户基础信息
- token 支持通过 JSON 请求体 `token` 字段或 `Authorization: Bearer <token>` 请求头传入
- token 缺失、无效或过期时返回 HTTP `401 Unauthorized`
- token 有效但用户基础数据缺失时，会自动创建默认用户资料后返回
- 静默登录无法直接获取抖音昵称和头像，新用户会自动生成唯一默认用户名并分配统一默认头像

默认资料规则：

- 默认用户名按用户自增 ID 生成，格式为 `用户<base36_id>`，避免系统内重名
- 默认头像 URL 为 `https://tt35b94304aab3ccd201-env-jyqimo1dsz.tos-cn-beijing.volces.com/4f5c5effb214491c8e4753aadeb3b4b7~tplv-nvscq0fgd4-jpg.jpeg`
- 同一 OpenID 重新登录时不会重新生成用户名和头像

## 项目目录结构

当前项目已按标准 Go 服务目录完成重构：应用入口在 `cmd/`，核心业务在 `internal/`，领域模型、业务逻辑和数据访问按 DDD 分层放置。

```text
douyincloud-gin-demo/
├── cmd/
│   ├── api/main.go                 # API 服务入口：只加载配置、创建路由、启动服务
│   └── job/main.go                 # 定时任务/消费者入口，当前预留
├── internal/
│   ├── config/                     # 配置加载
│   ├── domain/                     # 领域模型和领域错误
│   ├── handler/                    # Gin 路由、HTTP handler、统一响应
│   ├── middleware/                 # Gin 中间件，当前预留
│   ├── repository/                 # MySQL 连接、登录态存储、用户资料查询
│   └── service/                    # 登录、健康检查、用户查询业务逻辑
├── pkg/
│   ├── logger/                     # 可复用日志工具
│   └── validator/                  # 可复用校验工具
├── api/
│   ├── proto/                      # protobuf IDL，当前预留
│   ├── openapi.yaml                # OpenAPI 基础定义
│   └── api-interface-v1.md         # 家庭记账 V1 API 契约定义
├── configs/app.yaml                # 配置文件模板
├── scripts/
│   ├── docker/                     # Docker 脚本目录，当前预留
│   └── migrate/001_init.sql        # MySQL 初始化脚本
├── test/                           # 外部测试资源和 e2e 测试，当前预留
├── web/                            # 静态资源和 HTML 模板，当前预留
├── assets/                         # 图片和资源文件，当前预留
├── docs/
│   ├── API.md                      # 当前已实现接口文档
│   └── backend-technical-solution-v1.md
├── .env.example                    # 本地环境变量模板
├── docker-compose.yml              # 本地 MySQL 依赖编排
├── Dockerfile                      # 抖音云托管镜像构建文件
├── Makefile                        # 本地运行、测试、构建命令
├── run.sh                          # 容器启动脚本
├── go.mod                          # Go module 定义
├── go.sum                          # Go 依赖校验文件
├── LICENSE                         # 开源许可证
└── README.md                       # 项目说明文档
```

## 本地运行

### 1. 准备依赖

仓库提供了一个本地开发用的 `docker-compose.yml`，会启动 MySQL。

```bash
docker compose up -d
```

如果你的环境还在使用旧版 Docker Compose，也可以执行：

```bash
docker-compose up -d
```

### 2. 准备环境变量

复制 `.env.example` 并按需填写：

```bash
cp .env.example .env
```

最小本地运行至少需要保证以下变量正确：

- `MYSQL_ADDRESS`
- `MYSQL_USERNAME`
- `MYSQL_PASSWORD`
- `MYSQL_DATABASE`

如果你要调试 `/api/auth/douyin/login`，还需要补充：

- `DOUYIN_APP_ID`
- `DOUYIN_APP_SECRET`

说明：

- 代码默认监听 `:8000`，可通过 `PORT` 环境变量覆盖
- 没有有效的抖音开放平台凭据时，`/api/health/mysql` 仍可使用，但 `/api/auth/douyin/login` 会失败

### 3. 启动服务

```bash
set -a
source .env
set +a
go run ./cmd/api
```

如果你更习惯一次性注入环境变量，也可以使用自己的方式启动。

## 接口示例

### `GET /api/health/mysql`

请求：

```bash
curl "http://127.0.0.1:8000/api/health/mysql"
```

响应示例：

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "healthy": true,
    "response_time_ms": 3
  }
}
```

失败响应示例：

```json
{
  "error": 503,
  "message": "MySQL 连接不可用",
  "data": {
    "healthy": false,
    "response_time_ms": 5000
  }
}
```

### `POST /api/auth/douyin/login`

请求：

```bash
curl -X POST "http://127.0.0.1:8000/api/auth/douyin/login" \
  -H "Content-Type: application/json" \
  -d '{"code":"YOUR_TT_LOGIN_CODE"}'
```

响应示例：

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "token": "server-generated-token",
    "openid": "real-openid-from-douyin",
    "unionid": "real-unionid-from-douyin"
  }
}
```

常见失败：

- 请求体缺少 `code` 或 JSON 格式错误：HTTP `400`
- `code` 无效或已过期：HTTP `400`
- 未配置 `DOUYIN_APP_ID` / `DOUYIN_APP_SECRET`：HTTP `503`
- 抖音开放平台不可用或返回非 200：HTTP `502`
- MySQL 登录态写入失败：HTTP `503`

### `POST /api/auth/token/refresh`

请求体传 token：

```bash
curl -X POST "http://127.0.0.1:8000/api/auth/token/refresh" \
  -H "Content-Type: application/json" \
  -d '{"token":"server-generated-token"}'
```

或使用 Bearer token：

```bash
curl -X POST "http://127.0.0.1:8000/api/auth/token/refresh" \
  -H "Authorization: Bearer server-generated-token"
```

响应示例：

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "token": "new-server-generated-token"
  }
}
```

常见失败：

- token 缺失、无效或过期：HTTP `401`
- MySQL 登录态更新失败：HTTP `503`

### `POST /api/account/user_info`

请求体传 token：

```bash
curl -X POST "http://127.0.0.1:8000/api/account/user_info" \
  -H "Content-Type: application/json" \
  -d '{"token":"server-generated-token"}'
```

或使用 Bearer token：

```bash
curl -X POST "http://127.0.0.1:8000/api/account/user_info" \
  -H "Authorization: Bearer server-generated-token"
```

响应示例：

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "user_id": 1,
    "phone": "",
    "douyin_open_id": "real-openid-from-douyin",
    "username": "用户1",
    "avatar_url": "https://tt35b94304aab3ccd201-env-jyqimo1dsz.tos-cn-beijing.volces.com/4f5c5effb214491c8e4753aadeb3b4b7~tplv-nvscq0fgd4-jpg.jpeg",
    "gender": "",
    "birthday": "",
    "current_household_id": 0
  }
}
```

常见失败：

- token 缺失、无效或过期：HTTP `401`
- token 有效但用户资料创建后仍无法查询：HTTP `404`
- MySQL 不可用：HTTP `503`

## 环境变量说明

### MySQL

- `MYSQL_ADDRESS`
- `MYSQL_USERNAME`
- `MYSQL_PASSWORD`
- `MYSQL_DATABASE`

### 抖音登录

- `DOUYIN_APP_ID`
- `DOUYIN_APP_SECRET`

## 抖音云托管部署说明

抖音云托管支持基于 Git 代码或 Docker 镜像部署。本仓库中的 `Dockerfile` 可作为镜像部署参考。

在抖音云托管平台上启用组件后，平台会自动将组件地址、账号和密码注入到环境变量中。部署在平台上的服务日志应输出到标准输出，便于在平台日志功能中查看。

## 已知限制

- `/api/auth/douyin/login` 依赖外网访问抖音开放平台
- 当前开发阶段暂不保留自动化测试文件，待自动化测试环境配置完成后再补充

## License

This project is licensed under the [Apache-2.0 License](LICENSE).
