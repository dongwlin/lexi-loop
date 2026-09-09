# Lookup / Enrich 与多来源合并（Dictionary Enrichment）

> 适用阶段：本地查询与最小词条兜底为现行行为；Online Provider、在线 Lookup、Enrich 与 Merge 为 V2 后续设计，AI 整理为 V3 后续设计，均不代表已接入。阶段状态见 [Roadmap](../product/roadmap.md)。

> Lookup 与 Enrich 是同一套 Provider 机制的两个方向，放在同一篇：本文档定义在线补充的两条链路、`DictionaryProvider` 接口、多来源 Merge 原则。
> 表字段定义见 [data-model.md](data-model.md)，导入时的归一前置见 [normalization.md](normalization.md)。

## 1. ECDICT 全量本地化：dictionary_entries 就是本地词典库

ECDICT 作为**导入期数据源**：大型 CSV 随镜像分发（`deploy/dict/fetch.sh` 获取 pinned 数据，构建期 COPY 进镜像），serve 启动后在 server 进程内**异步**导入 PostgreSQL——不阻塞 HTTP 服务，导入期间服务正常响应；进度经 Meta API `GET /api/v1/dictionary-import` 暴露，前端右上角指示组件轮询展示并在完成后收敛。启动时先做版本完整性守卫（`source_version` 达到 manifest 期望行数即跳过），因此同时覆盖全新部署、中断续传与数据集升级三种情况；导入失败置 `failed` 不影响进程，重启自动续传。手动 `lexi-loop import-ecdict` 命令保留为逃生口。获取与部署详见 [deploy/release.md](../deploy/release.md)，进度契约见 [Meta API §3](../api/meta.md)。运行时不读取 CSV。

在这一架构下，`dictionary_entries` 不是「按需查询 ECDICT 的缓存」，而是**系统本地词典库本身**。凡 ECDICT 收录的词，导入后都直接查询 `dictionary_entries` 命中，例如 `Lookup("ambiguous")`；不存在运行时「查不到再从 ECDICT 查并写回」的链路。

仅含词形关系的展示释义可读取本地原形补全，规则见 [data-model.md §7.1](data-model.md#71-仅含词形关系的释义补全)；此过程不调用在线 Enrich、不写回词典。

本地查询不受网络与 API 调用配额影响，大批量导入和复习无需等待第三方词典。外部访问仅限下节的 V2 在线链路。

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

MVP 未接入在线补充时，本地未命中一律创建仅含 headword 的最小词条，释义可由用户手动补充；不会运行时回查 CSV。

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

具体实现是 `FreeDictionaryAPIProvider`（实现上述两个方法）；未来增加 Cambridge、Oxford、Collins 等作为补充源时，只需新增 Provider，生词业务无需修改。**ECDICT 不是 Provider**，导入机制见第 1 节。

## 4. 多来源合并原则

Enrich 只补空缺，不覆盖已有 ECDICT 字段；Lookup 创建在线词条后同样标记 `source`。数据源各自提供的内容见 [overview.md §5](overview.md#5-数据源分工)，字段结构见 [data-model.md](data-model.md)，本文不另列 schema。

## 5. 阶段边界（词典相关能力）

阶段与实施顺序统一见 [Roadmap](../product/roadmap.md#1-版本总览)。本文第 2～4 节描述 V2 在线补充；V3 的 AI 释义整理规则见 [data-model.md §6](data-model.md#6-review_meanings词典层的默认复习释义)，AI 不制造词典事实的原则见 [overview.md §3](overview.md#3-三条原则)。
