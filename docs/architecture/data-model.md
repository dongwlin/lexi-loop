# 数据模型总览（关系级）

> 本文档只描述四张核心表的**关系与职责**，是整个数据模型的地图。它不重复列出各表字段——表字段的唯一权威在各自领域文档：
>
> - `dictionary_entries` / `user_words` 字段：见 [dictionary/data-model.md](../dictionary/data-model.md)
> - `review_sessions` / `review_items` 字段：见 [review/data-model.md](../review/data-model.md)

## 1. 四表关系

```text
dictionary_entries         客观词典事实（词条：单词是什么）
        1
        │  概念模型 1:N；MVP 单用户表现为 1:0..1（见第 3 节）
        N
    user_words              用户学习状态（用户对某词条的一行学习记录）
        1
        │  1:N（同一词可出现在多轮复习里）
        N
   review_items             单次复习结果（某轮里某词的作答）
        N
        │  N:1（一次作答属于某一轮）
        1
 review_sessions           一轮复习
```

主链路：`dictionary_entries → user_words → review_items → review_sessions`。

## 2. 各表职责

| 表 | 负责什么 | 一句话 |
| --- | --- | --- |
| `dictionary_entries` | 客观词典事实 | 单词是什么（音标、释义、词形等），全局共享，不随用户改动 |
| `user_words` | 用户学习状态 | 用户对一个词条掌握得怎么样（遇词 / 复习统计 + 个人释义覆盖） |
| `review_items` | 单次复习结果 | 一轮里每个单词的逐次「记得 / 忘记」记录 |
| `review_sessions` | 一轮复习 | 一次完整的复习会话及其汇总（active / completed / abandoned） |

字段与语义分别在 [dictionary/data-model.md](../dictionary/data-model.md)（前两张）与 [review/data-model.md](../review/data-model.md)（后两张）定义，本文档不重复维护任何 schema。

## 3. 概念模型 1:N，MVP 实际表现为 1:0..1

`dictionary_entries 1:N user_words` 是面向未来多用户的概念模型，对应约束 `UNIQUE(user_id, dictionary_entry_id)`——同一用户对一个词条只能有一行学习状态。

MVP 只有单用户：`user_words` 不含 `user_id`，以 `UNIQUE(dictionary_entry_id)` 保证一用户一词条一行，因此每个 `dictionary_entries` 至多对应 0 或 1 条 `user_words`（一个词条可能还没被任何用户导入），实际关系表现为 1:0..1。未来引入多用户时，加 `user_id` 并把约束改为 `UNIQUE(user_id, dictionary_entry_id)` 即恢复 1:N。

各表的「职责拆分」是词典方案冻结的核心结论（决策索引 [D008](../decisions/README.md)），字段级说明见各自的 data-model 文档。
