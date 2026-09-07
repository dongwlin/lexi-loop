# Review API（复习）

> 复习接口的契约与核心约束：一轮开始、逐次作答、放弃进行中的一轮、获取结果。
> Session 生命周期与 `review_items` 状态定义见 [review/data-model.md](../review/data-model.md)；抽样与权重算法见 [review/algorithm.md](../review/algorithm.md)。
> 所有接口遵循 [HTTP API 设计规范](../specs/backend/HTTP%20API%20设计规范.md) 的标准响应结构。

## 1. 接口总览

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| POST | `/api/v1/reviews` | 开始一轮复习（抽词 + 建 session） |
| POST | `/api/v1/reviews/:sessionId/items/:itemId` | 提交一个单词的结果（记得 / 不记得） |
| POST | `/api/v1/reviews/:sessionId/abandon` | 放弃一轮进行中的复习（active → abandoned） |
| GET | `/api/v1/reviews/:id` | 获取一轮复习的汇总与逐词结果 |

`sessionId`、`:id` 与 `itemId` 均使用 UUID v7；路径和 JSON 中都按字符串传输。

## 2. 开始一轮复习

```text
POST /api/v1/reviews
```

请求：

```json
{
  "count": 30
}
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "sessionId": "01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001",
    "requestedCount": 50,
    "totalCount": 23,
    "items": [
      {
        "itemId": "01991f3e-7b4c-7a21-8e3f-2c5d7a9b2002",
        "word": "ambiguous",
        "phonetic": "/æmˈbɪɡjuəs/",
        "effectiveReviewMeaning": [
          { "pos": "adjective", "translations": ["模棱两可的", "含糊不清的"] }
        ]
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `sessionId` | string（UUID v7） | 本轮复习的 session ID |
| `requestedCount` | int | 用户请求数量 |
| `totalCount` | int | 实际抽取数量（截断后的值） |
| `items` | array | 本轮抽中的单词列表 |

服务器完成：获取候选词（`deleted_at IS NULL` 的 `user_words`，join 词典信息）→ 计算每个词权重（[review/algorithm.md](../review/algorithm.md)）→ 加权随机不放回抽取 → 创建 `review_session`（若已存在 `active` session，先标记 abandoned，规则见 [review/data-model.md](../review/data-model.md)）→ 创建对应数量的 `review_items` → 返回本轮单词。

上述“放弃旧 active session → 抽样 → 创建新 session 与全部 items”在同一数据库事务中完成。MVP 通过事务级作用域锁串行化并发创建，并由 active session 的部分唯一索引兜底；不会向客户端返回只创建了 session、但 items 不完整的结果。具体事务顺序见 [backend/structure.md](../backend/structure.md)。

### 契约：count 超出可复习词数 → 截断而不是报错

当请求的 `count` 大于当前可复习的 `user_words` 数量时，**不报错**，按实际可复习数截断：

- `available_count`：`deleted_at IS NULL` 的 `user_words` 数量。
- `requested_count = count`，`total_count = min(count, available_count)`，即 `total_count` 表示本轮实际抽取的单词数。
- 例如用户请求 50，但只有 23 个可复习生词，则 `requested_count = 50`、`total_count = 23`，本轮复习全部 23 个。
- UI 提示：`当前只有 23 个可复习生词，本轮将复习全部 23 个。`（「开始复习」页面在输入大于词库总量时同样提示，见 [frontend/review-flow.md](../frontend/review-flow.md)）
- 服务端至少校验 `count > 0`；上限（如 ≤ 500）在正式 API 契约中确定。

可复习词数为 0 时返回：

```json
{
  "code": "BASE.BIZ.USER_DISABLED",
  "message": "no reviewable words available",
  "data": {}
}
```

## 3. 提交一个单词的结果

```text
POST /api/v1/reviews/:sessionId/items/:itemId
```

请求：

```json
{
  "result": "remembered"
}
```

或：

```json
{
  "result": "forgotten"
}
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

### 契约：pending → remembered / forgotten，只能发生一次，且必须归属匹配

服务器带幂等保护（防止双击「记得」或浏览器重试导致重复计数），并校验 item 确实属于 URL 中的 session，用一条条件 UPDATE 完成：

```text
UPDATE review_items
SET result = :result, reviewed_at = now()
WHERE id = :item_id
  AND session_id = :session_id
  AND result = 'pending'
RETURNING user_word_id
```

