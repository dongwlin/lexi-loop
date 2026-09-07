# MVP 初期工程落地顺序（历史归档）

> 归档日期：2026-09-07。来源：`product/roadmap.md` 原第 4 节。以下保留初期计划以供追溯，不是实际交付记录，也不再指导开发。
> 文中的完全随机抽样是已结束的过渡设想；“只复习错误单词”、筛选和排序不能因出现在此处就视为已交付。当前产品边界见 [PRD](../product/prd.md)，后续阶段见 [Roadmap](../product/roadmap.md)，现行算法见 [review/algorithm.md](../review/algorithm.md)。


MVP 的开发不按「先全部设计完数据库 → 再写完所有 API → 再写前端」的顺序推进，而按垂直切片推进：

### 数据准备（工程步骤）

离线程序把 ECDICT CSV 导入 `dictionary_entries`，运行时不读 CSV（见 [backend/structure.md](../backend/structure.md)）。

### 第一阶段：单词导入

完成 `user_words` 与词典打通、`/import` 页面、`POST /api/v1/words/import`、`/words` 页面。做到粘贴生词 → 数据库出现（含自动补全的释义）→ 重复粘贴 encounter +1。

### 第二阶段：最简单复习

完成 `/review`，随机抽 20 个，查看释义，记得 / 不记得。暂时可以完全随机，先把流程跑通。

### 第三阶段：复习历史

加入 `review_sessions`、`review_items`，开始正式记录学习数据。

### 第四阶段：权重抽样

加入 `UserWord.ReviewWeight(now)` 与 `WeightedSampler`，替换完全随机抽样；公式仍以 [review/algorithm.md](../review/algorithm.md) 为准，实现位置见 [backend/structure.md](../backend/structure.md)。

### 第五阶段：统计

加入 mastery、session result、word statistics。

### 第六阶段：体验优化

加入键盘快捷键、复习进度、只复习错误单词、动画、筛选、排序。

> 各阶段涉及的页面与接口契约以 [frontend/review-flow.md](../frontend/review-flow.md)、[api/words.md](../api/words.md)、[api/reviews.md](../api/reviews.md) 等专项文档为准，本文件只定义顺序。
