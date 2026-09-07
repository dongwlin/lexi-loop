# LexiLoop Roadmap（阶段规划）

> 本文档回答「以后准备怎么长」：MVP → V2 → V3 的能力演进、复习的两种形态、MVP 完成状态与后续开发入口。
> 它只做版本与顺序层面的规划，各项能力的详细规则在对应专项文档；「当前产品具体是什么、MVP 详细边界」以 [prd.md](prd.md) 为准。

## 1. 版本总览

截至 2026-09-07，MVP 阶段目标已完成，发布基线为仓库标签 `v0.0.1`。当前产品规则继续由 [prd.md](prd.md) 及其领域文档维护；发布操作见 [release.md](../deploy/release.md)。

```text
MVP（第一个可用闭环）
↓
V2（在线词典增强 + 算法升级）
↓
V3（AI 整理 + 完整 SRS + 多用户）
```

### MVP（已完成，现行基线）

- **生词导入**：多行批量粘贴；同一单词重复导入累计遇词次数；词形归并（见 [dictionary/normalization.md](../dictionary/normalization.md)）。
- **本地词典**：ECDICT 全量离线导入本地，导入即自动补全原形 / 音标 / 词性 / 中文释义（见 [dictionary/overview.md](../dictionary/overview.md)）。
- **加权随机复习**：用户选定本轮数量 → 一轮固定抽取（加权、不放回）→ 先回忆后看释义 → 记得 / 不记得（页面交互见 [frontend/review-flow.md](../frontend/review-flow.md)，算法见 [review/algorithm.md](../review/algorithm.md)）。
- **复习历史**：`review_sessions` / `review_items` 逐次记录；每词统计、掌握程度与复习优先级展示（数据模型见 [review/data-model.md](../review/data-model.md)）。

MVP 明确不做：到期复习提醒、完整 SRS / FSRS、词典在线增强（Lookup 在线分支与 Enrich）、AI 整理、多用户。MVP 的完成标准见第 3 节。

### V2（后续规划）

- **Lookup 在线兜底 + Enrich 增强**：为本地词典库没有的词创建词条、为已有词条补英文释义 / 音频 / 例句（阶段边界见 [dictionary/enrichment.md](../dictionary/enrichment.md)）。
- **复习算法升级**：统计最近 N 次表现、对越新的结果指数加权，用于评估「到期复习」提示（机制见 [review/algorithm.md](../review/algorithm.md)）。
- **错词复习**：只复习本轮忘记的单词入口。
- **「到期复习」评估**（B 形态，见第 2 节）：V2 评估提示机制，完整模式仍在 V3 结合 FSRS 评估落地（D012）。

### V3（后续规划）

- **AI review_meanings**：`raw_meanings → AI → 复习释义` 的整理层，AI 不制造词典事实（见 [dictionary/data-model.md](../dictionary/data-model.md)）。
- **FSRS / 到期复习**：再评估 FSRS 这类完整间隔重复算法并正式引入到期复习。
- **多用户与账号体系**：数据模型按 1:N 恢复（见 [architecture/data-model.md](../architecture/data-model.md)）。

## 2. 复习的两种形态

复习存在两种形态，产品上需要分清，**第一版只实现 A**：

- **A. 随机复习**：用户指定「我现在想背 30 个」，系统加权随机抽取。第一版实现。
- **B. 到期复习**：系统判断某些词已接近遗忘临界点，提示「今天有 23 个单词建议复习」。传统 SRS 形态，第一版不做，V2 评估、V3 结合 FSRS 落地。

## 3. MVP 完成定义

以下为已达成的 MVP 产品闭环，保留作为后续迭代的回归依据：

```text
我今天做了一篇英语阅读
↓
复制：derive / ambiguous / subtle / constrain
↓
导入（系统自动补全释义）
↓
第二天又碰到 ambiguous
↓
再次导入 ambiguous → encounter_count + 1
↓
晚上选择复习 20 个
↓
ambiguous 因为出现次数高，更容易被抽到
↓
我忘记 ambiguous → forget_count + 1
↓
下一轮 ambiguous 的权重进一步升高
↓
后来连续几次都记住
↓
它的优先级逐渐下降
```

这个流程即 [prd.md](prd.md) 中「核心循环」的产品闭环。

## 4. 后续开发与历史归档

MVP 已完成，后续工作在现有闭环上迭代。V2 / V3 是能力演进方向，尚不表示全部细节已定稿或已经实现；具体任务开始前，先更新对应领域的权威文档、契约与必要的数据迁移，再同步实现和引用。

早期的垂直切片开发顺序已移至 [历史归档](../archive/mvp-implementation-plan.md)，不再作为待办清单。各模块 README 仅维护运行与实现入口，阶段规划统一在本文维护。
