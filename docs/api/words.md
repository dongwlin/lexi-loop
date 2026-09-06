# Word API（生词）

> 生词类 API 沿用 `/api/v1/words` 的用户视角命名，操作对象是「用户生词」（`user_words` 行），释义等词典信息由服务端关联 `dictionary_entries` 后返回。
> `user_words` / `dictionary_entries` 的字段定义见 [dictionary/data-model.md](../dictionary/data-model.md)；导入时的词形归一与查词条行为分别见 [dictionary/normalization.md](../dictionary/normalization.md) 与 [dictionary/enrichment.md](../dictionary/enrichment.md)。
> 所有接口遵循 [HTTP API 设计规范](../specs/backend/HTTP%20API%20设计规范.md) 的标准响应结构。

## 1. 接口总览

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| POST | `/api/v1/words/import` | 批量导入生词（累计遇词次数） |
| GET | `/api/v1/words` | 生词库列表（分页 / 搜索） |
| GET | `/api/v1/words/:id` | 单词详情 |
| PATCH | `/api/v1/words/:id` | 更新复习释义（用户自定义） |
| DELETE | `/api/v1/words/:id` | 删除生词（软删除） |

路径参数 `:id` 是 `user_words.id`，使用 UUID v7；JSON 中按字符串传输。

## 2. 导入

```text
POST /api/v1/words/import
```

请求（前端已做 trim / lowercase / 空行移除，并把重复单词聚合成 count，见 [frontend/review-flow.md](../frontend/review-flow.md)）：

```json
{
  "words": [
    { "word": "ambiguous", "count": 1 },
    { "word": "constrain", "count": 2 },
    { "word": "derive", "count": 1 }
  ]
}
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "encounters": 4,
    "created": 2,
    "updated": 1,
    "items": [
      { "word": "ambiguous", "count": 1, "result": "created" },
      { "word": "constrain", "count": 2, "result": "updated" },
      { "word": "derive", "count": 1, "result": "created" }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `encounters` | int | 本次计入的遇词次数合计 |
| `created` | int | 新建的 `user_words` 词条数 |
| `updated` | int | 已存在、遇词次数被累计的词条数 |
| `items` | array | 逐词结果，与聚合后的输入单词一一对应（按首次出现顺序） |
| `items[].word` | string | 归一后的单词（服务端再做一次 trim / lowercase） |
| `items[].count` | int | 该词本次计入的遇词次数 |
| `items[].result` | string | `created`（本次新建）/ `updated`（已存在并累计，含软删除恢复） |

`items` 是导入反馈（review-flow §3「ambiguous 新增 / constrain 已存在 +2」）的数据来源，为单词粒度：`result` 表达该单词在本次导入前是否已在用户词库中（`created` = 本次新建，`updated` = 已存在并累计，含软删除恢复）。`created` / `updated` 聚合统计则是词条行粒度（`user_words` 行数），两者计量单位不同——归一到同一新建词条的多个输入词形在 `items` 中都记 `created`。

### 服务端逻辑

```text
for item in words:
    entry := DictionaryService.Lookup(item.word)
        // 词形归一 + 查询本地词典库 dictionary_entries
        // Lookup 的兜底行为（本地未命中 / 最小词条 / V2 在线分支）见 dictionary/enrichment.md
    upsert user_words where dictionary_entry_id = entry.id
        encounter_count += item.count
        deleted_at = NULL   // 重新导入时恢复已软删除的词条
