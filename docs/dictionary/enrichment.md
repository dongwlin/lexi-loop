# Lookup / Enrich 与多来源合并（Dictionary Enrichment）

> 适用阶段：本地查询与最小词条兜底为现行行为；Online Provider、在线 Lookup、Enrich 与 Merge 为 V2 后续设计，AI 整理为 V3 后续设计，均不代表已接入。阶段状态见 [Roadmap](../product/roadmap.md)。

> Lookup 与 Enrich 是同一套 Provider 机制的两个方向，放在同一篇：本文档定义在线补充的两条链路、`DictionaryProvider` 接口、多来源 Merge 原则与词典能力的阶段边界。
> 表字段定义见 [data-model.md](data-model.md)，导入时的归一前置见 [normalization.md](normalization.md)。

## 1. ECDICT 全量本地化：dictionary_entries 就是本地词典库

ECDICT 作为**导入期数据源**：大型 CSV 通过离线脚本一次性导入 PostgreSQL，运行时不读取 CSV。

在这一架构下，`dictionary_entries` 不是「按需查询 ECDICT 的缓存」，而是**系统本地词典库本身**。凡 ECDICT 收录的词，导入后都直接查询 `dictionary_entries` 命中，例如 `Lookup("ambiguous")`；不存在运行时「查不到再从 ECDICT 查并写回」的链路。

这样具有：

- 查询快
- 无网络依赖
- 无 API 调用次数限制
- 导入大量单词时性能稳定
- 复习页面无需等待第三方词典

当前运行时不访问外部词典。V2 的外部访问仅限下节定义的在线 Lookup 兜底与 Enrich 增强两条链路。

## 2. 两条链路的划分

「在线补充」分两条独立链路，用两个职责不同的方法承载：

```text
Lookup   词条不存在时兜底：本地词典库未命中 → Online Provider → 创建词条
Enrich   词条已存在时增强：已有本地词条 → 检查缺少的增强字段 → Online Provider → Merge

MVP：只做「本地词典库查询」；Lookup 的在线分支与 Enrich 都属第二阶段。
```

**Lookup**——业务层导入生词时调用，返回一个可用的 `DictionaryEntry`：

```text
DictionaryService.Lookup(word)
↓
查询本地词典库 dictionary_entries（DictionaryRepository）
├── 命中 → 直接返回（无论是否缺少增强字段，都不在此处补）
└── 未命中 → Online Provider.Lookup（第二阶段起）
    ↓
    Normalize
    ↓
    写入 dictionary_entries（source 标记在线来源）
    ↓
    返回（在线补充失败则返回仅含 headword 的最小词条）
```

注意：运行时不会再走「ECDICT Lookup」这一步，因为 ECDICT 已在导入期全部灌入本地词典库；本地未命中即代表 ECDICT 也没有，进入 Lookup 在线分支（MVP 未接入在线补充时，未命中一律创建最小词条）。

**Enrich**——把已存在的本地词条补全增强字段（英文释义、音频、例句、更结构化词性），第二阶段起在后台按需执行：

```text
DictionaryService.Enrich(entryID)
↓
读取本地 dictionary_entries（入口参数也可直接传入 entry）
↓
检查是否缺少增强字段（如 raw_meanings.en / audio / example 为空）
│   ├── 齐全 → 直接返回，不调用在线源
│   └── 缺少 → Online Provider.Enrich
│       ↓
│       Merge（按第 4 节原则合并，不覆盖已有 ECDICT 字段）
│       ↓
│       更新 dictionary_entries（source 标记在线来源）
```

Enrich 是幂等且后台化的：可以全量扫描缺少增强字段的词条批量补全，也可以对单个词条按需触发；同一词条重复 Enrich 只补齐空缺，不重复调用、不覆盖已有数据。

两者职责边界一句话：**Lookup 解决「没有这个词条」，Enrich 解决「有词条但缺增强字段」**。这样 ECDICT 已收录的普通单词（如 `derive`）也能在第二阶段获得英文释义、音频、例句——Enrich 会补齐它们，而不是因为 Lookup 直接命中而永远得不到增强。

