# 词典与生词数据模型（Dictionary Data Model）

> **`dictionary_entries` 与 `user_words` 字段的唯一权威定义在本文件。** 它同时回答「为什么拆两张表」与「复习时释义怎么取值」。别处（包括 review 文档）需要引用这些字段时只引用、不复制 schema。
> 复习相关两张表（`review_sessions` / `review_items`）的字段在 [review/data-model.md](../review/data-model.md) 定义，四表关系级地图见 [architecture/data-model.md](../architecture/data-model.md)。

## 1. 为什么拆两张表：词典数据 ≠ 用户学习数据

词典字段（音标、释义等）和学习字段（遇词次数、复习结果等）不要放进同一张表，应拆成两张表：

```text
dictionary_entries   负责客观词典数据
user_words           负责用户的个人学习状态
```

系统内不存在早期设计里 `words` 这种「词典事实 + 学习状态」混在一行的概念。

## 2. dictionary_entries 表

`dictionary_entries` 记录一个英语单词的词典事实：单词本身是什么、有哪些释义。这是全局共享的客观数据，任何用户都不应直接修改它。

```text
dictionary_entries
id                    UUID v7，应用层生成，PostgreSQL `uuid`
headword              唯一（UNIQUE）
lemma                 词条原形（一般等于 headword，词形归一时指向它）
phonetic_uk
phonetic_us
raw_meanings          JSONB，原始词典义项（结构见第 4 节）
review_meanings       JSONB，词典层的默认复习释义，可空（见第 6 节）
exchange              JSONB，词形变化
frequency             JSONB，词频数据
tags                  JSONB，考试 / 语料标签
source                数据来源
source_version        数据版本
created_at
updated_at
```

示例：

```text
headword: ambiguous
phonetic_uk: /æmˈbɪɡjuəs/
raw_meanings: [ { "pos": "adjective", "translations": ["模棱两可的", "含糊不清的", "有歧义的"] } ]
```

## 3. user_words 表

`user_words` 记录用户对这个单词的学习状态，与词典数据完全分离。

```text
user_words
id                     UUID v7，应用层生成，PostgreSQL `uuid`
dictionary_entry_id    UUID，外键 → dictionary_entries.id；UNIQUE（MVP 单用户下一个词条至多一行；多用户时改为 UNIQUE(user_id, dictionary_entry_id)，见 architecture/data-model.md 第 3 节）
encounter_count
review_count
remember_count
forget_count
current_streak
last_reviewed_at
custom_review_meaning  JSONB，用户个人自定义复习释义，NULL 表示未自定义
deleted_at             软删除标记，NULL 表示未删除（见第 9 节）
created_at
updated_at
```

学习字段的语义约定：

- `encounter_count`：首次导入为 1，重复导入 +count。不同词形归一后累计在原形词条上（见 [normalization.md](normalization.md)）。
- `review_count`：总复习次数，应满足 `review_count = remember_count + forget_count`。
- `current_streak`：正数 = 连续记得，负数 = 连续忘记。例如 `3` 表示连续 3 次记得，`-2` 表示连续 2 次没记住。复习提交时更新（规则见 [api/reviews.md](../api/reviews.md)）。
- `mastery`、复习权重等指标从这些字段动态计算、不落库（见 [review/algorithm.md](../review/algorithm.md)）。

## 4. 释义结构（raw_meanings）

ECDICT 导入解析 `translation` 时，将字面量 `\n` 还原为换行，再按行识别词性与义项；实际换行同样支持。

不建议把释义只保存成一段 `TEXT`，`raw_meanings` 推荐使用结构化 JSON：

```json
[
  {
    "pos": "adjective",
    "translations": [
      "模棱两可的",
      "含糊不清的",
      "有歧义的"
    ]
  }
]
```

第二阶段引入 Free Dictionary API 后（Lookup 在线分支 + Enrich），英文释义按词性挂到对应义项上，这样以后可以按词性展示，而不用重新解析纯文本。多来源合并原则见 [enrichment.md](enrichment.md)。