```

首次导入创建 `user_words`，`encounter_count = count`。数据库层用 `INSERT ... ON CONFLICT (dictionary_entry_id) DO UPDATE` 批量完成。

### 重新导入恢复语义

被软删除（`deleted_at` 非空）的词条再次导入时，不新建一行，而是恢复该行（`deleted_at` 置 NULL）并继续累计 `encounter_count`，历史复习记录不丢失（数据层语义见 [dictionary/data-model.md](../dictionary/data-model.md) 的软删除一节）。

## 3. 获取生词列表

```text
GET /api/v1/words?page=1&pageSize=50&search=amb
```

请求参数：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | int | 否 | 1 | 页码，从 1 开始 |
| `pageSize` | int | 否 | 20 | 每页条数，上限 100 |
| `search` | string | 否 | — | 搜索关键词，匹配单词或释义 |

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "list": [
      {
        "id": "01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001",
        "word": "ambiguous",
        "phonetic": "/æmˈbɪɡjuəs/",
        "effectiveReviewMeaning": [
          { "pos": "adjective", "translations": ["模棱两可的", "含糊不清的"] }
        ],
        "encounterCount": 3,
        "reviewCount": 2,
        "rememberCount": 1,
        "forgetCount": 1,
        "currentStreak": -1,
        "lastReviewedAt": "2026-09-01T10:30:00Z",
        "masteryScore": 33,
        "reviewWeight": 4.62
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 50,
      "total": 427,
      "totalPages": 9,
      "hasMore": true
    }
  }
}
```

列表过滤 `deleted_at IS NULL` 的 `user_words`，被删除的词不再出现。空列表时 `list` 返回 `[]`。

列表项与详情（§4）都附带两个动态计算的派生指标（权重 / mastery 不落库、读取时按当前时间计算，公式权威见 [review/algorithm.md](../review/algorithm.md) §3–§6，冻结决策 D011）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `masteryScore` | int | 掌握程度 0–100：`(remember_count + 1) / (review_count + 2) × 100` 四舍五入取整 |
| `reviewWeight` | number | 复习优先级权重（algorithm.md §3–§4 公式），保留两位小数 |

## 4. 获取单词详情

```text
GET /api/v1/words/:id
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "id": "01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001",
    "word": "ambiguous",
    "phonetic": "/æmˈbɪɡjuəs/",
    "definition": "open to more than one interpretation...",
    "effectiveReviewMeaning": [
      { "pos": "adjective", "translations": ["模棱两可的", "含糊不清的"] }
    ],
    "meaningSource": "custom",
    "encounterCount": 3,
    "reviewCount": 2,
    "rememberCount": 1,
    "forgetCount": 1,
    "currentStreak": -1,
    "lastReviewedAt": "2026-09-01T10:30:00Z",
    "masteryScore": 33,
    "reviewWeight": 4.62
  }
}
```

返回 `user_words` 学习字段 + `dictionary_entries` 词典字段 + `effectiveReviewMeaning`（含是否来自用户自定义的标记），并附带与列表项相同的 `masteryScore` / `reviewWeight`（见 §3）。取值规则见 [dictionary/data-model.md](../dictionary/data-model.md)。

资源不存在时返回：

```json
{
  "code": "BASE.NOT_FOUND.USER",
  "message": "word not found",
  "data": {}
}
```

## 5. 更新复习释义

```text
PATCH /api/v1/words/:id
```

请求：

```json
{
  "customReviewMeaning": [
    { "pos": "adjective", "translations": ["模棱两可的", "含糊不清的"] }
  ]
}
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

- 这个接口只写 `user_words.custom_review_meaning`，**绝不修改 `dictionary_entries`**（词典数据是客观共享数据）。
- 传 `null` 表示清除自定义、回退到词典层的默认复习释义。

## 6. 删除生词（软删除）

```text
DELETE /api/v1/words/:id
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

MVP 采用软删除：置 `user_words.deleted_at = now()`，不物理删除行。

- 生词库列表与加权抽样统一过滤 `deleted_at IS NULL`，被删除的词不再出现。
- `review_items.user_word_id` 仍指向保留的行，历史复习记录与历史 session 统计（`总计 / 记得 / 忘记`）保持完整，不受删除影响。
- `dictionary_entries` 是共享词典数据，不随个人删除。
- 以后重新导入同一词条时，恢复这条 `user_words`（`deleted_at` 置 NULL）并继续累计 `encounter_count`，而不是新建一行、也不会丢失历史（见第 2 节）。

不做物理删除的原因与数据层规则见 [dictionary/data-model.md](../dictionary/data-model.md) 的软删除一节。
