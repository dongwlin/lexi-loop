# LexiLoop 文档中心

LexiLoop（词环）：英语生词复习系统。本文档目录回答「要找某类内容去哪篇」，结构为**少量上层文档 + 按领域拆开的专项设计文档**，遵循「小而权威、明确引用、单一真相源」：同一份 schema / 公式 / 决策只在一个文件里权威定义，其它地方只能引用。

## 项目架构：Monorepo

项目采用 **pnpm monorepo** 架构管理，顶层使用 pnpm 作为脚本管理器：

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

## 目录结构

```text
docs/
├─ README.md                  ← 本文档（导航）
├─ agent-log/
│  ├─ README.md              Agent 工作记录规则
│  └─ YYYY-MM.md             按月分文件、月内按时间逆序的工作记录
├─ product/
│  ├─ prd.md                  产品需求（现在是什么、业务规则、MVP 边界）
│  └─ roadmap.md              阶段规划（以后怎么长：MVP → V2 → V3）
├─ architecture/
│  ├─ overview.md             系统总体架构（入口，先读这篇）
│  └─ data-model.md           四表关系级模型
├─ frontend/
│  └─ review-flow.md          页面结构与复习交互（含状态机、键盘、恢复 UI）
├─ api/
│  ├─ words.md                Word API 契约
│  ├─ reviews.md              Review API 契约
│  └─ meta.md                 Meta API（版本信息）契约
├─ dictionary/
│  ├─ overview.md             词典子系统入口
│  ├─ data-model.md           dictionary_entries / user_words 字段
│  ├─ normalization.md        词形归一
│  └─ enrichment.md           Lookup / Enrich / 多来源 Merge
├─ review/
│  ├─ data-model.md           review_sessions / review_items 字段与生命周期
│  └─ algorithm.md            权重 / mastery / 抽样（唯一权威公式）
├─ backend/
│  └─ structure.md            Go 工程结构（领域 → package）
├─ openapi/
│  ├─ openapi.json            OpenAPI 3.1 spec（由 `lexi-loop openapi` 离线生成，勿手改；契约权威仍是 api/*）
│  └─ openapi.yaml            同上（YAML 格式）
├─ specs/
│  ├─ backend/
│  │  ├─ Go 技术栈.md          后端框架 / 数据库 / CLI / 日志 / 测试等选型
│  │  ├─ Go 单体应用架构规范.md  后端分层 / Domain / Repo / Service / Handler / 事务 / 缓存
│  │  ├─ Go 测试规范.md          TDD / 分层测试 / 测试替身 / CI
│  │  └─ HTTP API 设计规范.md    响应结构 / 错误码 / 分页 / 路由版本
│  └─ frontend/
│     ├─ 前端技术栈.md                    前端框架 / 构建工具 / 依赖选型
│     ├─ 前端应用架构规范.md              目录结构 / 依赖方向 / 状态归属 / API 分层
│     ├─ 前端 API 与认证集成规范.md       Bearer Token / Refresh / 并发刷新 / 请求重放
│     ├─ 前端交互与可访问性规范.md        WCAG 2.2 AA / 键盘 / 焦点 / 表单 / 动画
│     ├─ 前端测试规范.md                  Vitest / Testing Library / MSW / Storybook / Playwright
│     ├─ Vue 组件设计系统方案.md           Token / 色彩 / 组件配方 / 主题
│     └─ Vue 性能与缓存优化.md            KeepAlive / 预取 / 虚拟列表 / 缓存策略
├─ brand/
│  ├─ naming.md               项目名称 / Slogan
│  ├─ logo.png                Logo 主图
│  └─ logo概念图.png           Logo 概念图
└─ decisions/
   └─ README.md               冻结决策索引
```

## 依赖关系

```text
                 product/prd.md
                       │
              ┌────────┴─────────┐
              ↓                  ↓
 architecture/overview       product/roadmap
              │
      ┌───────┼─────────┐
      ↓       ↓         ↓
 dictionary  review   frontend
      │       │
      └───┬───┘
          ↓
         api
          ↓
 backend/structure

brand/naming.md  ← 品牌旁路，不依赖其它文档
```

