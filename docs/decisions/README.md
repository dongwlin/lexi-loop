# 决策索引（Decisions）

> 记录项目已冻结的关键决策。当前规模下只维护这张索引表，不在每条决策单建 ADR 文件；只有将来遇到需要保留**决策背景与权衡过程**的问题（例如「为什么当时不用 FSRS？」）时，才为单条决策补写 ADR。

## 决策登记

ID | 决策 | 状态 | 权威文档 | 一句话
--- | --- | --- | --- | ---
D001 | ECDICT 全量本地化 | Accepted | [dictionary/overview.md](../dictionary/overview.md) | ECDICT 是导入期数据源，`dictionary_entries` 即本地词典库本体，运行时无「再查 ECDICT」链路（细节见 [enrichment.md](../dictionary/enrichment.md)） |
D002 | Lookup / Enrich 分离 | Accepted | [dictionary/enrichment.md](../dictionary/enrichment.md) | Lookup 解决「没有词条」，Enrich 解决「词条缺增强字段」，同一 Provider 机制的两个方向 |
D003 | lemma 不确定不归并 | Accepted | [dictionary/normalization.md](../dictionary/normalization.md) | 能确定才自动归一；不能确定（saw / lay / bound / better）保留原词 |
D004 | 生词软删除 | Accepted | [dictionary/data-model.md](../dictionary/data-model.md) | 置 `user_words.deleted_at` 不物理删行，保 `review_items` 历史；重导入恢复该行 |
D005 | Review Submit 幂等与原子提交 | Accepted | [api/reviews.md](../api/reviews.md) | 条件 UPDATE 短路重复提交与跨 session 错配；item、词级累计与 session 完成在同一事务提交 |
D006 | 词典数据 ≠ 用户学习数据 | Accepted | [dictionary/data-model.md](../dictionary/data-model.md) | `dictionary_entries` / `user_words` 两表分离，系统内不存在「词典事实 + 学习状态」混表 |
D007 | 释义三层取值 | Accepted | [dictionary/data-model.md](../dictionary/data-model.md) | 展示时 `custom ?? review_meanings ?? raw_meanings`；用户自定义只写 `user_words` |
D008 | 四表链路与 1:N 概念关系 | Accepted | [architecture/data-model.md](../architecture/data-model.md) | `dictionary_entries → user_words → review_items → review_sessions`；概念 1:N，MVP 单用户表现为 1:0..1 |
D009 | 复习数量截断 | Accepted | [api/reviews.md](../api/reviews.md) | count > available 时 `total_count = min(count, available)` 不报错，UI 提示实际数量 |
D010 | Session 恢复与单 active | Accepted | [review/data-model.md](../review/data-model.md) | 重进 `/review` 由用户选择继续或放弃；同一时间仅一个 active session |
D011 | 权重 / mastery 不落库 | Accepted | [review/algorithm.md](../review/algorithm.md) | 派生指标使用前动态计算，库中只存 `encounter / review / streak / last_reviewed_at` 等原始数据 |
D012 | MVP 只做随机复习 | Accepted | [product/roadmap.md](../product/roadmap.md) | 到期复习（SRS 形态）后置到 V2 评估 / V3 结合 FSRS |

## 约定

- 决策的**结论与依据**以「权威文档」列为准；本表只做索引与一句话摘要，不重复完整论证。
- 状态仅 Accepted（已冻结）。新决策先落在对应领域文档，再登记到本表。
- 出现跨文档冲突时，按 [AGENTS.md](../../AGENTS.md) 的冲突裁决原则处理。
