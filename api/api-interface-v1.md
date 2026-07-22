# 家庭记账应用 V1.0 API 接口定义

> 本文定义前后端对接使用的 HTTP API Interface。API Interface 只描述调用方必须知道的契约：路径、请求、响应、字段约束、错误码和关键不变量。事务、统计更新、操作日志、权限判断细节属于后端 Implementation。

## 1. 通用约定

### 1.1 API 前缀

```text
/api/
```

### 1.2 统一响应

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {}
}
```

错误响应：

```json
{
  "code": "PERMISSION_DENIED",
  "message": "权限不足",
  "data": null
}
```

V1.0 的错误响应只保留 `code + message` 作为错误表达；`data` 固定为 `null`，不返回 `field_errors` 或其他字段级错误结构。

HTTP 错误状态码规则：

- V1.0 成功响应统一使用 `200 OK`。
- V1.0 失败响应保留标准 HTTP 错误状态码，不统一压成 `200`。
- 推荐映射：
  - `INVALID_ARGUMENT`、`INVALID_AMOUNT`、`CATEGORY_TYPE_MISMATCH`、`READONLY_FIELD_SUBMITTED` -> `400 Bad Request`
  - `PERMISSION_DENIED` -> `403 Forbidden`
  - `HOUSEHOLD_NOT_FOUND`、`HOUSEHOLD_MEMBER_NOT_FOUND`、`LEDGER_NOT_FOUND`、`BILL_NOT_FOUND` -> `404 Not Found`
  - `INVITE_EXPIRED`、`INVITE_USED` -> `400 Bad Request`

### 1.3 时间格式

所有时间字段使用 ISO 8601 字符串，包含时区信息。
请求中的时间字段可以携带合法时区；响应中的时间字段统一返回标准时区格式。

示例：

```text
2026-07-03T10:30:00+08:00
```

### 1.4 金额格式

API 金额统一使用 `amount_cent`，单位为分，类型为正整数。

前端负责将输入框中的元转换为分；后端只接收和校验整数分。

示例：

```text
12.35 元 -> amount_cent = 1235
```

## 2. 账单变更 Module

账单变更 Module 覆盖新增账单、编辑账单、删除账单三个写操作。

该 Module 的 Interface 只暴露账单变更意图；以下行为属于后端 Implementation：

- 校验账簿 `write` 权限。
- 校验分类与收支类型匹配。
- 在数据库事务内更新 `bills`。
- 同步更新账簿、家庭、分类统计。
- 必要时重算 `ledgers.last_recorded_at`。
- 写入关键操作日志。

### 2.1 账单展示模型

新增和编辑账单成功后返回完整账单展示模型，便于前端直接刷新来源页或详情页，避免额外查询。
其中 `creator` 固定返回账单创建时写入的快照值，不返回实时用户资料。
V1.0 的新增和编辑成功响应不返回账簿、家庭、分类统计等副作用结果。
V1.0 的新增和编辑成功响应不返回 `source_type` 或 `source_payload_json`。

```json
{
  "id": "bill_001",
  "household_id": "hh_001",
  "ledger_id": "ld_001",
  "bill_type": "expense",
  "category": {
    "id": "cat_food",
    "name": "餐饮",
    "icon_key": "food",
    "color": "#F59E0B"
  },
  "channel": {
    "id": "ch_wechat",
    "name": "微信",
    "icon_key": "wechat"
  },
  "amount_cent": 1235,
  "occurred_at": "2026-07-03T10:30:00+08:00",
  "remark": "午餐",
  "creator": {
    "user_id": "user_001",
    "nickname": "小明",
    "avatar_url": "https://example.com/avatar.png"
  },
  "created_at": "2026-07-03T10:31:00+08:00",
  "updated_at": "2026-07-03T10:31:00+08:00"
}
```

字段说明：

| 字段            | 类型            | 必返 | 说明                        |
| ------------- | ------------- | -- | ------------------------- |
| id            | string        | 是  | 账单 ID                     |
| household\_id | string        | 是  | 家庭 ID                     |
| ledger\_id    | string        | 是  | 账簿 ID                     |
| bill\_type    | string        | 是  | `income` / `expense`      |
| category      | object        | 是  | 系统预设分类展示信息                |
| channel       | object / null | 是  | 系统预设支付渠道展示信息；未填写时为 `null` |
| amount\_cent  | integer       | 是  | 金额，单位分，正整数                |
| occurred\_at  | string        | 是  | 交易时间                      |
| remark        | string / null | 是  | 备注，最大 200 字符              |
| creator       | object        | 是  | 创建人快照展示信息                 |
| created\_at   | string        | 是  | 创建时间                      |
| updated\_at   | string        | 是  | 更新时间                      |

`occurred_at` 规则：

- 请求接受合法的 ISO 8601 时间字符串和时区信息。
- 响应统一返回标准时区格式，不保留客户端原始时区表达。
- V1.0 不返回 `display_time` 或其他格式化时间文案字段。

`creator` 规则：

- `creator.nickname` 返回 `creator_nickname_snapshot`。
- `creator.avatar_url` 返回 `creator_avatar_snapshot` 对应的可展示地址或降级后的默认头像地址。
- 用户后续修改昵称、头像，或成员退出家庭后，不影响已返回和后续查询到的历史账单创建人展示值。
- V1.0 不返回 `is_deleted`、`is_left_household` 或其他创建人当前状态字段。

`channel` 规则：

- 请求中的 `channel_id = null` 时，响应固定返回 `channel: null`。
- 不返回空对象 `{}`。
- 不额外返回 `has_channel` 之类的布尔字段。
- V1.0 不返回 `channel_or_remark` 或其他渠道/备注 fallback 展示字段。

`category` 规则：

- 响应固定返回完整展示对象：`id`、`name`、`icon_key`、`color`。
- 不退化为单独返回 `category_id`。
- 不额外返回分类 `type`，收支语义由顶层 `bill_type` 表达。

`ledger` 规则：

- V1.0 的新增和编辑成功响应只返回 `ledger_id`。
- 不返回完整账簿展示对象，例如 `{ id, name }`。

金额展示规则：

- V1.0 的新增和编辑成功响应只返回 `amount_cent`。
- 不返回格式化金额字符串，例如 `amount_text`。
- `remark = null` 时不额外返回 `has_remark` 之类的布尔字段。
- V1.0 不返回 `display_amount`、正负号前缀或其他格式化金额展示字段。

`bill_type` 规则：

- V1.0 的新增和编辑成功响应继续返回原始枚举值：`income` / `expense`。
- 不返回本地化展示文案，例如“收入” / “支出”。

币种规则：

- V1.0 的新增和编辑成功响应不返回 `currency`。
- 金额币种默认由 V1.0 规则约定为人民币。

并发控制字段规则：

- V1.0 的新增、编辑成功响应不返回 `version`、`etag` 或其他并发控制字段。
- V1.0 不在账单变更 Module 的 Interface 中引入乐观锁约定。

统计副作用规则：

- V1.0 的新增、编辑成功响应不返回新的汇总值、统计差异或“统计已更新”之类的副作用结果。
- 首页、账簿详情、统计页需要的汇总数据应通过各自查询接口读取。

### 2.2 新增账单

```text
POST /api/bills
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "ledger_id": "ld_001",
  "bill_type": "expense",
  "category_id": "cat_food",
  "channel_id": "ch_wechat",
  "amount_cent": 1235,
  "occurred_at": "2026-07-03T10:30:00+08:00",
  "remark": "午餐"
}
```

请求字段：

| 字段            | 类型            | 必填 | 说明                         |
| ------------- | ------------- | -- | -------------------------- |
| household\_id | string        | 是  | 当前家庭 ID，用于校验账簿归属           |
| ledger\_id    | string        | 是  | 账簿 ID                      |
| bill\_type    | string        | 是  | `income` / `expense`       |
| category\_id  | string        | 是  | 系统预设分类 ID，必须匹配 `bill_type` |
| channel\_id   | string / null | 是  | 系统预设支付渠道 ID；无支付渠道时传 `null` |
| amount\_cent  | integer       | 是  | 金额，单位分，必须大于 0              |
| occurred\_at  | string        | 是  | 交易时间；后端不自动补时间              |
| remark        | string / null | 是  | 备注，最大 200 字符；无备注时传 `null`  |

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "bill_001",
    "household_id": "hh_001",
    "ledger_id": "ld_001",
    "bill_type": "expense",
    "category": {
      "id": "cat_food",
      "name": "餐饮",
      "icon_key": "food",
      "color": "#F59E0B"
    },
    "channel": {
      "id": "ch_wechat",
      "name": "微信",
      "icon_key": "wechat"
    },
    "amount_cent": 1235,
    "occurred_at": "2026-07-03T10:30:00+08:00",
    "remark": "午餐",
    "creator": {
      "user_id": "user_001",
      "nickname": "小明",
      "avatar_url": "https://example.com/avatar.png"
    },
    "created_at": "2026-07-03T10:31:00+08:00",
    "updated_at": "2026-07-03T10:31:00+08:00"
  }
}
```

Interface 规则：