## 5. 原则：原始释义 ≠ 复习释义（raw vs review）

系统应该区分 `raw_meanings`（完整词典内容）和 `review_meanings`（复习时只展示最重要的几个意思）。

例如 `run` 的完整释义包含 v. 跑 / 运行 / 经营 / 流动 / 延伸 / 参加竞选等义项，以及 n. 跑步 / 行程 / 趋势 / 连续等义项，但复习释义只展示：

```text
run
v.
① 跑
② 运行
③ 经营
```

并提供「查看全部释义」选项，避免释义过多增加记忆负担。

## 6. review_meanings：词典层的默认复习释义

`review_meanings` 是挂在 `dictionary_entries` 上的精简释义（JSONB），属于词典数据的展示层派生结果，可空。它由 AI 或后续工具从 `raw_meanings` 整理生成（V3，AI 只整理不制造事实，见 [overview.md](overview.md)），以后整理策略改变时可以重新生成，不会破坏原始词典数据。

`review_meanings` 为空时（例如第一版还没有 AI 整理），复习页直接展示 `raw_meanings`，保证用户始终看得到释义。

## 7. 用户自定义：只写 user_words

自动词典可能出现的偏差：释义与个人理解不一致、某个义项不重要、考研语境义项缺失、用户希望使用自己的记忆方式。因此需要支持「编辑复习释义」。

但用户修改**不能改动 `dictionary_entries`**——那是客观公共词典数据。用户的个人表达应写入 `user_words.custom_review_meaning`（NULL 表示未自定义）。

复习展示按以下顺序取首个非 NULL 的释义（`effective_review_meaning`）：

```text
effective_review_meaning =
    user_words.custom_review_meaning
    ?? dictionary_entries.review_meanings
    ?? raw_meanings（未整理前的直接展示）
```

写接口只允许 PATCH `user_words.custom_review_meaning`（[api/words.md](../api/words.md)）。

## 8. 发音、词频与标签

- **发音**：音标优先使用本地词典数据。如果在线 Provider 存在音频，可以保存 `audio`（第二阶段起，见 [enrichment.md](enrichment.md)），但第一版音频不是核心功能。
- **词频**：ECDICT 提供的词频信息（`frequency`、`bnc_frequency`、`coca_frequency`）可以保留；离线导入时柯林斯词频星级（`collins`）与牛津三千核心词标记（`oxford`）同属客观语料元数据，一并存入 `frequency`（0 与空串视为无数据、不写入键）。未来复习算法可以考虑「个人 `encounter_count` + 客观 `frequency`」，但客观词频只能作为弱信号——复习权重主要依赖用户自己遇到多少次、是否记得、多久没复习，而不是公共语料库词频（[review/algorithm.md](../review/algorithm.md)）。

## 9. 软删除（deleted_at）

删除生词采用软删除：置 `user_words.deleted_at = now()`，不物理删行。

- 生词库列表与加权抽样统一过滤 `deleted_at IS NULL`，被删除的词不再出现。
- `review_items.user_word_id` 仍指向保留的行，历史复习记录与历史 session 统计保持完整，不受删除影响。
- `dictionary_entries` 是共享词典数据，不随个人删除。
- 重新导入同一词条时，恢复这条 `user_words`（`deleted_at` 置 NULL）并继续累计 `encounter_count`，而不是新建一行、也不会丢失历史。

不做物理删除（CASCADE 会连带抹掉复习历史，RESTRICT 会让有复习记录的词删不掉），软删除同时避开两个问题。API 语义见 [api/words.md](../api/words.md)。

## 10. 第一版 MVP 的最小词典要求

MVP 基础信息范围见 [PRD](../product/prd.md#11-第一版-mvp)；未收录词的最小词条兜底见 [Lookup 链路](enrichment.md#2-两条链路的划分)，无法确定 lemma 时保留原词的规则见 [词形归一](normalization.md)。
