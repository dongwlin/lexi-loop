# 词形归一化（Normalization）

> 本文档定义「用户输入 → 标准词 → lemma」的归一规则：何时该自动归并、何时必须保留原词。将来实现 lemma resolver 时，本文档可直接作为实现规格。

## 1. 归一化的位置

归一发生在导入流程的开头，是 `DictionaryService.Lookup` 的前置步骤：

```text
用户输入（derived / ambiguous / constraints）
↓
Normalize（trim、lowercase、去除空行）
↓
Lemma Resolve（derived → derive，constraints → constraint；无法确定时保留原词，见第 3 节）
↓
查询本地词典库 dictionary_entries / 兜底创建词条（Lookup 分支见 enrichment.md）
↓
创建或更新 user_words（encounter_count + count）
```

（导入的完整业务与 API 见 [api/words.md](../api/words.md)。）

## 2. 为什么需要归一

系统导入生词时，需要识别不同词形属于同一个词。

例如 `derived`、`deriving`、`derive` 应该归一到 `derive`；`constraints` 归一为 `constraint`。

最终 `derive` 的 `encounter_count` 累计为 3，而不是创建三个不同生词。

### 归一规则

- **trim**：去除首尾空格。
- **lowercase**：统一小写后再匹配。
- **Lemma Resolve**：借助 ECDICT 的词形变化表（`exchange`）把多数规则变化（三单、过去式、比较级等）归一到原形：

```text
derived    → derive
deriving   → derive
constraints → constraint
```

## 3. 歧义原则：能确定 → 自动归一；不能确定 → 保留输入

多数规则变化可以借助 ECDICT 的词形变化表确定原形，但部分输入没有唯一答案：

```text
saw → see（动词 see 的过去式）/ saw（名词「锯」）
lay → lie（躺）/ lay（放置）
bound → bind（捆绑）/ bound（跳跃、边界）
better → good / well（比较级）
```

此时遵循最终原则：

```text
能确定 → 自动归一
不能确定 → 保留原词，不做激进归并
```

宁可多一个词条，也不要把用户遇到的词错误合并。无法归一的输入以自身为 headword 创建词条；若它实际是某个词的词形，留待词典数据（ECDICT 的 lemma 字段或 exchange 表）给出明确映射后再归并。

## 4. 归一对学习数据的影响

归一成功的词形，遇词次数累计到原形词条上（`encounter_count` 语义见 [data-model.md](data-model.md)）；保留原词的输入则以自身独立成词条。