- 新增账单必须显式传入 `household_id`，后端用它校验当前家庭上下文和账簿归属。
- V1.0 的新增账单接口不支持 `Idempotency-Key` 或其他幂等键。
- 用户必须对 `ledger_id` 对应账簿拥有 `write` 权限。
- `ledger_id` 必须属于 `household_id`。
- `household_id` 不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `ledger_id` 不存在、不可访问，或属于其他家庭而不属于当前 `household_id` 时返回 `LEDGER_NOT_FOUND`。
- `amount_cent` 必须是正整数，否则返回 `INVALID_AMOUNT`。
- `category_id` 必须存在、启用，且分类类型必须匹配 `bill_type`；否则返回 `CATEGORY_TYPE_MISMATCH`。
- `channel_id` 必须显式传入，可为 `null`；非 `null` 时必须存在且启用，否则返回 `INVALID_ARGUMENT`，并在 `message` 中说明 `channel_id` 不存在或不可用。
- `remark` 必须显式传入，可为 `null`；非 `null` 时最大 200 字符，否则返回 `INVALID_ARGUMENT`，并在 `message` 中说明 `remark` 超过 200 字符。
- 缺少任一请求字段时返回 `INVALID_ARGUMENT`，并在 `message` 中说明缺失字段。
- `occurred_at` 必填，前端默认当前时间，后端不自动补时间；为空或格式不合法时返回 `INVALID_ARGUMENT`，并在 `message` 中说明 `occurred_at` 不合法。
- V1.0 不支持客户端传入 `source_type` 或 `source_payload_json`；新增账单时由后端固定写入 `source_type = manual`。
- 在不支持幂等键的前提下，完全相同内容的重复提交允许落成两条独立账单，不做隐式去重。
- 创建人和创建人快照由后端根据登录态生成，前端不能提交。
- 禁止提交只读字段：`creator_user_id`、`creator_nickname_snapshot`、`creator_avatar_snapshot`、`source_type`、`source_payload_json`、`created_at`、`updated_at`。
- 提交只读字段时返回 `READONLY_FIELD_SUBMITTED`。

### 2.3 编辑账单

```text
POST /api/bills/update
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "bill_id": "bill_001",
  "household_id": "hh_001",
  "ledger_id": "ld_002",
  "bill_type": "income",
  "category_id": "cat_salary",
  "channel_id": null,
  "amount_cent": 500000,
  "occurred_at": "2026-07-03T09:00:00+08:00",
  "remark": "工资"
}
```

请求字段：

