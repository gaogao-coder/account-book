# douyincloud-gin-demo

本项目是一个基于 Go + Gin 的抖音云托管示例服务，提供 MySQL 健康检查、抖音小程序登录和用户信息查询接口。

服务依赖以下外部组件：

- MySQL

## 当前能力

### MySQL 健康检查接口

- `GET /api/hello`：检查 MySQL 连接状态，并返回连接是否正常和响应时间
- MySQL 连接失败或超时时返回 HTTP `503 Service Unavailable`

### 抖音小程序登录接口

- `POST /api/apps/login`

服务端会使用前端 `tt.login` 返回的 `code` 调用抖音开放平台官方 `jscode2session`，拿到 `openid`、`unionid`、`session_key` 后生成一个服务端 token，并把登录态写入 MySQL。

说明：

- 该接口依赖 `DOUYIN_APP_ID` 和 `DOUYIN_APP_SECRET`
- 登录接口固定请求抖音官方接口，不支持通过环境变量切换 mock 地址
- 登录态表 `douyin_users` 会在首次登录时自动创建或迁移
- 登录成功后会同步初始化 `users`、`user_profiles`、`user_settings` 基础用户数据
- 当前仓库只实现服务端 token 校验逻辑，不包含 token 刷新流程

### 用户信息接口

- `POST /api/account/get_user_info`：校验服务端 token，并返回当前用户基础信息
- token 支持通过 JSON 请求体 `token` 字段或 `Authorization: Bearer <token>` 请求头传入
- token 缺失、无效或过期时返回 HTTP `401 Unauthorized`
- token 有效但用户基础数据缺失时，会自动创建默认用户资料后返回
- 静默登录无法直接获取抖音昵称和头像，新用户会自动生成唯一默认用户名并分配统一默认头像

默认资料规则：

- 默认用户名按用户自增 ID 生成，格式为 `用户<base36_id>`，避免系统内重名
- 默认头像 URL 为 `https://lf3-static.bytednsdoc.com/obj/eden-cn/default-avatar.png`
- 重复登录同一用户时返回已保存的用户资料，不会重新生成用户名和头像

## 目录结构

```text
.
├── Dockerfile
├── README.md
├── docker-compose.yml
├── .env.example
├── main.go
├── run.sh
├── component
│   ├── auth_store.go
│   ├── mysql.go
│   ├── types.go
│   ├── user_store.go
│   └── user_store_test.go
├── module
│   └── user
│       ├── router.go
│       ├── router_test.go
│       ├── schema.go
│       └── service.go
└── service
    ├── login.go
    └── service.go
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

如果你要调试 `/api/apps/login`，还需要补充：

- `DOUYIN_APP_ID`
- `DOUYIN_APP_SECRET`

说明：

- 代码默认监听 `:8000`
- `.env.example` 中的 `PORT` 仅用于记录默认端口，当前代码不会读取该变量
- 没有有效的抖音开放平台凭据时，`/api/hello` 仍可使用，但 `/api/apps/login` 会失败

### 3. 启动服务

```bash
set -a
source .env
set +a
go run .
```

如果你更习惯一次性注入环境变量，也可以使用自己的方式启动。

## 接口示例

### `GET /api/hello`

请求：

```bash
curl "http://127.0.0.1:8000/api/hello"
```

响应示例：

```json
{
  "err_no": 0,
  "err_msg": "success",
  "data": {
    "healthy": true,
    "response_time_ms": 3
  }
}
```

失败响应示例：

```json
{
  "err_no": -1,
  "err_msg": "mysql connection unavailable",
  "data": {
    "healthy": false,
    "response_time_ms": 5000
  }
}
```

### `POST /api/apps/login`

请求：

```bash
curl -X POST "http://127.0.0.1:8000/api/apps/login" \
  -H "Content-Type: application/json" \
  -d '{"code":"YOUR_TT_LOGIN_CODE"}'
```

响应示例：

```json
{
  "err_no": 0,
  "err_msg": "success",
  "data": {
    "token": "server-generated-token",
    "openid": "real-openid-from-douyin",
    "unionid": "real-unionid-from-douyin"
  }
}
```

常见失败：

- 请求体缺少 `code` 或 JSON 格式错误：HTTP `400`
- 未配置 `DOUYIN_APP_ID` / `DOUYIN_APP_SECRET`：HTTP `503`
- 抖音开放平台不可用或返回错误：HTTP `502`
- MySQL 登录态写入失败：HTTP `503`

### `POST /api/account/get_user_info`

请求体传 token：

```bash
curl -X POST "http://127.0.0.1:8000/api/account/get_user_info" \
  -H "Content-Type: application/json" \
  -d '{"token":"server-generated-token"}'
```

或使用 Bearer token：

```bash
curl -X POST "http://127.0.0.1:8000/api/account/get_user_info" \
  -H "Authorization: Bearer server-generated-token"
```

响应示例：

```json
{
  "err_no": 0,
  "err_msg": "success",
  "data": {
    "user_id": 1,
    "phone": "",
    "douyin_open_id": "real-openid-from-douyin",
    "username": "用户1",
    "avatar_url": "https://lf3-static.bytednsdoc.com/obj/eden-cn/default-avatar.png",
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

- `/api/apps/login` 依赖外网访问抖音开放平台
- 当前只实现 token 校验和用户信息查询，不包含 token 刷新流程
- 自动化测试覆盖未登录访问、新用户首次登录、重复登录和默认用户名唯一性；真实 MySQL 集成场景需要可连接的 MySQL 和登录测试数据

## License

This project is licensed under the [Apache-2.0 License](LICENSE).
