# 词典子系统总览（Dictionary Overview）

> 词典子系统的入口：回答「单词是什么、释义从哪里来」。先读本文档（3～5 分钟），字段与 schema 见 [data-model.md](data-model.md)，词形归一见 [normalization.md](normalization.md)，在线补充见 [enrichment.md](enrichment.md)。

## 1. 数据链路

ECDICT 在导入期写入 `dictionary_entries`，该表就是本地词典库本体，运行时不会再次查询 ECDICT 文件（D001）。

```text
导入期：ECDICT → 离线导入 → dictionary_entries
运行时：业务 → DictionaryService → DictionaryRepository → dictionary_entries
```

MVP 查询未命中时创建最小词条。现行异步导入与守卫机制、V2 在线 Lookup / Enrich 设计见 [enrichment.md](enrichment.md)。部署入口见 [release.md](../deploy/release.md)，导入进度见 [Meta API §3](../api/meta.md#3-词典导入进度)。

## 2. 目标

以下为词典子系统的能力目标；当前仅交付 ECDICT 基础信息，在线增强与 AI 整理的阶段见 [Roadmap](../product/roadmap.md)。已收录词条可补全的信息受源数据覆盖情况限制：

- 单词原形
- 音标
- 词性
- 中文释义
- 英文释义（V2 后续设计）
- 词形变化
- 可选发音音频（V2 后续设计）
- 可选词频、考试标签等辅助信息

同时保证：

- 释义查询速度足够快
- 复习时不依赖实时联网
- 数据源可以替换
- 词典数据和用户学习数据相互独立
- 后续可以加入更高质量词典或 AI 整理，而不用重构核心业务

## 3. 三条原则

### 原则一：本地优先

MVP 查询与复习均使用本地词典库，不等待第三方服务；导入流程见第 1 节。

### 原则二：AI 不制造词典事实

AI 只做 `raw_meanings → 整理 → review_meanings` 的 presentation / transformation 层，而不是 `Word → AI 猜释义` 的词典来源，避免 AI 产生的错误词义进入长期词典数据。这是 `review_meanings` 与 `raw_meanings` 分离的原因之一，见 [data-model.md](data-model.md)。

### 原则三：词典数据 ≠ 用户学习数据

词典事实与个人学习状态分开维护；字段归属与个人释义覆盖规则见 [data-model.md](data-model.md)。

## 4. 为什么不使用实时翻译 API 作为核心方案

不建议采用「导入单词 → 调用翻译 API → 保存翻译结果」作为主要方案：

1. 翻译 API 返回结果通常不是真正的词典结构。
2. 词性、音标、词形等信息可能缺失。
3. 网络请求会增加导入延迟。
4. 大批量导入时可能产生大量 API 调用。
5. 部分商业词典 API 存在缓存和长期保存限制。
6. 系统会被某一个第三方服务绑定。

因此第一版优先采用本地词典数据库。

## 5. 数据源分工

- **ECDICT** 提供：中文释义、音标、词性、词形变化、lemma / 单词原形、BNC / COCA 词频、Oxford / Collins 等辅助标签、CET4 / CET6 / 高考等考试标签。
- **Free Dictionary API** 补充：结构化词性、英文释义、英文例句、发音、音频、同义词、反义词。
- **AI** 只负责压缩过长释义、合并重复义项、整理成适合复习的形式、按考试场景选常用义项——原始词典数据始终保留。

用户手动修改不属于数据源，而是用户个人的覆盖层（[data-model.md](data-model.md)），不会进入词典数据。

## 6. 能力阶段边界

能力阶段统一见 [Roadmap](../product/roadmap.md#1-版本总览)；在线链路与 AI 整理的设计分别见 [enrichment.md](enrichment.md) 和 [data-model.md](data-model.md)。

## 7. 文档导览

| 想了解的部分 | 文档 |
| --- | --- |
| `dictionary_entries` / `user_words` 字段与语义 | [data-model.md](data-model.md) |
| 词形归一（lemma resolve） | [normalization.md](normalization.md) |
| Lookup / Enrich 与多来源 Merge | [enrichment.md](enrichment.md) |
| 字段归属（两表如何服务于 word / review 领域） | [../architecture/data-model.md](../architecture/data-model.md) |
