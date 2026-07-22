# 当前已实现 API 接口文档

> 本文只记录当前代码中已经实现并可调用的接口。完整家庭记账产品契约见 `../api/api-interface-v1.md`，其中账单、家庭、账簿接口路径已统一为 `/api/` 前缀，但当前代码尚未实现这些业务模块。

## 通用约定

### API 前缀

```text
/api/
```

### 统一响应

成功响应：

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

业务错误响应：

```json
{
  "error": 400,
  "message": "请求体不是合法 JSON",
  "data": null
}
```

说明：

- 成功响应统一使用 HTTP `200 OK`。
- 失败响应保留标准 HTTP 错误状态码。
- `GET /api/health/mysql` 是基础设施健康检查接口，为便于探针诊断，MySQL 不可用时仍会在 `data` 中返回健康状态和响应耗时。

## 错误码

| error | HTTP 状态码 | 说明 |
| --- | --- | --- |
| `400` | `400` | 请求参数不合法 |
| `401` | `401` | 登录态缺失、无效或已过期 |
| `403` | `403` | 权限不足 |
| `409` | `409` | 请求创建的资源已存在 |
| `404` | `404` | 目标资源不存在 |
| `500` | `500` | 服务内部错误 |
| `502` | `502` | 抖音开放平台上游不可用或返回错误 |
| `503` | `503` | MySQL 等依赖不可用或登录服务未配置 |

## 1. MySQL 健康检查

### 请求地址

```text
GET /api/health/mysql
```

### 请求参数

无。

### 成功响应

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

响应字段：

| 字段 | 类型 | 必返 | 说明 |
| --- | --- | --- | --- |
| `healthy` | boolean | 是 | MySQL 是否可用 |
| `response_time_ms` | integer | 是 | 健康检查耗时，单位毫秒 |

### 示例请求

```bash
curl "http://127.0.0.1:8000/api/health/mysql"
```

### 示例失败响应

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

## 2. 抖音小程序登录

### 请求地址

```text
POST /api/auth/douyin/login
```

### 请求参数

Content-Type: `application/json`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `code` | string | 是 | 前端 `tt.login` 返回的临时登录凭证 |

### 成功响应

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

响应字段：

| 字段 | 类型 | 必返 | 说明 |
| --- | --- | --- | --- |
| `token` | string | 是 | 服务端登录凭证，后续接口可通过 `Authorization: Bearer <token>` 携带 |
| `openid` | string | 是 | 抖音用户 openid |
| `unionid` | string | 是 | 抖音用户 unionid，可能为空字符串 |

### 示例请求

```bash
curl -X POST "http://127.0.0.1:8000/api/auth/douyin/login" \
  -H "Content-Type: application/json" \
  -d '{"code":"YOUR_TT_LOGIN_CODE"}'
```

### 示例失败响应

```json
{
  "error": 400,
  "message": "code 不能为空",
  "data": null
}
```

### 错误码

| error | HTTP 状态码 | 触发场景 |
| --- | --- | --- |
| `400` | `400` | 请求体不是合法 JSON，或缺少 `code` |
| `409` | `409` | OpenID 已存在，不能重复创建用户 |
| `500` | `500` | 服务端生成 token 或构造上游请求失败 |
| `502` | `502` | 抖音开放平台不可用、返回非 200、返回错误码或响应缺少关键字段 |
| `503` | `503` | 未配置 `DOUYIN_APP_ID` / `DOUYIN_APP_SECRET`，或登录态写入 MySQL 失败 |

## 3. Token 刷新

### 请求地址

```text
POST /api/auth/token/refresh
```

### 请求参数

token 支持两种传入方式，优先读取请求体 `token`，其次读取 `Authorization` 请求头。当前 token 必须未过期；刷新成功后旧 token 会立即失效。

请求体方式：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `token` | string | 否 | 当前未过期的服务端登录凭证 |

请求头方式：

| Header | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `Authorization` | string | 否 | 格式为 `Bearer <token>` |

### 成功响应

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "token": "new-server-generated-token"
  }
}
```

响应字段：

| 字段 | 类型 | 必返 | 说明 |
| --- | --- | --- | --- |
| `token` | string | 是 | 刷新后的服务端登录凭证 |

### 示例请求

请求体传 token：

```bash
curl -X POST "http://127.0.0.1:8000/api/auth/token/refresh" \
  -H "Content-Type: application/json" \
  -d '{"token":"server-generated-token"}'
```

Bearer token：

```bash
curl -X POST "http://127.0.0.1:8000/api/auth/token/refresh" \
  -H "Authorization: Bearer server-generated-token"
```

### 示例失败响应

```json
{
  "error": 401,
  "message": "登录态无效或已过期",
  "data": null
}
```

### 错误码

| error | HTTP 状态码 | 触发场景 |
| --- | --- | --- |
| `400` | `400` | 请求体不是合法 JSON |
| `401` | `401` | token 缺失、无效或已过期 |
| `503` | `503` | MySQL 登录态更新失败 |

## 4. 当前用户信息查询

### 请求地址

```text
POST /api/account/user_info
```

### 请求参数

token 支持两种传入方式，优先读取请求体 `token`，其次读取 `Authorization` 请求头。

请求体方式：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `token` | string | 否 | 服务端登录凭证 |

请求头方式：

| Header | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `Authorization` | string | 否 | 格式为 `Bearer <token>` |

### 成功响应

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

响应字段：

| 字段 | 类型 | 必返 | 说明 |
| --- | --- | --- | --- |
| `user_id` | integer | 是 | 用户 ID |
| `phone` | string | 是 | 手机号，未绑定时为空字符串 |
| `douyin_open_id` | string | 是 | 抖音 openid |
| `username` | string | 是 | 用户昵称；静默登录新用户会根据用户 ID 自动生成唯一默认昵称 |
| `avatar_url` | string | 是 | 用户头像地址；静默登录新用户使用默认头像 |
| `gender` | string | 是 | 性别，未设置时为空字符串 |
| `birthday` | string | 是 | 生日，格式 `YYYY-MM-DD`，未设置时为空字符串 |
| `current_household_id` | integer | 是 | 当前家庭 ID，未设置时为 `0` |

### 示例请求

请求体传 token：

```bash
curl -X POST "http://127.0.0.1:8000/api/account/user_info" \
  -H "Content-Type: application/json" \
  -d '{"token":"server-generated-token"}'
```

Bearer token：

```bash
curl -X POST "http://127.0.0.1:8000/api/account/user_info" \
  -H "Authorization: Bearer server-generated-token"
```

### 示例失败响应

```json
{
  "error": 401,
  "message": "登录态无效或已过期",
  "data": null
}
```

### 错误码

| error | HTTP 状态码 | 触发场景 |
| --- | --- | --- |
| `400` | `400` | 请求体不是合法 JSON |
| `401` | `401` | token 缺失、无效或已过期 |
| `404` | `404` | token 有效但用户资料创建后仍无法查询 |
| `503` | `503` | MySQL 用户存储不可用 |

## 迁移说明

- 响应结构从旧版 `err_no + err_msg + data` 统一迁移为 `error + message + data`，其中 `error` 成功时为 `0`，失败时使用数字 HTTP 状态码。
- 当前已实现路径为 `/api/health/mysql`、`/api/auth/douyin/login`、`/api/auth/token/refresh`、`/api/account/user_info`。
- `api-interface-v1.md` 中所有产品接口已统一为 `/api/` 前缀，不再使用 `/api/v1/`。
- `api-interface-v1.md` 中的账单、家庭、账簿接口是产品契约定义，当前代码尚未实现对应路由；实现前前端不应直接调用这些端点。