| 字段            | 类型            | 必填 | 说明                         |
| ------------- | ------------- | -- | -------------------------- |
| bill\_id      | string        | 是  | 目标账单 ID                    |
| household\_id | string        | 是  | 当前家庭 ID，用于校验原账单和新账簿归属      |
| ledger\_id    | string        | 是  | 新账簿 ID；未迁移账簿时传原账簿 ID       |
| bill\_type    | string        | 是  | `income` / `expense`       |
| category\_id  | string        | 是  | 系统预设分类 ID，必须匹配 `bill_type` |
| channel\_id   | string / null | 是  | 系统预设支付渠道 ID；无支付渠道时传 `null` |
| amount\_cent  | integer       | 是  | 金额，单位分，必须大于 0              |
| occurred\_at  | string        | 是  | 交易时间                       |
| remark        | string / null | 是  | 备注，最大 200 字符；无备注时传 `null`  |

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "bill_001",
    "household_id": "hh_001",
    "ledger_id": "ld_002",
    "bill_type": "income",
    "category": {
      "id": "cat_salary",
      "name": "工资",
      "icon_key": "salary",
      "color": "#22C55E"
    },
    "channel": null,
    "amount_cent": 500000,
    "occurred_at": "2026-07-03T09:00:00+08:00",
    "remark": "工资",
    "creator": {
      "user_id": "user_001",
      "nickname": "小明",
      "avatar_url": "https://example.com/avatar.png"
    },
    "created_at": "2026-07-01T20:00:00+08:00",
    "updated_at": "2026-07-03T10:31:00+08:00"
  }
}
```

Interface 规则：

- `bill_id` 必须存在。
- 编辑账单要求提交完整请求字段，不支持只提交发生变化的字段。
- 完整请求字段包括：`bill_id`、`household_id`、`ledger_id`、`bill_type`、`category_id`、`channel_id`、`amount_cent`、`occurred_at`、`remark`。
- `household_id` 是上下文校验字段，不表示允许修改账单所属家庭。
- V1.0 的编辑账单接口不支持乐观锁，不比较 `updated_at`，也不做并发冲突检测。
- `channel_id` 和 `remark` 可以传 `null` 表示清空。
- 缺少任一完整请求字段时返回 `INVALID_ARGUMENT`，并在 `message` 中说明缺失字段。
- `household_id` 必须与原账单所属家庭一致，否则返回 `INVALID_ARGUMENT`。
- 用户必须对原账簿拥有 `write` 权限，否则返回 `PERMISSION_DENIED`。
- 允许修改 `ledger_id`，把账单迁移到其他账簿，但仅限同一 `household_id` 内。
- 如果 `ledger_id` 变更，用户还必须对新账簿拥有 `write` 权限，否则返回 `PERMISSION_DENIED`。
- 新 `ledger_id` 必须属于 `household_id`，否则返回 `LEDGER_NOT_FOUND`。
- `amount_cent` 必须是正整数，否则返回 `INVALID_AMOUNT`。
- `category_id` 不存在、未启用，或与 `bill_type` 不匹配时返回 `CATEGORY_TYPE_MISMATCH`。
- `channel_id` 非 `null` 时必须存在且启用，否则返回 `INVALID_ARGUMENT`，并在 `message` 中说明 `channel_id` 不存在或不可用。
- `remark` 非 `null` 时最大 200 字符，否则返回 `INVALID_ARGUMENT`，并在 `message` 中说明 `remark` 超过 200 字符。
- `occurred_at` 为空或格式不合法时返回 `INVALID_ARGUMENT`，并在 `message` 中说明 `occurred_at` 不合法。
- V1.0 不支持编辑 `source_type` 或 `source_payload_json`；账单编辑不改变已有 `source_type`。
- 如果请求内容与原账单完全一致，仍返回成功，不返回“无变更”错误。
- 允许修改 `ledger_id`、`bill_type`、`category_id`、`channel_id`、`amount_cent`、`occurred_at`、`remark`。
- 禁止提交只读字段：`creator_user_id`、`creator_nickname_snapshot`、`creator_avatar_snapshot`、`source_type`、`source_payload_json`、`created_at`、`updated_at`。
- 提交只读字段时返回 `READONLY_FIELD_SUBMITTED`。
- Implementation 必须先扣减旧账单影响，再增加新账单影响。
- 如果交易时间或账簿变化，Implementation 必须重算受影响账簿的 `last_recorded_at`。

### 2.4 删除账单

```text
POST /api/bills/delete
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "bill_id": "bill_001",
  "household_id": "hh_001"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "bill_id": "bill_001"
  }
}
```

Interface 规则：

- `bill_id` 必须存在。
- 删除账单必须在请求体中显式传入 `household_id`，用于校验当前家庭上下文。
- V1.0 的删除账单接口不支持条件删除，不接受 `if-unmodified-since`、`updated_at` 或其他并发前置条件。
- 缺少 `household_id` 时返回 `INVALID_ARGUMENT`，并在 `message` 中说明缺失字段。
- `household_id` 不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `bill_id` 在当前 `household_id` 上下文下不存在或不可访问时返回 `BILL_NOT_FOUND`。
- 同一账单被成功删除后，后续重复删除请求返回 `BILL_NOT_FOUND`。
- 用户必须对账单所属账簿拥有 `write` 权限，否则返回 `PERMISSION_DENIED`。
- 删除账单是物理删除，不支持恢复。
- 删除成功后响应只返回 `bill_id`，不返回账单详情。
- 删除成功后不返回原账单的 `ledger_id`。
- 删除成功后不返回 `deleted_at` 或其他删除元数据。
- 删除成功后不返回 `refresh_hint` 或其他来源页刷新信号。
- Implementation 必须在事务内删除 `bills` 并扣减账簿、家庭、分类统计。
- 如果删除的是账簿最后一条账单，Implementation 必须重算 `ledgers.last_recorded_at`。
- 删除账单需要写入 `operation_logs`。

### 2.5 错误码

| 错误码                        | 文案           | 触发场景                                                                                                |
| -------------------------- | ------------ | --------------------------------------------------------------------------------------------------- |
| INVALID\_ARGUMENT          | 请求参数不合法      | 缺少必填字段、字段类型错误、字段格式不合法，或 `channel_id` 不存在/不可用、`remark` 超过 200 字符、`occurred_at` 不合法；`message` 需说明具体字段 |
| PERMISSION\_DENIED         | 权限不足         | 用户没有账簿 `write` 权限                                                                                   |
| HOUSEHOLD\_NOT\_FOUND      | 家庭不存在        | `household_id` 不存在或不可访问                                                                             |
| LEDGER\_NOT\_FOUND         | 账簿不存在        | `ledger_id` 不存在、不属于家庭或不可访问                                                                          |
| BILL\_NOT\_FOUND           | 账单不存在        | `bill_id` 不存在，或在当前家庭上下文下不可访问                                                                        |
| INVALID\_AMOUNT            | 金额必须大于 0     | `amount_cent` 为空、非整数或小于等于 0                                                                         |
| CATEGORY\_TYPE\_MISMATCH   | 账单分类与收支类型不匹配 | `category_id` 不存在、未启用，或与 `bill_type` 不匹配                                                            |
| READONLY\_FIELD\_SUBMITTED | 请求包含不可修改字段   | 新增或编辑账单时提交只读字段                                                                                      |

## 3. 账单查询 Module

账单查询 Module 覆盖账单列表、筛选、排序、游标分页三个读操作场景。

该 Module 的 Interface 只暴露查询意图；以下行为属于后端 Implementation：

- 按当前家庭上下文过滤用户可访问的账簿集合。
- 在家庭维度和账簿维度之间切换查询范围。
- 执行筛选条件组合、排序稳定性和游标编解码。

### 3.1 查询端点

V1.0 先使用统一端点：

```text
POST /api/bills/query
```

HTTP 状态码：

```text
200 OK
```

请求体使用 JSON，对数组类筛选条件采用原生数组表达。

请求示例：

```json
{
  "household_id": "hh_001",
  "ledger_ids": [],
  "filters": {
    "bill_type": "all",
    "category_ids": [],
    "channel_ids": [],
    "stat_months": [],
    "amount_range": {}
  },
  "sort": "time_desc",
  "cursor": null,
  "limit": 20
}
```

Interface 规则：

- 家庭维度和账簿维度共用一个查询端点，不拆成多个列表端点。
- 查询范围通过请求体中的 `household_id + 可选 ledger_ids` 表达，不引入额外的 `scope` 参数。
- `household_id` 是必填参数。
- `household_id` 为空字符串、格式不合法或类型非法时，统一返回 `INVALID_ARGUMENT`。
- `HOUSEHOLD_NOT_FOUND` 只用于 `household_id` 格式合法但目标家庭不存在或当前用户不可访问的场景。
- 顶层字段只保留查询骨架：`household_id`、`ledger_ids`、`filters`、`sort`、`cursor`、`limit`。
- 具体筛选条件统一放入 `filters` 对象。
- `filters` 必须显式传入对象；没有筛选条件时也必须传入完整子字段结构。
- 服务端不对缺失的筛选字段做默认补全；请求缺少约定字段时统一返回 `INVALID_ARGUMENT`。
- 请求中出现未定义的顶层字段或未定义的 `filters` 子字段时，不做透传忽略，统一返回 `INVALID_ARGUMENT`。
- `filters.bill_type` 使用单值枚举：`income` | `expense` | `all`。
- `filters.bill_type` 必须显式传入；没有收支筛选时传 `all`。
- `filters.bill_type` 缺失、为空字符串、类型非法，或不是 `income`、`expense`、`all` 之一时，统一返回 `INVALID_ARGUMENT`。
- `filters.category_ids` 定义为数组。
- `filters.channel_ids` 定义为数组。
- `filters.stat_months` 定义为数组字段，用于月份筛选。
- `filters.category_ids`、`filters.channel_ids`、`filters.stat_months` 都必须显式传入数组；没有选择时传 `[]`。
- `filters.category_ids`、`filters.channel_ids` 中出现不存在或已禁用的 ID 时返回 `INVALID_ARGUMENT`。
- `filters.category_ids`、`filters.channel_ids` 中任一元素为空字符串、格式不合法或类型非法时，统一返回 `INVALID_ARGUMENT`。
- `filters.category_ids`、`filters.channel_ids` 中出现重复值时，不做自动去重，统一返回 `INVALID_ARGUMENT`。
- `filters.stat_months` 的元素格式固定为 `YYYY-MM`。
- `filters.stat_months` 传多个值时按并集处理。
- `filters.stat_months` 中出现重复值时，不做自动去重，统一返回 `INVALID_ARGUMENT`。
- `filters.stat_months` 中任一元素为空字符串、类型非法，或不符合合法 `YYYY-MM` 格式时，统一返回 `INVALID_ARGUMENT`。
- `filters.stat_months` 中出现非法格式值时返回 `INVALID_ARGUMENT`。
- `filters.amount_range` 定义为对象，包含 `min_amount_cent` 和 `max_amount_cent`。
- `filters.amount_range` 传入数组、字符串、数字或其他非对象类型时，统一返回 `INVALID_ARGUMENT`。
- `filters.amount_range` 中出现除 `min_amount_cent`、`max_amount_cent` 之外的未定义字段时，统一返回 `INVALID_ARGUMENT`。
- 没有金额筛选时，`filters.amount_range` 也必须显式传入 `{}`。
- `filters.amount_range.min_amount_cent` 和 `filters.amount_range.max_amount_cent` 都允许为空，表示只设单侧边界。
- `filters.amount_range.min_amount_cent` 和 `filters.amount_range.max_amount_cent` 如传入值，必须为大于 `0` 的整数分；不允许传 `0`。
- `filters.amount_range.min_amount_cent` 和 `filters.amount_range.max_amount_cent` 传入小数、字符串、负数或其他非法类型时，统一返回 `INVALID_ARGUMENT`。
- `filters.amount_range.min_amount_cent > filters.amount_range.max_amount_cent` 时返回 `INVALID_ARGUMENT`。
- `sort` 继续沿用四个枚举值：`time_desc`、`time_asc`、`amount_desc`、`amount_asc`。
- `sort` 必须显式传入；字段缺失时统一返回 `INVALID_ARGUMENT`。
- `sort` 传入未定义枚举值或非法类型时，统一返回 `INVALID_ARGUMENT`。
- V1.0 约定默认排序值为 `time_desc`，但调用方仍需显式传入，不依赖服务端缺省补全。
- `time_desc` 和 `time_asc` 使用 `occurred_at + id` 作为稳定排序键。
- `amount_desc` 和 `amount_asc` 使用 `amount_cent + id` 作为稳定排序键。
- `limit` 必须显式传入；字段缺失时统一返回 `INVALID_ARGUMENT`。
- V1.0 协议默认页大小定义为 `20`，但调用方仍需显式传入，不依赖服务端缺省补全。
- `limit` 如显式传入，必须为大于 `0` 的整数；传 `0`、负数、小数、字符串或其他非法类型时，统一返回 `INVALID_ARGUMENT`。
- `limit` 最大值为 `100`，超过上限时返回 `INVALID_ARGUMENT`。
- `cursor` 必须显式传入；首次查询传 `null`，后续翻页传不透明字符串，缺失时统一返回 `INVALID_ARGUMENT`。
- `cursor` 允许为 `null`，表示首次查询第一页。
- `cursor` 定义为不透明字符串，调用方不应解析或拼装其内部结构。
- `cursor` 与本次请求的 `sort`、`ledger_ids`、`filters` 不匹配，或 `cursor` 本身损坏、不可解析时，统一返回 `INVALID_ARGUMENT`。
- 后端不对非法 `cursor` 做自动纠偏，也不降级为重新查询第一页。
- V1.0 不支持 `with_total` 或其他总数开关。
- 筛选面板中的“查看 {n} 条结果”之类即时数量，不由本查询接口负责。
- V1.0 不支持按 `remark` 关键字搜索。
- 查询错误响应继续只保留 `code + message`，不额外返回“无权限账簿列表”“非法筛选项列表”或其他结构化错误明细。
- `ledger_ids` 必须显式传入数组。
- `ledger_ids` 中任一元素为空字符串、格式不合法或类型非法时，统一返回 `INVALID_ARGUMENT`。
- `ledger_ids` 中出现重复值时，不做自动去重，统一返回 `INVALID_ARGUMENT`。
- `household_id + ledger_ids = []` 时表示当前家庭下的全部可访问账单列表。
- `ledger_ids` 只在当前 `household_id` 上下文下使用。
- `ledger_ids` 只有一个元素时表示当前家庭下指定单账簿的账单列表。
- `ledger_ids` 有多个元素时表示当前家庭下多账簿筛选。
- `ledger_ids` 中任一账簿不属于当前 `household_id`，或当前用户对其没有 `read` 权限时，返回 `LEDGER_NOT_FOUND`。
- 家庭维度查询时，后端自动过滤为当前用户有权限访问的账簿账单。

### 3.2 查询请求字段

顶层请求字段：

| 字段            | 类型            | 必填 | 说明                                                           |
| ------------- | ------------- | -- | ------------------------------------------------------------ |
| household\_id | string        | 是  | 家庭 ID；为空字符串、格式不合法或类型非法时返回 `INVALID_ARGUMENT`                 |
| ledger\_ids   | string\[]     | 是  | 账簿 ID 数组；`[]` 表示当前家庭下全部可访问账簿                                 |
| filters       | object        | 是  | 查询筛选对象；必须显式传入完整结构                                            |
| sort          | string        | 是  | 排序枚举：`time_desc` / `time_asc` / `amount_desc` / `amount_asc` |
| cursor        | string / null | 是  | 游标；首次查询传 `null`，后续翻页传不透明字符串                                  |
| limit         | integer       | 是  | 页大小；必须为大于 `0` 的整数，最大 `100`                                   |

`filters` 字段：

| 字段            | 类型        | 必填 | 说明                                                 |
| ------------- | --------- | -- | -------------------------------------------------- |
| bill\_type    | string    | 是  | 收支类型：`income` / `expense` / `all`；无筛选时也必须显式传 `all` |
| category\_ids | string\[] | 是  | 分类 ID 数组；无筛选时传 `[]`                                |
| channel\_ids  | string\[] | 是  | 渠道 ID 数组；无筛选时传 `[]`                                |
| stat\_months  | string\[] | 是  | 月份数组；元素格式固定为 `YYYY-MM`；无筛选时传 `[]`                  |
| amount\_range | object    | 是  | 金额区间对象；无筛选时传 `{}`                                  |

`filters.amount_range` 字段：

| 字段                | 类型             | 必填 | 说明                       |
| ----------------- | -------------- | -- | ------------------------ |
| min\_amount\_cent | integer / null | 否  | 最小金额边界；传值时必须为大于 `0` 的整数分 |
| max\_amount\_cent | integer / null | 否  | 最大金额边界；传值时必须为大于 `0` 的整数分 |

请求约束补充：

- 顶层字段 `household_id`、`ledger_ids`、`filters`、`sort`、`cursor`、`limit` 都必须显式传入。
- `filters` 内部字段 `bill_type`、`category_ids`、`channel_ids`、`stat_months`、`amount_range` 也必须显式传入。
- `ledger_ids`、`category_ids`、`channel_ids`、`stat_months` 中出现重复值时，统一返回 `INVALID_ARGUMENT`。
- `ledger_ids` 中任一元素合法但不属于当前家庭，或当前用户对其没有 `read` 权限时，返回 `LEDGER_NOT_FOUND`。
- `category_ids`、`channel_ids` 中任一元素合法但不存在或已禁用时，统一返回 `INVALID_ARGUMENT`。
- `stat_months` 传多个值时按并集处理。
- `amount_range.min_amount_cent > amount_range.max_amount_cent` 时，统一返回 `INVALID_ARGUMENT`。
- 请求中出现未定义的顶层字段、未定义的 `filters` 子字段，或未定义的 `amount_range` 子字段时，统一返回 `INVALID_ARGUMENT`。

成功响应结构继续沿用：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "items": [],
    "next_cursor": "xxx",
    "has_more": true
  }
}
```