- **幂等与归属校验合一**：`AND result = 'pending'` 保证只有未作答的 item 能被作答并计数一次（双击、重试天然短路）；`AND session_id = :session_id` 防止「session A 的 URL + session B 的 item_id」这类组合被错误提交。
- 只有当上述 UPDATE 实际更新了一行（rows affected = 1），才继续更新 `user_words`：`review_count + 1` → `remember_count` / `forget_count + 1` → 更新 `current_streak` → 更新 `last_reviewed_at`。更新的目标行由 `RETURNING user_word_id` 返回，避免按 item_id 反查或信任客户端。`user_words` 字段见 [dictionary/data-model.md](../dictionary/data-model.md)，统计更新的含义见 [review/data-model.md](../review/data-model.md)。
- 若 UPDATE 未命中任何行（item 不存在、不属于该 session、或已不是 `pending`），直接返回当前状态，不再计数。
- session 本身已不是 `active`（abandoned / completed）时，同样幂等短路、不再计数——防止放弃或完成一轮之后，迟到的提交把结果计入已结束的轮次（放弃见第 6 节）。
- item 条件更新、`user_words` 累计字段更新，以及最后一题触发的 session 汇总与完成必须处于同一个数据库事务；任何一步失败都整体回滚。同一 session 的提交先锁定 session 行并按顺序执行，因此并发提交不同 item 也不会漏掉最后完成；相同 item 的重复提交等待首个事务结束后走幂等短路，不会重复累计。

幂等规则一句话：**只有 `pending` 状态允许被作答并计数一次，且 item 必须属于 URL 中的 session**，条件更新让重复请求与错配请求天然短路。

## 4. streak 更新

如果结果为 `remembered`：

- 原来 `streak >= 0` → `streak += 1`
- 原来 `streak < 0` → `streak = 1`

如果结果为 `forgotten`：

- 原来 `streak <= 0` → `streak -= 1`
- 原来 `streak > 0` → `streak = -1`

例如连续记得 3 次 `streak = 3`，然后忘记 → `streak = -1`。

## 5. 完成一轮复习与获取结果

所有 `review_items` 都完成时，`session.status = completed`，计算 `remembered_count`、`forgotten_count`、`completed_at`，然后返回复习总结（生命周期见 [review/data-model.md](../review/data-model.md)）。

```text
GET /api/v1/reviews/:id
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "sessionId": "01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001",
    "status": "completed",
    "total": 30,
    "remembered": 21,
    "forgotten": 9,
    "completedAt": "2026-09-05T14:00:00Z",
    "items": [
      { "word": "ambiguous", "result": "forgotten" }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `sessionId` | string（UUID v7） | session ID |
| `status` | string | session 状态：`active` / `completed` / `abandoned` |
| `total` | int | 本轮总题数 |
| `remembered` | int | 记得数 |
| `forgotten` | int | 忘记数 |
| `completedAt` | string | 完成时间（ISO 8601），未完成时为 `null` |
| `items` | array | 逐词结果列表 |

session 不存在时返回：

```json
{
  "code": "BASE.NOT_FOUND.USER",
  "message": "review session not found",
  "data": {}
}
```

## 6. 放弃一轮复习

```text
POST /api/v1/reviews/:sessionId/abandon
```

请求无请求体。成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

语义（生命周期见 [review/data-model.md](../review/data-model.md)）：

- 仅 `active` 的 session 可放弃：置 `status = abandoned`（`completed_at` 保持为空），已完成作答的 items 与词级统计保持原样，不回滚。
- **幂等**：对已是 `abandoned` 的 session 重复放弃同样返回成功，不报错——与提交的幂等短路哲学一致（D005）。
- 对 `completed` 的 session 放弃属于客户端缺陷（一轮已完成后不存在「放弃」），返回 `422`：

```json
{
  "code": "BASE.BIZ.USER_DISABLED",
  "message": "review session is not active",
  "data": {}
}
```

- session 不存在时返回 `404`（同第 5 节的 `BASE.NOT_FOUND.USER`）。
- 服务端无独立归属校验的额外要求：MVP 单用户阶段全库共享；V3 引入 `user_id` 后按用户校验归属（同 D005 的归属校验）。
- 与开始一轮的并发关系：放弃持有 session 行锁，与提交共享同一锁顺序（session → item → user_word）；放弃与提交并发时二者串行，先提交者按第 3 节计数，先放弃者使后续提交幂等短路。
- 前端「放弃本轮」调用本端点（UI 行为见 [frontend/review-flow.md](../frontend/review-flow.md) §6）；放弃成功后清除本地恢复快照并回到数量配置视图。
