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

## 3. 运行期数据血缘

导入与复习通过四张核心表关联；关系、基数与各表职责见 [关系级数据模型](data-model.md)。字段分别由词典与复习领域维护。

## 4. 词典运行时形态：本地优先

MVP 运行时只查询本地 `dictionary_entries`，不依赖外部词典；ECDICT 是导入期数据源。词典方案见 [dictionary/overview.md](../dictionary/overview.md)，导入与在线补充机制见 [enrichment.md](../dictionary/enrichment.md)。

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
├─ deploy/             发布脚本、Caddy 与容器入口；dict/fetch.sh 获取 pinned ECDICT
├─ Dockerfile          多阶段构建（web + server + 内置词典）
├─ pnpm-workspace.yaml  pnpm 工作区配置
├─ package.json         根 package.json（脚本入口）
└─ docs/                项目文档
```

### 5.2 各部分职责

`apps/web` 与 `apps/server` 可独立构建、部署；`packages/api-client` 从 OpenAPI 生成 TypeScript 客户端，供前端使用。发布操作见 [部署文档](../deploy/release.md)。

### 5.3 技术选型

- 后端 Go（根 `main.go` + Cobra 子命令：`serve`、`migrate`、`import-ecdict`、`version`），数据库 PostgreSQL，前端 Vue。
- 后端实体主键统一使用 UUID v7，由应用层生成；API 按字符串传输。
- 后端工程结构见 [backend/structure.md](../backend/structure.md)。
- MVP 单用户、无鉴权体系，`user_words` 暂不含 `user_id`。

## 6. 文档导览

完整导航与权威关系见 [文档中心](../README.md#权威关系)；继续阅读可按领域选择 [词典](../dictionary/overview.md)、[复习数据](../review/data-model.md)、[页面交互](../frontend/review-flow.md) 或 [Go 工程结构](../backend/structure.md)。