### 3.3 查询成功响应字段

响应字段：

| 字段                | 类型            | 必返 | 说明                           |
| ----------------- | ------------- | -- | ---------------------------- |
| data.items        | array         | 是  | 账单列表；单条元素直接复用第 2.1 节“账单展示模型” |
| data.next\_cursor | string / null | 是  | 下一页游标；无更多数据时返回 `null`        |
| data.has\_more    | boolean       | 是  | 是否还有更多数据                     |

- 查询成功响应中的 `data` 只返回 `items`、`next_cursor`、`has_more` 三个字段。
- V1.0 不在 `data` 顶层追加其他字段。
- 没有更多数据时，`next_cursor` 返回 `null`。
- `has_more` 继续保留，不仅依赖 `next_cursor` 推断。
- 查询命中 0 条数据时仍返回成功响应，不返回 `BILL_NOT_FOUND`。
- 空结果时返回 `items = []`、`next_cursor = null`、`has_more = false`。
- `items` 中的单条账单结构直接复用第 2.1 节“账单展示模型”。
- 查询结果中的 `remark` 继续返回原始值：有值时返回字符串，没值时返回 `null`。
- 不额外派生 `has_remark`、`remark_text` 或其他备注展示辅助字段。
- V1.0 不返回 `can_edit`、`can_delete` 或其他“是否可操作”的权限派生状态字段。
- V1.0 不返回 `is_deleted`、`is_archived`、`is_abnormal` 或其他业务状态字段。
- V1.0 不返回 `applied_filters`、`normalized_filters` 或其他筛选条件回显字段。
- V1.0 不返回 `page_income_total_cent`、`page_expense_total_cent` 或其他本页汇总字段。
- V1.0 不返回 `group_title`、`group_key`、`group_items` 或其他分组结果字段；查询结果保持扁平 `items` 结构。
- V1.0 不返回 `applied_sort`、`sort_key` 或其他排序回显字段。
- V1.0 不返回 `snapshot_version`、`server_time`、`sync_token` 或其他数据版本/同步协议字段。
- V1.0 不返回 `from_ledger_name`、`household_name`、`category_path_text` 或其他来源页提示/展示文案字段。

### 3.4 错误码

| 错误码                   | 文案      | 触发场景                                                                                                                                                                  |
| --------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| INVALID\_ARGUMENT     | 请求参数不合法 | 顶层字段缺失；`filters` 或 `amount_range` 结构不合法；存在未定义字段；`sort` / `limit` / `cursor` / `bill_type` 非法；ID 数组元素非法或重复；`stat_months` 非法；`amount_range` 非法；`cursor` 与本次查询条件不匹配或不可解析 |
| HOUSEHOLD\_NOT\_FOUND | 家庭不存在   | `household_id` 格式合法，但目标家庭不存在或当前用户不可访问                                                                                                                                 |
| LEDGER\_NOT\_FOUND    | 账簿不存在   | `ledger_ids` 中存在格式合法，但不属于当前家庭或当前用户无 `read` 权限的账簿                                                                                                                      |
| PERMISSION\_DENIED    | 权限不足    | 保留给其他需要显式返回权限错误的查询场景；当前列表查询中优先按既定规则返回 `HOUSEHOLD_NOT_FOUND` 或 `LEDGER_NOT_FOUND`                                                                                      |

## 4. 家庭 Module

家庭 Module 覆盖创建家庭、家庭列表、切换当前家庭、家庭详情、编辑家庭名称、邀请成员、接受邀请、成员设置、退出家庭、删除家庭等能力。

该 Module 的 Interface 只暴露家庭管理意图；以下行为属于后端 Implementation：

- 校验当前用户是否属于目标家庭。
- 校验管理员权限、最后一个管理员约束和成员存在性。
- 在数据库事务内更新 `households`、`household_members`、`household_invites`、`user_settings`。
- 删除家庭时级联删除账簿、账单、授权和统计汇总。
- 写入关键操作日志。

术语说明：

- `user_settings`：用户全局设置；V1.0 目前用于保存 `current_household_id`。
- `user_household_settings`：用户在某个家庭维度下的个人设置记录；一条记录对应 `user_id + household_id`，V1.0 用于保存该家庭下的 `current_ledger_id` 和 `last_selected_at`。
- `user_household_settings.last_selected_at`：该家庭最近一次成为当前家庭的时间；用于在删除当前家庭或退出当前家庭后，从剩余可访问家庭中选出“最近一次成为当前家庭的那个”。
- 当后端把 `user_settings.current_household_id` 切换到某个家庭时，如果缺少对应的 `user_household_settings` 记录，后端统一补建一条，并初始化 `current_ledger_id = null`；只要本次操作成功把该家庭设为当前家庭，不论它此前是否已经是当前家庭，后端都更新该记录的 `last_selected_at`。

### 4.1 家庭列表项展示模型

家庭列表接口返回家庭列表项展示模型，供首页左上角切换家庭面板和“家庭共享”列表页直接消费。

```json
{
  "id": "hh_001",
  "name": "三口之家",
  "role": "admin",
  "member_count": 3,
  "ledger_count": 2,
  "is_current": true,
  "created_at": "2026-07-01T10:00:00+08:00",
  "updated_at": "2026-07-03T09:30:00+08:00"
}
```

字段说明：

| 字段            | 类型      | 必返 | 说明                              |
| ------------- | ------- | -- | ------------------------------- |
| id            | string  | 是  | 家庭 ID                           |
| name          | string  | 是  | 家庭名称                            |
| role          | string  | 是  | 当前用户在该家庭中的角色：`admin` / `normal` |
| member\_count | integer | 是  | 家庭成员数                           |
| ledger\_count | integer | 是  | 家庭下账簿数                          |
| is\_current   | boolean | 是  | 是否为当前选中家庭                       |
| created\_at   | string  | 是  | 家庭创建时间                          |
| updated\_at   | string  | 是  | 家庭更新时间                          |

展示规则：

- `role` 继续返回原始枚举值：`admin` / `normal`。
- V1.0 不返回“管理员”“普通成员”等本地化展示文案。
- V1.0 不返回 `owner_user_id`、`can_manage`、`display_badge_text` 或其他派生展示字段。
- `member_count` 和 `ledger_count` 返回整数值；即使为 `0` 也返回 `0`，不返回 `null`。

### 4.2 家庭详情展示模型

家庭详情接口返回家庭基础信息、成员列表和共享账簿列表，供“家庭详情与设置”页直接消费。

```json
{
  "id": "hh_001",
  "name": "三口之家",
  "role": "admin",
  "member_count": 3,
  "ledger_count": 2,
  "members": [
    {
      "user_id": "user_001",
      "nickname": "小明",
      "avatar_url": "https://example.com/avatar.png",
      "role": "admin",
      "remark": "爸爸",
      "joined_at": "2026-07-01T10:00:00+08:00"
    }
  ],
  "shared_ledgers": [
    {
      "id": "ld_001",
      "name": "日常账簿"
    }
  ],
  "created_at": "2026-07-01T10:00:00+08:00",
  "updated_at": "2026-07-03T09:30:00+08:00"
}
```

字段说明：

| 字段              | 类型      | 必返 | 说明                              |
| --------------- | ------- | -- | ------------------------------- |
| id              | string  | 是  | 家庭 ID                           |
| name            | string  | 是  | 家庭名称                            |
| role            | string  | 是  | 当前用户在该家庭中的角色：`admin` / `normal` |
| member\_count   | integer | 是  | 家庭成员数                           |
| ledger\_count   | integer | 是  | 家庭下账簿数                          |
| members         | array   | 是  | 家庭成员列表                          |
| shared\_ledgers | array   | 是  | 家庭内共享账簿列表                       |
| created\_at     | string  | 是  | 家庭创建时间                          |
| updated\_at     | string  | 是  | 家庭更新时间                          |

`members` 元素字段：

| 字段          | 类型            | 必返 | 说明                      |
| ----------- | ------------- | -- | ----------------------- |
| user\_id    | string        | 是  | 成员用户 ID                 |
| nickname    | string        | 是  | 成员昵称快照或当前昵称展示值          |
| avatar\_url | string        | 是  | 成员头像展示地址                |
| role        | string        | 是  | 家庭角色：`admin` / `normal` |
| remark      | string / null | 是  | 家庭内备注；无备注时返回 `null`     |
| joined\_at  | string        | 是  | 入家时间                    |

`shared_ledgers` 元素字段：

| 字段   | 类型     | 必返 | 说明    |
| ---- | ------ | -- | ----- |
| id   | string | 是  | 账簿 ID |
| name | string | 是  | 账簿名称  |

详情规则：

- `members` 返回当前家庭下全部成员，不额外分页。
- `shared_ledgers` 返回当前家庭下 `is_shared = true` 的账簿；没有共享账簿时返回 `[]`。
- V1.0 不返回 `members_count_text`、`ledger_count_text`、`is_self`、`can_edit_role`、`can_remove_member` 或其他派生权限字段。
- V1.0 不返回成员手机号、外部账号标识或其他隐私字段。
- `remark` 没有值时固定返回 `null`，不返回空字符串 `""` 作为“无备注”表达。

### 4.3 创建家庭

```text
POST /api/households/create
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "name": "温暖之家"
}
```

请求字段：

