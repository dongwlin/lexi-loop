# 系统架构总览（Architecture Overview）

> 技术设计的入口文档：开发者第一次了解项目技术设计时先读它。本文档只描述系统级结构、领域边界与运行时形态，不放具体字段、公式或 endpoint——它们分别在各自的专项文档。

## 1. 系统分层

```text
Frontend（Vue 页面，完整路由见 frontend/review-flow.md）
   ↓ HTTP JSON
HTTP API（internal/handler）
   ↓
Word Service ──────────────→ Dictionary Service
（导入 / 列表 / 编辑释义 / 删除）   （查词条 / 增强，见 dictionary/overview.md）
   ↓
Review Service（开始一轮 / 提交结果 / 统计）
   ↓
PostgreSQL
```

- **Word Service**（`user_words` 领域）：用户生词的导入、查询、编辑自定义释义、软删除；导入时需要词典信息，因此依赖 Dictionary Service。
- **Review Service**（复习领域）：创建一轮、加权抽样、提交逐次结果、汇总统计，读写 `user_words` 的学习字段与 `review_sessions` / `review_items`。
- **Dictionary Service**（词典领域）：查询本地词典库 `dictionary_entries`，V2 起承担在线 Lookup / Enrich。业务层不直接接触数据源，只通过 Service / Repository。

## 2. 三个领域边界

```text
dictionary = 客观词典：语言事实（词条、音标、释义），全局共享，任何用户不可直接修改。
word       = 用户生词：用户对某个词的个人学习状态与自定义释义。
review     = 复习领域：一轮一轮的复习会话与逐次的复习结果。
```

一句话：`dictionary` 是字典本身，`word` 是「我的词」，`review` 是「我背词的过程」。三者数据相互独立，任何字段只属于其中一个领域。

## 3. 运行期数据血缘

四张核心表构成导入与复习的主链路，关系级模型见 [architecture/data-model.md](data-model.md)：

```text
dictionary_entries
       ↓ 一个词条 → 用户的词
user_words
       ↓ 每个词每轮的一次作答
review_items
       ↑ 一次作答属于某一轮
review_sessions
```

## 4. 词典运行时形态：本地优先

`dictionary_entries` 就是本地词典库本体：ECDICT 数据随镜像内置（构建期 COPY），serve 启动后在 server 进程内异步守卫式导入（导入期数据源，进度经 Meta API 暴露；手动 `import-ecdict` 命令保留），运行时不读 CSV、也没有「查不到再从 ECDICT Lookup」的链路。MVP 的运行时查询全部走本地、全程离线。

V2 起才有在线补充，分两条独立链路：**Lookup**（本地没有这个词条时在线创建）与 **Enrich**（词条已存在但缺增强字段时补全）。入口见 [dictionary/overview.md](../dictionary/overview.md)。

## 5. 项目架构：Monorepo

项目采用 **pnpm monorepo** 架构管理，顶层使用 pnpm 作为脚本管理器。

### 5.1 目录结构

```text
lexi-loop/
├─ apps/
│  ├─ web/              前端应用（Vue）
│  └─ server/           后端应用（Go）
├─ packages/
│  └─ api-client/       API 客户端（Orval 从 docs/openapi/ 自动生成）
├─ pnpm-workspace.yaml  pnpm 工作区配置
├─ package.json         根 package.json（脚本入口）
└─ docs/                项目文档
```

### 5.2 各部分职责

| 目录 | 说明 |
|------|------|
| `apps/web` | Vue 前端应用，独立构建与部署 |
| `apps/server` | Go 后端服务，独立构建与部署 |
| `packages/api-client` | TypeScript API 客户端，通过 OpenAPI 文档自动生成，供 `apps/web` 使用 |

### 5.3 技术选型

- 后端 Go（根 `main.go` + Cobra 子命令：`serve`、`migrate`、`import-ecdict`、`version`），数据库 PostgreSQL，前端 Vue。
- 后端实体主键统一使用 UUID v7，由应用层生成；API 按字符串传输。
- 后端工程结构见 [backend/structure.md](../backend/structure.md)。
- MVP 单用户、无鉴权体系，`user_words` 暂不含 `user_id`。

## 6. 文档导览

| 想了解的部分 | 文档 |
| --- | --- |
| 产品需求与 MVP 边界 | [product/prd.md](../product/prd.md) |
| 版本阶段规划 | [product/roadmap.md](../product/roadmap.md) |
| 四表关系级模型 | [architecture/data-model.md](data-model.md) |
| 页面与交互 | [frontend/review-flow.md](../frontend/review-flow.md) |
| Word API | [api/words.md](../api/words.md) |
| Review API | [api/reviews.md](../api/reviews.md) |
| Meta API（版本信息） | [api/meta.md](../api/meta.md) |
| 复习数据模型（sessions / items） | [review/data-model.md](../review/data-model.md) |
| 权重 / 掌握度 / 抽样算法 | [review/algorithm.md](../review/algorithm.md) |
| 词典子系统入口 | [dictionary/overview.md](../dictionary/overview.md) |
| 词典与生词表字段 | [dictionary/data-model.md](../dictionary/data-model.md) |
| 词形归一 | [dictionary/normalization.md](../dictionary/normalization.md) |
| Lookup / Enrich | [dictionary/enrichment.md](../dictionary/enrichment.md) |
| Go 工程结构 | [backend/structure.md](../backend/structure.md) |
| 项目名称 / Slogan | [brand/naming.md](../brand/naming.md) |