## 权威关系

找内容先定位下表，避免在多个文件重复维护同一份 schema / 公式：

| 内容 | 唯一权威文档 |
| --- | --- |
| 产品目标 / MVP | [product/prd.md](product/prd.md) |
| 阶段规划 | [product/roadmap.md](product/roadmap.md) |
| 系统总体架构 | [architecture/overview.md](architecture/overview.md) |
| Monorepo 架构 | [architecture/overview.md](architecture/overview.md) |
| 四表关系 | [architecture/data-model.md](architecture/data-model.md) |
| `dictionary_entries` | [dictionary/data-model.md](dictionary/data-model.md) |
| `user_words` | [dictionary/data-model.md](dictionary/data-model.md) |
| lemma / normalize | [dictionary/normalization.md](dictionary/normalization.md) |
| Lookup / Enrich | [dictionary/enrichment.md](dictionary/enrichment.md) |
| `review_sessions` / `review_items` | [review/data-model.md](review/data-model.md) |
| 权重 / mastery / sampling | [review/algorithm.md](review/algorithm.md) |
| 页面与状态机 | [frontend/review-flow.md](frontend/review-flow.md) |
| Word API | [api/words.md](api/words.md) |
| Review API | [api/reviews.md](api/reviews.md) |
| Meta API（版本信息） | [api/meta.md](api/meta.md) |
| Go package | [backend/structure.md](backend/structure.md) |
| Go 技术栈 | [specs/backend/Go 技术栈.md](specs/backend/Go%20技术栈.md) |
| Go 单体架构规范 | [specs/backend/Go 单体应用架构规范.md](specs/backend/Go%20单体应用架构规范.md) |
| Go 测试规范 | [specs/backend/Go 测试规范.md](specs/backend/Go%20测试规范.md) |
| HTTP API 设计规范 | [specs/backend/HTTP API 设计规范.md](specs/backend/HTTP%20API%20设计规范.md) |
| 前端技术栈 | [specs/frontend/前端技术栈.md](specs/frontend/前端技术栈.md) |
| 前端应用架构规范 | [specs/frontend/前端应用架构规范.md](specs/frontend/前端应用架构规范.md) |
| 前端 API 与认证集成规范 | [specs/frontend/前端 API 与认证集成规范.md](specs/frontend/前端%20API%20与认证集成规范.md) |
| 前端交互与可访问性规范 | [specs/frontend/前端交互与可访问性规范.md](specs/frontend/前端交互与可访问性规范.md) |
| 前端测试规范 | [specs/frontend/前端测试规范.md](specs/frontend/前端测试规范.md) |
| 设计系统方案 | [specs/frontend/Vue 组件设计系统方案.md](specs/frontend/Vue%20组件设计系统方案.md) |
| Vue 性能与缓存优化 | [specs/frontend/Vue 性能与缓存优化.md](specs/frontend/Vue%20性能与缓存优化.md) |
| 项目名称 / slogan | [brand/naming.md](brand/naming.md) |
| 冻结决策 | [decisions/README.md](decisions/README.md) |

## 快速定位示例

```text
我要改权重            → review/algorithm.md
我要改复习接口        → api/reviews.md
我要改版本接口        → api/meta.md
我要看接口 spec 产物  → openapi/openapi.yaml（由 lexi-loop openapi 离线生成，勿手改）
我要改词形解析        → dictionary/normalization.md
我要改数据库表        → dictionary/data-model.md / review/data-model.md
我要改页面            → frontend/review-flow.md
```

文档治理规则（含权威关系与冲突裁决）见仓库根目录的 [AGENTS.md](../AGENTS.md)。

Agent 在任务开始前可从 [agent-log/](agent-log/) 了解近期改动，完成仓库变更后按 [agent-log/README.md](agent-log/README.md) 记录。该日志只用于追溯工作，不是设计结论的权威来源。