| 字段   | 类型     | 必填 | 说明                  |
| ---- | ------ | -- | ------------------- |
| name | string | 是  | 家庭名称，必填，最大 `20` 个字符 |

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "hh_003",
    "name": "温暖之家",
    "role": "admin",
    "member_count": 1,
    "ledger_count": 0,
    "is_current": true,
    "created_at": "2026-07-03T11:30:00+08:00",
    "updated_at": "2026-07-03T11:30:00+08:00"
  }
}
```

Interface 规则：

- 创建家庭要求完整请求字段：`name`。
- `name` 必须显式传入；字段缺失时返回 `INVALID_ARGUMENT`。
- `name` 必须为非空字符串，最大 `20` 个字符；为空、超长或类型非法时返回 `INVALID_ARGUMENT`。
- 家庭名称允许重名；V1.0 不做同用户内唯一校验，也不做全局唯一校验。
- 创建成功后，创建人自动成为该家庭的 `admin` 成员。
- 创建成功后，后端保存 `user_settings.current_household_id` 为新建家庭。
- 创建成功后，后端为该家庭补建对应的 `user_household_settings` 记录，并明确写入 `current_ledger_id = null` 和当前时间的 `last_selected_at`，不做任何隐式兜底选择。
- 创建成功后允许正常进入该家庭；家庭级数据查询继续按该家庭上下文正常返回，不因 `current_ledger_id = null` 视为异常。
- 创建家庭成功响应直接返回第 4.1 节“家庭列表项展示模型”。
- 创建家庭后不会自动创建默认账簿；`ledger_count` 初始返回 `0`。
- 创建家庭接口不接受首批成员列表、首批邀请列表或角色分配列表；邀请成员必须在创建成功后单独调用第 4.8 节“创建家庭邀请”接口。
- 所有依赖具体账簿的入口和查询由调用方显式传入 `ledger_id`；当 `ledger_id` 为空时，按“无账簿”规则返回空列表，不做隐式兜底选择。
- 创建 `households`、写入创建者的 `household_members`、切换 `user_settings.current_household_id`、补建 `user_household_settings` 必须处于同一个事务中；任一步失败都要整体回滚。
- V1.0 不在创建家庭接口中隐式创建默认账簿、默认分类视图或其他附属资源。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.4 家庭列表查询

```text
POST /api/households/query
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "cursor": null,
  "limit": 20
}
```

请求字段：

| 字段     | 类型            | 必填 | 说明                          |
| ------ | ------------- | -- | --------------------------- |
| cursor | string / null | 是  | 游标；首次查询传 `null`，后续翻页传不透明字符串 |
| limit  | integer       | 是  | 页大小；必须为大于 `0` 的整数，最大 `100`  |

Interface 规则：

- 家庭列表查询继续统一使用 `POST`，不引入 `GET /households`。
- 顶层字段只保留 `cursor`、`limit` 两个字段。
- `cursor` 和 `limit` 都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `cursor` 首次查询必须显式传 `null`；后续翻页传不透明字符串。
- `cursor` 损坏、不可解析时返回 `INVALID_ARGUMENT`。
- `limit` 必须为大于 `0` 的整数，最大 `100`。
- 家庭列表只返回当前登录用户所属家庭。
- V1.0 不支持角色筛选、名称搜索或其他筛选参数；请求中出现未定义字段时返回 `INVALID_ARGUMENT`。
- 查询结果使用稳定顺序；Implementation 至少保证同一游标链路下顺序稳定。

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "items": [
      {
        "id": "hh_001",
        "name": "三口之家",
        "role": "admin",
        "member_count": 3,
        "ledger_count": 2,
        "is_current": true,
        "created_at": "2026-07-01T10:00:00+08:00",
        "updated_at": "2026-07-03T09:30:00+08:00"
      }
    ],
    "next_cursor": null,
    "has_more": false
  }
}
```

查询成功响应字段：

| 字段                | 类型            | 必返 | 说明                            |
| ----------------- | ------------- | -- | ----------------------------- |
| data.items        | array         | 是  | 家庭列表；单条元素复用第 4.1 节“家庭列表项展示模型” |
| data.next\_cursor | string / null | 是  | 下一页游标；无更多数据时返回 `null`         |
| data.has\_more    | boolean       | 是  | 是否还有更多数据                      |

响应规则：

- 查询成功响应中的 `data` 只返回 `items`、`next_cursor`、`has_more` 三个字段。
- V1.0 不在 `data` 顶层追加 `total`、`current_household_id` 或其他字段。
- 空结果时返回 `items = []`、`next_cursor = null`、`has_more = false`。

### 4.5 切换当前家庭

```text
POST /api/households/switch
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_002"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_002"
  }
}
```

Interface 规则：

- 切换当前家庭必须显式传入 `household_id`。
- `household_id` 为空字符串、格式不合法或类型非法时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不属于该家庭时返回 `HOUSEHOLD_NOT_FOUND`。
- 切换成功后，后端保存 `user_settings.current_household_id`。
- 切换成功后，如果目标家庭缺少对应的 `user_household_settings` 记录，后端统一补建一条，并初始化 `current_ledger_id = null`。
- 切换成功后，后端更新目标家庭对应 `user_household_settings.last_selected_at` 为当前时间。
- 如果目标家庭下当前 `current_ledger_id` 为空，仍允许切换成功；家庭级数据查询继续正常返回。
- 所有依赖具体账簿的入口和查询由调用方显式传入 `ledger_id`；当 `ledger_id` 为空时，按“无账簿”规则返回空列表，不做隐式兜底选择。
- 如果请求目标家庭已经是当前家庭，仍返回成功，不返回“无变更”错误。
- 成功响应只返回 `household_id`，不额外返回家庭详情、首页统计或当前账簿信息。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.6 家庭详情查询

```text
POST /api/households/detail
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "hh_001",
    "name": "三口之家",
    "role": "admin",
    "member_count": 3,
    "ledger_count": 2,
    "members": [
      {
        "user_id": "user_001",
        "nickname": "小明",
        "avatar_url": "https://example.com/avatar.png",
        "role": "admin",
        "remark": "爸爸",
        "joined_at": "2026-07-01T10:00:00+08:00"
      }
    ],
    "shared_ledgers": [
      {
        "id": "ld_001",
        "name": "日常账簿"
      }
    ],
    "created_at": "2026-07-01T10:00:00+08:00",
    "updated_at": "2026-07-03T09:30:00+08:00"
  }
}
```

Interface 规则：

- 家庭详情查询必须显式传入 `household_id`。
- `household_id` 为空字符串、格式不合法或类型非法时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- 成功响应直接返回第 4.2 节“家庭详情展示模型”。
- V1.0 不返回额外的编辑态权限提示字段，由前端结合登录用户和 `role` 自行判断展示。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.7 编辑家庭名称

```text
POST /api/households/update_name
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "name": "温暖之家"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "hh_001",
    "name": "温暖之家",
    "updated_at": "2026-07-03T11:00:00+08:00"
  }
}
```

Interface 规则：

- 编辑家庭名称要求完整请求字段：`household_id`、`name`。
- `household_id` 和 `name` 都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- 只有家庭管理员可以编辑家庭名称，否则返回 `PERMISSION_DENIED`。
- `name` 必须为非空字符串，最大 `20` 个字符；为空、超长或类型非法时返回 `INVALID_ARGUMENT`。
- 家庭名称允许重名；V1.0 不做同用户内唯一校验，也不做全局唯一校验。
- 如果请求名称与原名称完全一致，仍返回成功，不返回“无变更”错误。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.8 创建家庭邀请

```text
POST /api/households/invites/create
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "role": "normal"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "invite_id": "inv_001",
    "invite_url": "https://example.com/invite?token=raw_invite_token",
    "role": "normal",
    "expires_at": "2026-07-10T11:00:00+08:00"
  }
}
```

Interface 规则：

- 创建家庭邀请要求完整请求字段：`household_id`、`role`。
- `household_id` 和 `role` 都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- 只有家庭管理员可以创建邀请，否则返回 `PERMISSION_DENIED`。
- `role` 只允许 `admin` 或 `normal`；传入其他值、空字符串或非法类型时返回 `INVALID_ARGUMENT`。
- 即使同一家庭、同一角色下仍存在 `pending` 邀请，V1.0 仍允许重复创建新的邀请，不要求复用旧邀请链接。
- 创建邀请接口不预判最终点击人是否已经在目标家庭中，也不因此做前置拦截；相关幂等处理统一由第 4.9 节“接受家庭邀请”接口承担。
- 成功响应返回可直接分享的 `invite_url`，不额外返回分享文案模板。
- 邀请链接为单次使用，创建后 7 天过期。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.9 接受家庭邀请

```text
POST /api/households/invites/accept
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "invite_token": "raw_invite_token"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001"
  }
}
```

Interface 规则：

- 接受邀请必须显式传入 `invite_token`。
- `invite_token` 为空字符串、格式不合法或类型非法时返回 `INVALID_ARGUMENT`。
- `invite_token` 对应邀请已过期时返回 `INVITE_EXPIRED`。
- `invite_token` 对应邀请已被其他用户成功使用时返回 `INVITE_USED`。
- 如果当前用户已在目标家庭中，仍返回成功，不消耗 token，也不修改角色，但仍将当前家庭切换为目标家庭。
- 即使当前邀请链接的 `role` 与该用户在家庭中的现有角色不同，V1.0 也继续保持“不修改角色”的规则；角色变更只能通过第 4.10 节“修改成员角色”接口完成。
- 如果当前用户已在目标家庭中，但缺少对应的 `user_household_settings` 记录，后端补建一条，并初始化 `current_ledger_id = null`。
- 接受邀请成功后，后端保存 `user_settings.current_household_id` 为目标家庭。
- 接受邀请成功后，如果该家庭缺少对应的 `user_household_settings` 记录，后端统一补建一条，并初始化 `current_ledger_id = null`，不做任何隐式账簿兜底选择。
- 接受邀请成功后，后端更新目标家庭对应 `user_household_settings.last_selected_at` 为当前时间。
- 如果当前用户原本不在目标家庭中，则一旦接受邀请成功，该 `invite_token` 立即失效并标记为已使用，后续不允许再次使用。
- 对于“原本不在目标家庭中”的接受邀请流程，写入 `household_members`、更新邀请状态、切换 `user_settings.current_household_id`、补建 `user_household_settings` 必须处于同一个事务中；任一步失败都要整体回滚。
- 成功响应只返回 `household_id`，不额外返回邀请详情或家庭详情。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.10 修改成员角色

