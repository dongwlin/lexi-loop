# 复习数据模型（Review Data Model）

> 复习领域的两张表：`review_sessions`（一轮复习）与 `review_items`（一轮里逐次的复习结果），以及它们的状态迁移与恢复规则。
> 四张核心表的关系级地图见 [architecture/data-model.md](../architecture/data-model.md)。
>
> Review 使用 `user_words.review_count / remember_count / forget_count / current_streak / last_reviewed_at` 作为词级累计状态，字段定义见 [dictionary/data-model.md](../dictionary/data-model.md)，本文档不重复列出。

## 1. review_sessions 表

```sql
review_sessions
---------------
id                 UUID v7，应用层生成，PostgreSQL `uuid`
requested_count
total_count
remembered_count
forgotten_count
status
started_at
completed_at
created_at
```

status 可以是 `active`、`completed`、`abandoned`。

## 2. review_items 表

```sql
review_items
------------
id             UUID v7，应用层生成，PostgreSQL `uuid`
session_id     UUID，外键 → review_sessions.id
user_word_id   UUID，外键 → user_words.id
position
result
created_at
reviewed_at
```

- `user_word_id` 指向 `user_words.id`。
- `result` 可以是 `remembered`、`forgotten` 或 `pending`（还没回答）。
- 一个单词可以出现在多次复习记录里（不同轮），一轮复习包含多个单词记录。
- 同一 session 内 `position` 唯一，同一个 `user_word_id` 至多出现一次，分别由 `UNIQUE(session_id, position)` 与 `UNIQUE(session_id, user_word_id)` 保证。

## 3. 为什么保留逐次记录

`user_words` 保存累计结果，`review_items` 保存逐次记录，两者用途不同。

只有累计数字，算不出「最近一次忘记在什么时候」「连续忘记几次」「最近 10 次正确率」「一个月前的掌握水平」这些后续统计和算法需要的数据。逐次记录（`review_items`）是这些数据的来源。

## 4. Session 状态迁移

### 状态

```text
active
completed
abandoned
```

### 迁移规则（MVP）

```text
创建
↓
active（本轮所有 item 为 pending）
├─ 最后一个 pending item 被作答 → completed（回写 remembered_count / forgotten_count / completed_at）
└─ 用户主动放弃 → abandoned
```

边界行为：

- **复习到 12/30 后关闭页面**：session 停留在 `active`，进度由 `review_items` 保存。
- **重新进入 `/review`**：检测到存在 `active` 的未完成一轮时，提示「继续复习」或「放弃本轮」；继续则回到原进度（12/30），放弃则由前端调用 `POST /api/v1/reviews/:sessionId/abandon` 显式标记 `abandoned`（契约见 [api/reviews.md](../api/reviews.md) §6）。MVP 不自动进入旧 Session，也不自动放弃它——是否恢复由用户明确选择（UI 见 [frontend/review-flow.md](../frontend/review-flow.md)）。
- **同一时间只允许一个 `active` session**：`POST /api/v1/reviews` 创建新 session 时，若已存在 `active` 的一轮，先将其标记为 `abandoned` 再创建。用户明确开始新一轮，即视为放弃旧一轮。
- **数据库最终约束**：MVP 单用户阶段通过 `WHERE status = 'active'` 的部分唯一索引保证全库至多一条 active session；创建流程同时使用事务级作用域锁串行化“放弃旧轮次 → 创建新轮次”。V3 引入 `user_id` 后，约束和锁均改为按用户隔离。具体事务顺序见 [backend/structure.md](../backend/structure.md)。

## 5. 提交结果对 user_words 的影响

一次有效作答（`result = 'pending'` 且归属匹配，见 [api/reviews.md](../api/reviews.md)）后，同步更新目标 `user_words`：

```text
review_count + 1
remember_count / forget_count + 1（按结果）
current_streak 更新（规则见 api/reviews.md 第 4 节）
last_reviewed_at = now()
```

`user_words` 各字段的语义见 [dictionary/data-model.md](../dictionary/data-model.md)；`weight` / `mastery` 等派生指标不落库、从这些字段动态计算（[review/algorithm.md](algorithm.md)）。
