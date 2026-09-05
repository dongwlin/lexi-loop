# 复习权重与抽样算法（Review Algorithm）

> **`review/algorithm.md` 是权重公式的唯一权威定义位置。** 本文档包含复习算法的全部数学：weight、mastery、加权不放回抽样、streak、overdue、new_word_bonus。
> [prd.md](../product/prd.md) 只描述这些概念的产品语义，不含公式；公式在别处只能引用、不复制。
> 产品上的两种复习形态（随机复习 / 到期复习）与版本边界见 [roadmap.md](../product/roadmap.md)。

## 1. 权重不落库，动态计算

`weight` 默认动态计算，不作为持久化字段存库。权重包含时间因素：同一个词今天算出 `weight = 4`，三天后可能已经是 6，落库的固定值会失真。

数据库保存的是 `encounter_count`、复习记录、`current_streak`、`last_reviewed_at` 等原始数据（字段见 [dictionary/data-model.md](../dictionary/data-model.md)），`weight` 在使用时按需计算。

## 2. 为什么不用单一累计公式

累计计数公式（如 `encounter + forget × 2 - remember × 0.5`）不适合第一版：一个词早期忘过 20 次、最近连续记得 15 次，累计 `forget_count` 仍会把它推成高权重，但它实际可能已经掌握。因此权重按因素拆开计算：

```text
基础权重 + 遇到频率 + 当前困难度 + 连续表现修正 + 时间因素
```

## 3. 第一版权重公式

```text
weight = base
       + encounter_factor
       + difficulty_factor
       + streak_adjustment
       + overdue_factor
       + new_word_bonus
```

**base**：恒为 1。

**encounter_factor**：`log2(encounter_count + 1)`，效果：

```text
遇到 1 次  ≈ 1
遇到 3 次  ≈ 2
遇到 7 次  ≈ 3
遇到 15 次 ≈ 4
```

避免出现 50 次的高频词垄断抽池。

**difficulty_factor**：先计算整体正确率 `remember_rate = remember_count / review_count`，再取 `difficulty = 1 - remember_rate`。

例如正确率 90% → difficulty = 0.1；正确率 30% → difficulty = 0.7。`difficulty_factor = difficulty × 4`。

**streak_adjustment**：连续忘记显著提权，连续记得较多则略降权：

- `current_streak < 0` → 加上 `-current_streak`（连续忘记 2 次 +2，3 次 +3）
- `current_streak >= 4` → 减去 1

**overdue_factor**（时间因素）：`min(days_since_last_review / 7, 3)`，即每天 +1/7，封顶 3：

```text
0 天          +0
1 天          +0.14
3 天          +0.43
7 天          +1
14 天         +2
21 天及以上    +3
```

新词没有 `last_reviewed_at`，不参与时间因素计算，见第 4 节。

**new_word_bonus**：新词保底，`review_count = 0` 时为 2，否则为 0。见第 4 节。

## 4. 新词与下限

新导入的单词 `review_count = 0` 且 `last_reviewed_at = NULL`，两项都无法直接计算：

- 没有正确率可用，不能 `0 / 0`：`difficulty` 直接取 0.5（未知掌握程度）。
- `days_since_last_review` 不存在：`overdue_factor` 直接取 0，第一次复习之后才开始计算时间因素。
- `new_word_bonus = 2`，避免新词长期抽不到。

即 `review_count == 0` 时：

```text
weight = base + encounter_factor + 0.5×4 + 0 + 2
```

最终权重取 `max(weight, 0.5)`，保证所有词都保留最低出现概率。

## 5. 最终示例

例如：

```text
encounter_count = 5
review_count = 6, remember_count = 2
current_streak = -2
距离上次复习 = 4 天
```

计算：

```text
base = 1
encounter = log2(6) ≈ 2.58
difficulty = 1 - 2/6 ≈ 0.67
difficulty_factor = 0.67 × 4 ≈ 2.68
streak_adjustment = 2（连续忘记 2 次）
overdue_factor ≈ 0.57
new_word_bonus = 0（已复习过）

weight ≈ 1 + 2.58 + 2.68 + 2 + 0.57 + 0 ≈ 8.83
```

该词权重高，会被高频抽中复习。

## 6. mastery_score

`mastery_score`（掌握程度）与 `review_weight`（复习优先级）是两个指标，前者衡量「这个词我掌握了吗」，后者衡量「这个词现在值不值得抽出来复习」。一个词 `mastery = 80%`，但两个月没有复习，`review_weight` 仍然可能升高——因为时间因素。产品语义见 [prd.md](../product/prd.md)。

`mastery_score` 第一版直接从计数推导、不落库，建议平滑处理：

```text
mastery = (remember_count + 1) / (review_count + 2) × 100
```

相当于给每个词一个初始 50% 的未知状态：还没复习 → 1/2 = 50%，第一次记得 → 2/3 ≈ 67%，第一次忘记 → 1/3 ≈ 33%。比直接给 100% 或 0% 更稳定。

## 7. 累计正确率的局限

累计正确率的局限：一个词早期全忘、近期全对，累计率可能只有 67%，但用户最近已经掌握。

V2 改为统计最近 N 次结果，或对越新的结果赋予更高权重（指数衰减）。见第 9 节。

## 8. 加权随机抽样

假设 A weight = 1、B weight = 3、C weight = 6，总权重 10，单次抽取概率 A 10%、B 30%、C 60%。

一轮采用 weighted sampling without replacement：抽到 C 后将其从候选集合移除，剩下 A 和 B 继续抽，直到抽够 30 个。

不要用 `ORDER BY weight DESC LIMIT 30` 的方式取最高权重的前 30 个——这会导致困难词永远出现、熟悉词永远消失。高权重只表示更容易出现，不保证一定出现，抽样必须保留随机性。

一轮固定抽取一批（不中途换词）的产品规则与示例见 [frontend/review-flow.md](../frontend/review-flow.md)。

## 9. 版本边界（算法演进）

```text
MVP Weight
↓
Recent-N / 指数衰减
↓
FSRS / 到期复习
```

### MVP

使用 `encounter_count`、`remember_count`、`forget_count`、`current_streak`、`last_reviewed_at` 足够（即本文档第 3～8 节的公式）。

复习存在两种形态（产品语义见 [roadmap.md](../product/roadmap.md)），第一版只实现 A：

- **A. 随机复习**：用户指定「我现在想背 30 个」，系统加权随机抽取。第一版实现。
- **B. 到期复习**：系统判断某些词已接近遗忘临界点，提示「今天有 23 个单词建议复习」。传统 SRS 形态，第一版不做。

### V2

加入最近 N 次复习表现统计、对越新结果加权（指数衰减），评估「到期复习」提示。数据来源是逐次的 `review_items`（[review/data-model.md](data-model.md)）。

### V3

再评估 FSRS 这类完整的间隔重复算法，第一版不引入。

> 改公式时，同步更新本文档第 3、4、5 节（公式、新词规则与最终示例），避免三处不一致。