```text
POST /api/households/members/update_role
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "member_user_id": "user_002",
  "role": "admin"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001",
    "member_user_id": "user_002",
    "role": "admin",
    "updated_at": "2026-07-03T11:10:00+08:00"
  }
}
```

Interface 规则：

- 修改成员角色要求完整请求字段：`household_id`、`member_user_id`、`role`。
- 三个字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `member_user_id` 格式合法但目标成员不属于该家庭时返回 `HOUSEHOLD_MEMBER_NOT_FOUND`。
- 只有家庭管理员可以修改成员角色，否则返回 `PERMISSION_DENIED`。
- `role` 只允许 `admin` 或 `normal`；传入其他值、空字符串或非法类型时返回 `INVALID_ARGUMENT`。
- 允许管理员修改自己的角色，包括把自己从 `admin` 改为 `normal`。
- 如果修改结果会导致家庭没有管理员，返回 `INVALID_ARGUMENT`，并在 `message` 中说明必须先完成管理员转让。
- 涉及管理员数量约束的校验和角色更新必须处于同一个事务中，避免并发修改导致家庭短暂无管理员状态。
- 如果请求角色与当前角色一致，仍返回成功，不返回“无变更”错误。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.11 修改成员备注

```text
POST /api/households/members/update_remark
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "member_user_id": "user_002",
  "remark": "妈妈"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001",
    "member_user_id": "user_002",
    "remark": "妈妈",
    "updated_at": "2026-07-03T11:12:00+08:00"
  }
}
```

Interface 规则：

- 修改成员备注要求完整请求字段：`household_id`、`member_user_id`、`remark`。
- 三个字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `member_user_id` 格式合法但目标成员不属于该家庭时返回 `HOUSEHOLD_MEMBER_NOT_FOUND`。
- 只有家庭管理员可以修改成员备注，否则返回 `PERMISSION_DENIED`。
- `remark` 允许传 `null` 表示清空备注。
- `remark` 非 `null` 时必须为最大 `12` 个字符的字符串；为空字符串、超长或类型非法时返回 `INVALID_ARGUMENT`。
- 如果请求备注与原备注完全一致，仍返回成功，不返回“无变更”错误。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.12 移出家庭成员

```text
POST /api/households/members/remove
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "member_user_id": "user_002"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001",
    "member_user_id": "user_002"
  }
}
```

Interface 规则：

- 移出成员要求完整请求字段：`household_id`、`member_user_id`。
- 两个字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `member_user_id` 格式合法但目标成员不属于该家庭时返回 `HOUSEHOLD_MEMBER_NOT_FOUND`。
- 只有家庭管理员可以移出成员，否则返回 `PERMISSION_DENIED`。
- 不允许通过该接口移出自己；当前用户离开家庭应使用第 4.13 节“退出家庭”接口。
- 允许管理员移出家庭内最后一个其他成员；移出后家庭可以退化为单人家庭。
- 如果移出目标会导致家庭没有管理员，返回 `INVALID_ARGUMENT`，并在 `message` 中说明必须先完成管理员转让。
- 涉及管理员数量约束的校验和成员移出动作必须处于同一个事务中，避免并发移出导致家庭短暂无管理员状态。
- 移出成员后，该成员失去后续家庭访问权限，但历史账单记录保留。
- 移出成员后，后端直接删除该成员在该家庭下整条 `user_household_settings` 记录，不保留悬空家庭设置。
- 如果移出成员后家庭退化为单人家庭，后端自动把该家庭下所有账簿的 `is_shared` 改为 `false`，并直接清理这些账簿上非 owner 的 `ledger_members` 关系以及相关 `authorization_grants`。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.13 退出家庭

```text
POST /api/households/leave
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001"
  }
}
```

Interface 规则：

- 退出家庭必须显式传入 `household_id`。
- `household_id` 为空字符串、格式不合法或类型非法时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- 如果当前用户是该家庭最后一个管理员且家庭中还有其他成员，返回 `INVALID_ARGUMENT`，并在 `message` 中说明必须先转让管理员权限。
- 涉及“最后一个管理员不能直接退出”的校验和退出动作必须处于同一个事务中，避免并发退出导致家庭短暂无管理员状态。
- 如果当前用户是家庭唯一成员，V1.0 不通过“退出家庭”隐式解散家庭；应调用第 4.14 节“删除家庭”接口。
- 退出成功后，该用户失去该家庭及其账簿的后续访问权限。
- 退出家庭后，后端直接删除当前用户在该家庭下整条 `user_household_settings` 记录，不保留悬空家庭设置。
- 如果退出的是当前家庭，后端自动把 `user_settings.current_household_id` 切到该用户剩余可访问家庭中 `user_household_settings.last_selected_at` 最新的那个；如果一个都没有，则置空。切换成功时，如果目标家庭缺少对应的 `user_household_settings` 记录，后端统一补建一条，并初始化 `current_ledger_id = null`，同时把 `last_selected_at` 更新为当前时间。
- 如果退出家庭后剩余家庭成员数变为 `1`，后端自动把该家庭下所有账簿的 `is_shared` 改为 `false`，并直接清理这些账簿上非 owner 的 `ledger_members` 关系以及相关 `authorization_grants`。
- 成功响应只返回 `household_id`，不额外返回跳转提示或新的当前家庭信息。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.14 删除家庭

```text
POST /api/households/delete
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001"
  }
}
```

Interface 规则：

- 删除家庭必须显式传入 `household_id`。
- `household_id` 为空字符串、格式不合法或类型非法时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- 只有家庭管理员可以删除家庭，否则返回 `PERMISSION_DENIED`。
- 删除家庭是物理删除，不支持恢复。
- 如果被删除的是当前家庭，后端自动把 `user_settings.current_household_id` 切到该用户剩余可访问家庭中 `user_household_settings.last_selected_at` 最新的那个；如果一个都没有，则置空。切换成功时，如果目标家庭缺少对应的 `user_household_settings` 记录，后端统一补建一条，并初始化 `current_ledger_id = null`，同时把 `last_selected_at` 更新为当前时间。
- 成功响应只返回 `household_id`，不返回删除影响明细。
- V1.0 不在删除家庭接口中引入 `confirm_token`、`confirm_phrase` 或其他接口级二次确认字段；二次确认由前端交互承担。
- 管理员权限校验、家庭存在性校验和真正删除动作必须处于同一个事务中，避免并发状态变化导致删除过程观察到不一致结果。
- Implementation 必须在事务内删除家庭、成员、邀请、账簿、账单、授权和统计汇总。
- 删除家庭时，后端直接删除所有成员在该家庭下的 `user_household_settings` 记录，不保留悬空家庭设置。
- 事务提交后异步删除账簿封面文件，不删除用户头像文件。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 4.15 错误码

| 错误码                           | 文案      | 触发场景                                                        |
| ----------------------------- | ------- | ----------------------------------------------------------- |
| INVALID\_ARGUMENT             | 请求参数不合法 | 缺少必填字段；字段类型错误；字段格式不合法；角色枚举非法；名称或备注超长；请求中包含未定义字段；违反最后一个管理员约束 |
| PERMISSION\_DENIED            | 权限不足    | 当前用户不是家庭管理员，但尝试编辑家庭名称、邀请成员、修改成员角色、修改成员备注、移出成员或删除家庭          |
| HOUSEHOLD\_NOT\_FOUND         | 家庭不存在   | `household_id` 格式合法，但目标家庭不存在或当前用户不可访问                       |
| HOUSEHOLD\_MEMBER\_NOT\_FOUND | 家庭成员不存在 | `member_user_id` 格式合法，但目标成员不属于该家庭                           |
| INVITE\_EXPIRED               | 邀请已过期   | `invite_token` 对应邀请已过期                                      |
| INVITE\_USED                  | 邀请已使用   | `invite_token` 对应邀请已被其他用户成功使用                               |

## 5. 账簿 Module

账簿 Module 覆盖创建账簿、账簿列表、切换当前账簿、账簿详情、编辑账簿、删除账簿等能力。

该 Module 的 Interface 只暴露账簿管理意图；以下行为属于后端 Implementation：

- 校验目标家庭存在、账簿归属和当前用户可访问性。
- 校验家庭管理员权限、账簿 `manage` 权限和单人家庭不能开启共享等规则。
- 在数据库事务内更新 `ledgers`、`ledger_members`、`ledger_summaries`、`user_household_settings`。
- 删除账簿时级联删除账单、授权和相关统计汇总。
- 写入关键操作日志。

术语说明：

- `user_household_settings.current_ledger_id`：用户在某个家庭维度下当前选中的账簿；允许为 `null`。
- V1.0 不存在“隐式默认账簿”接口约定；所有依赖具体账簿的查询和写操作都必须显式传入 `ledger_id`。
- `default_flag` 属于后端内部数据设计；V1.0 不把它暴露为调用方契约，也不依赖它做接口级兜底选择。
- `shared_member_user_ids` 只表达账簿共享范围；V1.0 在创建/编辑账簿接口中不表达 `read` / `write` / `manage` 的账簿级权限细分。