## 3. Provider 抽象：只面向在线补充

业务层不直接依赖具体词典来源。本地词典库的读取走 `DictionaryRepository`（查 `dictionary_entries` 表），**不经 Provider**。

Provider 抽象只服务于两条在线链路（Lookup 兜底与 Enrich 增强），接口包含两个方法：

```go
type DictionaryProvider interface {
    // Lookup：查一个词的完整词典数据（用于本地未命中时创建词条）
    Lookup(ctx context.Context, word string) (*DictionaryEntry, error)
    // Enrich：查某个已有词条的增强字段（英文释义、音频、例句等）
    Enrich(ctx context.Context, headword string) (*Enrichment, error)
}
```

`Enrichment` 指 `raw_meanings.en`、`audio`、`example` 等 ECDICT 缺失、需要在线源补全的字段。

具体实现是 `FreeDictionaryAPIProvider`（实现上述两个方法）；未来增加 Cambridge、Oxford、Collins 等作为补充源时，只需新增 Provider，生词业务无需修改。**ECDICT 不是 Provider**——它是一次性离线导入脚本，导入后即完成使命。

## 4. 多来源合并原则

不同数据源不要简单互相覆盖。ECDICT 的数据已在导入期写入本地词典库（负责中文释义、音标、词频、考试标签、词形）；Free Dictionary API 只作为运行时在线补充，通过两条链路进入：Lookup 在线分支（本地未命中时创建词条）与 Enrich（已有词条补增强字段），负责英文释义、音频、例句、更结构化的词性。最终合并结果：

```text
DictionaryEntry
├─ headword
├─ phonetic_uk / phonetic_us
├─ raw_meanings（含中文义项、英文释义、词性）
├─ review_meanings（词典层默认复习释义，可空）
├─ exchange
├─ frequency
├─ tags
└─ audio（可选，第二阶段起）
```

合并时**不覆盖已有 ECDICT 字段**，只补空缺；`source` 标记数据来源。

## 5. 阶段边界（词典相关能力）

完整的产品 Roadmap 仍以 [product/roadmap.md](../product/roadmap.md) 为准，本文档只记录与词典相关的能力状态：

### MVP

ECDICT only：本地词典库查询 + 本地未命中时创建仅含 headword 的最小词条（释义留待用户手动补充）。音频、英文例句、同义词 / 反义词、AI 整理、Lookup 在线分支、Enrich、多词典融合、商业词典都不做。

### V2

MVP 稳定后接入 Free Dictionary API，启用两个运行时能力：

1. **Lookup 在线分支**：本地词典库未命中时，调用在线源创建词条，补全英文释义、发音、词性、例句等；失败则退回最小词条。
2. **Enrich**：为本地已存在的词条补全增强字段（英文释义、发音音频、英文例句、更结构化的词性），可后台批量执行、幂等可重入。

这一阶段起，ECDICT 已收录的普通单词也能通过 Enrich 获得英文释义、音频与例句，实现「ECDICT 主数据 + Free Dictionary 补充数据」。

### V3

再加入 AI：`raw_meanings → AI → 适合复习的 review_meanings`（AI 只整理、不制造词典事实，见 [overview.md](overview.md) 与 [data-model.md](data-model.md)）。例如自动输出：

```text
derive /dɪˈraɪv/ v.
① 推导；推断
② 获得
③ 源于
```

## 6. 最终原则

词典模块遵循以下原则：

- 本地优先
- 结构化存储
- 词典事实和用户学习状态分离（`dictionary_entries` / `user_words`）
- 原始释义和复习释义分离（`raw_meanings` / `review_meanings`）
- 用户自定义只写 `user_words.custom_review_meaning`
- Provider 可替换
- AI 只整理，不负责制造事实

最终系统应该做到：

> 用户只负责粘贴生词，系统自动完成词形识别、音标获取、词性获取和中文释义补全；之后这些数据全部保存在本地，复习过程不再依赖外部词典服务。
