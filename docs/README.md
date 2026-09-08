# LexiLoop 文档中心

## 阅读范围

阶段状态见 [Roadmap](product/roadmap.md)：MVP 已完成，后续开发从现行基线继续。首次了解技术仍先读 [系统架构总览](architecture/overview.md)。

| 材料 | 使用方式 |
| --- | --- |
| `product/prd.md`、架构、领域、API、页面与部署文档 | 现行产品和实现约束；文中明确标注 V2 / V3 的内容属于未来设计 |
| `product/roadmap.md` | 阶段状态与后续规划的唯一入口 |
| `specs/` | 工程规范；按需能力和认证等后续阶段规范不代表当前已实现 |
| 应用与共享包 README | 本地运行、验证和实现入口，不另建阶段待办 |
| [archive/](archive/README.md) | 已失效的阶段计划，只供追溯 |
| [agent-log/](agent-log/README.md) | 当时的工作记录，历史“当前 / 待办”不能当作今天的状态 |

LexiLoop（词环）：英语生词复习系统。本文档目录回答「要找某类内容去哪篇」，结构为**少量上层文档 + 按领域拆开的专项设计文档**，遵循「小而权威、明确引用、单一真相源」：同一份 schema / 公式 / 决策只在一个文件里权威定义，其它地方只能引用。

## 项目架构：Monorepo

目录布局与应用职责见 [系统架构总览 §5](architecture/overview.md#5-项目架构monorepo)，部署脚本、镜像构建与词典数据依赖见 [发布流程](deploy/release.md)。

项目 clone 后的标准初始化：

- `pnpm install`（依赖）
- `deploy/dict/fetch.sh`（词典数据，幂等可重跑；产物 gitignore 忽略）
- `go mod download`（后端按需）

## 目录导航

产品、领域设计与工程规范按下方「权威关系」表定位；其他材料入口：

| 目录 | 内容 |
| --- | --- |
| [openapi/](openapi/) | [JSON](openapi/openapi.json) / [YAML](openapi/openapi.yaml) spec，由 `lexi-loop openapi` 离线生成，勿手改；契约以 `api/*` 为准 |
| [brand/](brand/) | 命名文档、Logo 主图与概念图 |
| [decisions/](decisions/README.md) | 冻结决策索引 |
| [archive/](archive/README.md) | 历史材料与归档规则 |
| [agent-log/](agent-log/README.md) | 工作记录规则与按月日志 |

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
| 系统总体架构 / Monorepo 布局 | [architecture/overview.md](architecture/overview.md) |
| 四表关系 | [architecture/data-model.md](architecture/data-model.md) |
| `dictionary_entries` / `user_words` | [dictionary/data-model.md](dictionary/data-model.md) |
| 词典方案 / 数据源分工 | [dictionary/overview.md](dictionary/overview.md) |
| lemma / normalize | [dictionary/normalization.md](dictionary/normalization.md) |
| Lookup / Enrich | [dictionary/enrichment.md](dictionary/enrichment.md) |
| `review_sessions` / `review_items` | [review/data-model.md](review/data-model.md) |
| 权重 / mastery / sampling | [review/algorithm.md](review/algorithm.md) |
| 页面与状态机 | [frontend/review-flow.md](frontend/review-flow.md) |
| Word API | [api/words.md](api/words.md) |
| Review API | [api/reviews.md](api/reviews.md) |
| Meta API（版本信息 / 词典导入进度） | [api/meta.md](api/meta.md) |
| 发布流程 / 镜像构建 / 词典数据依赖 | [deploy/release.md](deploy/release.md) |
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

文档治理与冲突裁决见根目录 [AGENTS.md](../AGENTS.md)；Agent 工作前阅读近期记录，完成变更后按 [日志规则](agent-log/README.md) 记录。