### 5.1 账簿列表项展示模型

账簿列表接口返回账簿列表项展示模型，供账簿侧边栏和“我的账簿”列表直接消费。

```json
{
  "id": "ld_001",
  "household_id": "hh_001",
  "name": "日常账簿",
  "cover": {
    "type": "preset",
    "preset_key": "notebook",
    "asset_url": null
  },
  "is_shared": true,
  "my_permission": "manage",
  "is_current": true,
  "created_at": "2026-07-01T10:00:00+08:00",
  "updated_at": "2026-07-03T09:30:00+08:00"
}
```

字段说明：

| 字段             | 类型      | 必返 | 说明                                        |
| -------------- | ------- | -- | ----------------------------------------- |
| id             | string  | 是  | 账簿 ID                                     |
| household\_id  | string  | 是  | 家庭 ID                                     |
| name           | string  | 是  | 账簿名                                       |
| cover          | object  | 是  | 账簿封面展示信息                                  |
| is\_shared     | boolean | 是  | 是否开启账簿共享                                  |
| my\_permission | string  | 是  | 当前用户对该账簿的有效权限：`read` / `write` / `manage` |
| is\_current    | boolean | 是  | 是否为该家庭下当前选中的账簿                            |
| created\_at    | string  | 是  | 创建时间                                      |
| updated\_at    | string  | 是  | 更新时间                                      |

`cover` 字段说明：

| 字段          | 类型            | 必返 | 说明                               |
| ----------- | ------------- | -- | -------------------------------- |
| type        | string        | 是  | `preset` / `custom`              |
| preset\_key | string / null | 是  | 预设封面 key；`type = preset` 时返回非空   |
| asset\_url  | string / null | 是  | 自定义封面可展示地址；`type = custom` 时返回非空 |

展示规则：

- `my_permission` 返回后端按既定权限顺序计算出的最终有效权限，不额外返回权限来源。
- `is_current` 依据该用户在 `user_household_settings.current_ledger_id` 中的记录计算；当该家庭下 `current_ledger_id = null` 时，所有列表项都返回 `false`。
- V1.0 不返回 `default_flag`、`can_manage`、`display_cover_url`、`cover_thumbnail_url`、`bill_count` 或其他派生字段。
- `cover.type = preset` 时，`preset_key` 返回非空字符串，`asset_url` 固定返回 `null`。
- `cover.type = custom` 时，`asset_url` 返回可展示地址，`preset_key` 固定返回 `null`。

### 5.2 账簿详情展示模型

账簿详情接口在账簿主页、编辑账簿页回显和删除前确认场景复用同一展示模型。

```json
{
  "id": "ld_001",
  "household_id": "hh_001",
  "name": "日常账簿",
  "cover": {
    "type": "preset",
    "preset_key": "notebook",
    "asset_url": null
  },
  "is_shared": true,
  "my_permission": "manage",
  "shared_member_user_ids": [
    "user_002",
    "user_003"
  ],
  "bill_count": 18,
  "total_income_cent": 300000,
  "total_expense_cent": 125500,
  "last_recorded_at": "2026-07-03T10:30:00+08:00",
  "note": "家庭日常开销",
  "created_at": "2026-07-01T10:00:00+08:00",
  "updated_at": "2026-07-03T09:30:00+08:00"
}
```

字段说明：

| 字段                        | 类型            | 必返 | 说明                                        |
| ------------------------- | ------------- | -- | ----------------------------------------- |
| id                        | string        | 是  | 账簿 ID                                     |
| household\_id             | string        | 是  | 家庭 ID                                     |
| name                      | string        | 是  | 账簿名                                       |
| cover                     | object        | 是  | 账簿封面展示信息；字段定义复用第 5.1 节                    |
| is\_shared                | boolean       | 是  | 是否开启账簿共享                                  |
| my\_permission            | string        | 是  | 当前用户对该账簿的有效权限：`read` / `write` / `manage` |
| shared\_member\_user\_ids | array         | 是  | 账簿共享成员用户 ID 列表                            |
| bill\_count               | integer       | 是  | 收录笔数                                      |
| total\_income\_cent       | integer       | 是  | 总收入，单位分                                   |
| total\_expense\_cent      | integer       | 是  | 总支出，单位分                                   |
| last\_recorded\_at        | string / null | 是  | 最后记录日期；无记录时返回 `null`                      |
| note                      | string / null | 是  | 备注；无备注时返回 `null`                          |
| created\_at               | string        | 是  | 创建时间                                      |
| updated\_at               | string        | 是  | 更新时间                                      |

详情规则：

- `shared_member_user_ids` 只返回通过账簿共享显式加入的其他家庭成员 ID；不重复返回创建者本人，也不把家庭管理员隐式展开进该数组。
- `is_shared = false` 时，`shared_member_user_ids` 固定返回 `[]`。
- 无账单时，`bill_count` 返回 `0`，`total_income_cent` 返回 `0`，`total_expense_cent` 返回 `0`，`last_recorded_at` 返回 `null`。
- V1.0 不返回完整共享成员资料对象；编辑页需要的人名、头像等信息应通过第 4.6 节“家庭详情查询”接口中的 `members` 获取。
- V1.0 不返回 `default_flag`、`can_edit`、`can_delete`、`can_share`、`stats_snapshot` 或其他派生字段。

### 5.3 创建账簿

```text
POST /api/ledgers/create
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "name": "旅行基金",
  "cover": {
    "type": "preset",
    "preset_key": "bell",
    "asset_id": null
  },
  "is_shared": true,
  "shared_member_user_ids": [
    "user_002"
  ],
  "note": "2026 年旅行预算"
}
```

请求字段：

| 字段                        | 类型            | 必填 | 说明                           |
| ------------------------- | ------------- | -- | ---------------------------- |
| household\_id             | string        | 是  | 家庭 ID                        |
| name                      | string        | 是  | 账簿名，必填，最大 `20` 个字符           |
| cover                     | object        | 是  | 账簿封面配置                       |
| is\_shared                | boolean       | 是  | 是否开启账簿共享                     |
| shared\_member\_user\_ids | array         | 是  | 共享成员用户 ID 列表；无共享成员时传 `[]`    |
| note                      | string / null | 是  | 备注；无备注时传 `null`，最大 `200` 个字符 |

`cover` 请求字段：

| 字段          | 类型            | 必填 | 说明                                         |
| ----------- | ------------- | -- | ------------------------------------------ |
| type        | string        | 是  | `preset` / `custom`                        |
| preset\_key | string / null | 是  | `type = preset` 时必填                        |
| asset\_id   | string / null | 是  | `type = custom` 时必填；取文件上传 Module 成功后的文件 ID |

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "ld_003",
    "household_id": "hh_001",
    "name": "旅行基金",
    "cover": {
      "type": "preset",
      "preset_key": "bell",
      "asset_url": null
    },
    "is_shared": true,
    "my_permission": "manage",
    "shared_member_user_ids": [
      "user_002"
    ],
    "bill_count": 0,
    "total_income_cent": 0,
    "total_expense_cent": 0,
    "last_recorded_at": null,
    "note": "2026 年旅行预算",
    "created_at": "2026-07-03T12:00:00+08:00",
    "updated_at": "2026-07-03T12:00:00+08:00"
  }
}
```

Interface 规则：

- 创建账簿要求完整请求字段：`household_id`、`name`、`cover`、`is_shared`、`shared_member_user_ids`、`note`。
- 所有顶层字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- 只有家庭管理员可以创建账簿，否则返回 `PERMISSION_DENIED`。
- `name` 必须为非空字符串，最大 `20` 个字符；为空、超长或类型非法时返回 `INVALID_ARGUMENT`。
- `note` 允许为 `null`；非空时最大 `200` 个字符，超长或类型非法时返回 `INVALID_ARGUMENT`。
- `cover.type = preset` 时，`preset_key` 必填，`asset_id` 必须显式传 `null`。
- `cover.type = custom` 时，`asset_id` 必填，`preset_key` 必须显式传 `null`。
- `cover.type` 非法、`preset_key` / `asset_id` 与 `type` 不匹配，或 `cover` 中出现未定义字段时返回 `INVALID_ARGUMENT`。
- `is_shared = false` 时，`shared_member_user_ids` 必须显式传 `[]`，不能夹带共享成员。
- `is_shared = true` 时，家庭成员数必须大于 `1`，否则返回 `INVALID_ARGUMENT`。
- `is_shared = true` 时，`shared_member_user_ids` 中的每个用户都必须属于该家庭、不能重复、不能包含当前创建人自己；否则返回 `INVALID_ARGUMENT`。
- V1.0 的 `shared_member_user_ids` 只表达共享范围；这些成员通过该接口获得该账簿 `read` 权限，不在该接口中表达 `write` / `manage`。
- 创建成功后，后端把当前用户在该 `household_id` 下的 `user_household_settings.current_ledger_id` 设置为新账簿 ID；若记录不存在则统一补建。
- 创建成功响应直接返回第 5.2 节“账簿详情展示模型”。
- V1.0 不在创建账簿接口中隐式创建分类、授权记录或其他附属资源。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 5.4 账簿列表查询

```text
POST /api/ledgers/query
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "cursor": null,
  "limit": 20
}
```

请求字段：

| 字段            | 类型            | 必填 | 说明                          |
| ------------- | ------------- | -- | --------------------------- |
| household\_id | string        | 是  | 家庭 ID                       |
| cursor        | string / null | 是  | 游标；首次查询传 `null`，后续翻页传不透明字符串 |
| limit         | integer       | 是  | 页大小；必须为大于 `0` 的整数，最大 `100`  |

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "items": [
      {
        "id": "ld_001",
        "household_id": "hh_001",
        "name": "日常账簿",
        "cover": {
          "type": "preset",
          "preset_key": "notebook",
          "asset_url": null
        },
        "is_shared": true,
        "my_permission": "manage",
        "is_current": true,
        "created_at": "2026-07-01T10:00:00+08:00",
        "updated_at": "2026-07-03T09:30:00+08:00"
      }
    ],
    "next_cursor": null,
    "has_more": false
  }
}
```

Interface 规则：

- 账簿列表查询继续统一使用 `POST`，不引入 `GET /ledgers`。
- 顶层字段只保留 `household_id`、`cursor`、`limit` 三个字段。
- 三个字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `cursor` 首次查询必须显式传 `null`；后续翻页传不透明字符串。
- `cursor` 损坏、不可解析时返回 `INVALID_ARGUMENT`。
- `limit` 必须为大于 `0` 的整数，最大 `100`。
- 账簿列表只返回当前用户在该家庭下可访问的账簿。
- 当该家庭下 `current_ledger_id = null` 时，返回结果中允许所有 `items[].is_current = false`；后端不做隐式账簿兜底选择。
- V1.0 不支持名称搜索、共享状态筛选、权限筛选或排序参数；请求中出现未定义字段时返回 `INVALID_ARGUMENT`。
- 查询结果使用稳定顺序；Implementation 至少保证同一游标链路下顺序稳定。

查询成功响应字段：

| 字段                | 类型            | 必返 | 说明                            |
| ----------------- | ------------- | -- | ----------------------------- |
| data.items        | array         | 是  | 账簿列表；单条元素复用第 5.1 节“账簿列表项展示模型” |
| data.next\_cursor | string / null | 是  | 下一页游标；无更多数据时返回 `null`         |
| data.has\_more    | boolean       | 是  | 是否还有更多数据                      |

响应规则：

- 查询成功响应中的 `data` 只返回 `items`、`next_cursor`、`has_more` 三个字段。
- V1.0 不在 `data` 顶层追加 `current_ledger_id`、`total` 或其他字段。
- 空结果时返回 `items = []`、`next_cursor = null`、`has_more = false`。

### 5.5 切换当前账簿

```text
POST /api/ledgers/switch
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "ledger_id": "ld_002"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001",
    "ledger_id": "ld_002"
  }
}
```

Interface 规则：

- 切换当前账簿要求完整请求字段：`household_id`、`ledger_id`。
- 两个字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `ledger_id` 为空字符串、格式不合法或类型非法时返回 `INVALID_ARGUMENT`。
- `ledger_id` 格式合法但目标账簿不存在、不属于该家庭或当前用户无访问权限时返回 `LEDGER_NOT_FOUND`。
- 切换成功后，后端把该 `household_id` 对应的 `user_household_settings.current_ledger_id` 更新为目标账簿；若记录不存在则统一补建。
- 该接口只更新指定家庭下的 `current_ledger_id`，不修改 `user_settings.current_household_id`。
- 如果目标账簿已经是当前账簿，仍返回成功，不返回“无变更”错误。
- 成功响应只返回 `household_id`、`ledger_id`，不额外返回账簿详情、首页统计或侧边栏列表。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 5.6 账簿详情查询

```text
POST /api/ledgers/detail
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "ledger_id": "ld_001"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "ld_001",
    "household_id": "hh_001",
    "name": "日常账簿",
    "cover": {
      "type": "preset",
      "preset_key": "notebook",
      "asset_url": null
    },
    "is_shared": true,
    "my_permission": "manage",
    "shared_member_user_ids": [
      "user_002",
      "user_003"
    ],
    "bill_count": 18,
    "total_income_cent": 300000,
    "total_expense_cent": 125500,
    "last_recorded_at": "2026-07-03T10:30:00+08:00",
    "note": "家庭日常开销",
    "created_at": "2026-07-01T10:00:00+08:00",
    "updated_at": "2026-07-03T09:30:00+08:00"
  }
}
```

Interface 规则：

- 账簿详情查询要求完整请求字段：`household_id`、`ledger_id`。
- 两个字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `ledger_id` 格式合法但目标账簿不存在、不属于该家庭或当前用户无访问权限时返回 `LEDGER_NOT_FOUND`。
- 成功响应直接返回第 5.2 节“账簿详情展示模型”。
- V1.0 不提供“省略 `ledger_id` 时自动读取当前账簿”的接口约定；调用方必须显式传入目标账簿 ID。
- V1.0 不返回额外的页面态权限提示字段，由调用方结合 `my_permission` 判断编辑、删除和授权入口展示。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 5.7 编辑账簿

```text
POST /api/ledgers/update
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "ledger_id": "ld_001",
  "name": "旅行基金",
  "cover": {
    "type": "custom",
    "preset_key": null,
    "asset_id": "asset_001"
  },
  "is_shared": true,
  "shared_member_user_ids": [
    "user_002",
    "user_003"
  ],
  "note": "2026 年旅行预算"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "id": "ld_001",
    "household_id": "hh_001",
    "name": "旅行基金",
    "cover": {
      "type": "custom",
      "preset_key": null,
      "asset_url": "https://example.com/ledger-cover.png"
    },
    "is_shared": true,
    "my_permission": "manage",
    "shared_member_user_ids": [
      "user_002",
      "user_003"
    ],
    "bill_count": 18,
    "total_income_cent": 300000,
    "total_expense_cent": 125500,
    "last_recorded_at": "2026-07-03T10:30:00+08:00",
    "note": "2026 年旅行预算",
    "created_at": "2026-07-01T10:00:00+08:00",
    "updated_at": "2026-07-03T12:10:00+08:00"
  }
}
```

Interface 规则：

- 编辑账簿要求完整请求字段：`household_id`、`ledger_id`、`name`、`cover`、`is_shared`、`shared_member_user_ids`、`note`。
- 所有字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `ledger_id` 格式合法但目标账簿不存在、不属于该家庭或当前用户无访问权限时返回 `LEDGER_NOT_FOUND`。
- 当前用户对目标账簿没有 `manage` 权限时返回 `PERMISSION_DENIED`。
- `name`、`note`、`cover`、`is_shared`、`shared_member_user_ids` 的校验规则与第 5.3 节“创建账簿”保持一致。
- `is_shared = false` 时，后端直接清空该账簿的共享成员范围，响应中 `shared_member_user_ids` 返回 `[]`。
- 单人家庭不允许把账簿编辑为共享账簿；若 `is_shared = true` 且家庭成员数不大于 `1`，返回 `INVALID_ARGUMENT`。
- V1.0 的编辑账簿接口只维护共享范围，不在该接口中直接授予或回收账簿级 `write` / `manage` 权限。
- 编辑成功后不修改该账簿在其他用户侧的“是否当前账簿”选择状态；只有显式调用第 5.5 节“切换当前账簿”接口或第 5.8 节“删除账簿”接口时，`current_ledger_id` 才会变化。
- 如果自定义封面被替换，事务提交后异步清理旧封面文件。
- 成功响应直接返回第 5.2 节“账簿详情展示模型”。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 5.8 删除账簿

```text
POST /api/ledgers/delete
```

HTTP 状态码：

```text
200 OK
```

请求：

```json
{
  "household_id": "hh_001",
  "ledger_id": "ld_001"
}
```

成功响应：

```json
{
  "code": "SUCCESS",
  "message": "success",
  "data": {
    "household_id": "hh_001",
    "ledger_id": "ld_001"
  }
}
```

Interface 规则：

- 删除账簿要求完整请求字段：`household_id`、`ledger_id`。
- 两个字段都必须显式传入；缺失任一字段时返回 `INVALID_ARGUMENT`。
- `household_id` 格式合法但目标家庭不存在或当前用户不可访问时返回 `HOUSEHOLD_NOT_FOUND`。
- `ledger_id` 格式合法但目标账簿不存在、不属于该家庭或当前用户无访问权限时返回 `LEDGER_NOT_FOUND`。
- 当前用户对目标账簿没有 `manage` 权限时返回 `PERMISSION_DENIED`。
- 删除账簿是物理删除，不支持恢复。
- 允许删除家庭下最后一个账簿。
- 删除账簿后，后端要把该家庭下所有 `user_household_settings.current_ledger_id = 目标账簿 ID` 的记录统一置为 `null`，不自动切到其他账簿。
- 成功响应只返回 `household_id`、`ledger_id`，不返回新的当前账簿信息。
- 删除后若该家庭已无任何账簿，依赖账簿的查询继续按“无账簿”规则返回空列表或 `0` 值；记账写操作应由对应接口返回“请先创建账簿”类错误。
- Implementation 必须在事务内删除账簿、账簿成员关系、账单、授权和相关统计汇总。
- 事务提交后异步删除账簿封面文件。
- 请求中出现未定义字段时返回 `INVALID_ARGUMENT`。

### 5.9 错误码

| 错误码                   | 文案      | 触发场景                                                                        |
| --------------------- | ------- | --------------------------------------------------------------------------- |
| INVALID\_ARGUMENT     | 请求参数不合法 | 缺少必填字段；字段类型错误；字段格式不合法；名称或备注超长；封面字段与 `type` 不匹配；共享成员列表非法；请求中包含未定义字段；单人家庭开启共享 |
| PERMISSION\_DENIED    | 权限不足    | 当前用户没有家庭管理员权限但尝试创建账簿；或当前用户没有目标账簿 `manage` 权限但尝试编辑、删除账簿                      |
| HOUSEHOLD\_NOT\_FOUND | 家庭不存在   | `household_id` 格式合法，但目标家庭不存在或当前用户不可访问                                       |
| LEDGER\_NOT\_FOUND    | 账簿不存在   | `ledger_id` 格式合法，但目标账簿不存在、不属于指定家庭或当前用户无访问权限                                 |

