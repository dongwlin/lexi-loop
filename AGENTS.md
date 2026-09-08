# AGENTS.md — lexi-loop

> 当前阶段与后续规划以 [docs/product/roadmap.md](docs/product/roadmap.md) 为准。MVP 已完成，其现行产品规则与冻结决策继续生效；[docs/archive/](docs/archive/README.md) 和 Agent Log 中的旧计划、过渡实现与历史待办仅供追溯，不得作为当前任务清单。

LexiLoop（词环）：英语生词复习系统。本仓库按 pnpm monorepo 布局同时保存代码与文档：`docs/` 为产品与技术文档，`apps/` 为应用代码（`server` 是 Go 后端，`web` 是 Vue 前端），`packages/` 为共享包；目录布局与导航以 [docs/README.md](docs/README.md) 为准。

## 文档地图

```text
PRD（product/prd.md：做什么）
→ architecture（overview 系统结构 → data-model 四表关系）
→ domain design（dictionary/* · review/* · frontend/review-flow.md）
→ API（api/words.md · api/reviews.md · api/meta.md）
→ implementation（backend/structure.md）
→ specs（specs/backend/ — 后端工程规范；specs/frontend/ — 前端工程规范）
```

新人/Agent 先读 [docs/architecture/overview.md](docs/architecture/overview.md)，再按需下沉；导航与权威关系表见 [docs/README.md](docs/README.md)。在 `apps/server` / `apps/web` 内工作时，先读对应子树级指引（与本文档叠加生效）：[apps/server/AGENTS.md](apps/server/AGENTS.md)、[apps/web/AGENTS.md](apps/web/AGENTS.md)。

## 按任务选择入口

| 任务 | 入口与范围 |
| --- | --- |
| 后端实现 | [apps/server/AGENTS.md](apps/server/AGENTS.md) 与 [运行说明](apps/server/README.md) |
| 前端实现 | [apps/web/AGENTS.md](apps/web/AGENTS.md) 与 [运行说明](apps/web/README.md) |
| 共享 API 客户端 | [packages/api-client/README.md](packages/api-client/README.md) 与 [前端 API 集成规范](docs/specs/frontend/前端%20API%20与认证集成规范.md) |
| 构建、词典数据与发布 | [docs/deploy/release.md](docs/deploy/release.md)；发布脚本会提交与推送，执行前确认任务包含发布 |
| 文档修改 | 下方权威表与 [docs/README.md](docs/README.md)；变更规则时同步显式引用，纯排版不改规则 |

## 契约与验证

- API 契约变更按 `docs/api/*` → 后端 Handler / DTO → `docs/openapi/` → `packages/api-client` → 前端调用方的顺序同步。先在 `apps/server` 执行 `go run . openapi`，再在仓库根执行 `pnpm -F @lexi-loop/api-client generate`；生成流程见 [API 客户端说明](packages/api-client/README.md#重新生成)。OpenAPI、派生 spec 与 `src/generated/` 不手改。
- 根 `package.json` 的 `build` 只构建 web，`typecheck` / `test:run` 覆盖 pnpm workspace，不包含 Go 后端。后端检查进入 `apps/server` 执行；命令与环境前提见子树指引。
- 根据改动范围执行相应检查；完整门禁以 [.github/workflows/ci.yml](.github/workflows/ci.yml) 为准。数据库与事务变更须实际运行 PostgreSQL 集成测试；因环境跳过时明确说明，不能记为已验证。
- 纯文档改动检查本地链接、章节引用与 `git diff --check`，不要求运行应用测试。完成前审阅 diff，确认没有意外生成物或无关改动，再记录实际验证结果。

## Agent 工作记录

- 开始任何会修改仓库的任务前，查看 [docs/agent-log/](docs/agent-log/) 中按文件名排序最新的 `YYYY-MM.md` 文件头部的最近记录，用于了解近期改动与上下文。日志只是辅助线索；实际工作前仍须核对相关权威文档和当前文件。
- 完成一个产生仓库文件变化的工作单元后，由主 Agent 按 [docs/agent-log/README.md](docs/agent-log/README.md) 的格式记录。纯问答、只读分析、未落地的方案和中间命令不记录。
- 多 Agent 协作时，子 Agent 只向主 Agent 报告结果，由主 Agent 在整体任务完成后统一写入一条日志，避免并发修改。
- Agent Log 不是产品、架构、API、schema、公式或决策的权威来源，不得用日志替代正文修改。与正式文档冲突时，以本文件指定的领域权威文档为准。

## 权威规则（单一真相源）

| 内容 | 唯一权威位置 | 其它地方只能 |
| --- | --- | --- |
| 产品目标 / MVP | [docs/product/prd.md](docs/product/prd.md) | 引用 |
| 阶段规划 | [docs/product/roadmap.md](docs/product/roadmap.md) | 引用 |
| 系统总体架构 / Monorepo 布局 | [docs/architecture/overview.md](docs/architecture/overview.md) | 引用 |
| 词典方案 / 数据源分工 | [docs/dictionary/overview.md](docs/dictionary/overview.md) | 引用 |
| 四表关系级模型 | [docs/architecture/data-model.md](docs/architecture/data-model.md) | 引用 |
| `dictionary_entries` / `user_words` 字段 | [docs/dictionary/data-model.md](docs/dictionary/data-model.md) | 引用 |
| 词形归一 | [docs/dictionary/normalization.md](docs/dictionary/normalization.md) | 引用 |
| Lookup / Enrich | [docs/dictionary/enrichment.md](docs/dictionary/enrichment.md) | 引用 |
| `review_sessions` / `review_items` 字段 | [docs/review/data-model.md](docs/review/data-model.md) | 引用 |
| 权重公式 / mastery / 抽样 | [docs/review/algorithm.md](docs/review/algorithm.md) | 引用 |
| Word API | [docs/api/words.md](docs/api/words.md) | 引用 |
| Review API | [docs/api/reviews.md](docs/api/reviews.md) | 引用 |
| Meta API（版本信息 / 词典导入进度） | [docs/api/meta.md](docs/api/meta.md) | 引用 |
| 发布流程（tag → 镜像构建 → compose） | [docs/deploy/release.md](docs/deploy/release.md) | 引用 |
| 页面与状态机 | [docs/frontend/review-flow.md](docs/frontend/review-flow.md) | 引用 |
| Go 工程结构 | [docs/backend/structure.md](docs/backend/structure.md) | 引用 |
| Go 技术选型 | [docs/specs/backend/Go 技术栈.md](docs/specs/backend/Go%20技术栈.md) | 引用 |
| Go 单体架构规范（分层 / Domain / Repo / Service / Handler / 事务 / 缓存） | [docs/specs/backend/Go 单体应用架构规范.md](docs/specs/backend/Go%20单体应用架构规范.md) | 引用 |
| Go 测试规范 | [docs/specs/backend/Go 测试规范.md](docs/specs/backend/Go%20测试规范.md) | 引用 |
| HTTP API 设计规范（响应结构 / 错误码 / 分页 / 路由版本） | [docs/specs/backend/HTTP API 设计规范.md](docs/specs/backend/HTTP%20API%20设计规范.md) | 引用 |
| 前端技术栈 | [docs/specs/frontend/前端技术栈.md](docs/specs/frontend/前端技术栈.md) | 引用 |
| 前端应用架构规范 | [docs/specs/frontend/前端应用架构规范.md](docs/specs/frontend/前端应用架构规范.md) | 引用 |
| 前端 API 与认证集成规范 | [docs/specs/frontend/前端 API 与认证集成规范.md](docs/specs/frontend/前端%20API%20与认证集成规范.md) | 引用 |
| 前端交互与可访问性规范 | [docs/specs/frontend/前端交互与可访问性规范.md](docs/specs/frontend/前端交互与可访问性规范.md) | 引用 |
| 前端测试规范 | [docs/specs/frontend/前端测试规范.md](docs/specs/frontend/前端测试规范.md) | 引用 |
| 设计系统方案 | [docs/specs/frontend/Vue 组件设计系统方案.md](docs/specs/frontend/Vue%20组件设计系统方案.md) | 引用 |
| Vue 性能与缓存优化 | [docs/specs/frontend/Vue 性能与缓存优化.md](docs/specs/frontend/Vue%20性能与缓存优化.md) | 引用 |
| 品牌 | [docs/brand/naming.md](docs/brand/naming.md) | 引用 |

不要在同一文档之外**复制一份再维护**任何 schema、公式或决策结论——跨文件不一致大多由此产生。

## 冲突裁决原则

跨文档冲突时按领域裁决，理想状态下不应存在冲突：

```text
产品规则冲突     → product/prd.md 为准
词典相关设计冲突 → dictionary/* 为准（overview 为入口）
复习算法冲突     → review/algorithm.md 为准
复习数据冲突     → review/data-model.md 为准
API 契约冲突     → api/* 为准
工程结构冲突     → backend/structure.md 为准
后端技术选型冲突 → specs/backend/Go 技术栈.md 为准
后端架构规范冲突 → specs/backend/Go 单体应用架构规范.md 为准
后端测试规范冲突 → specs/backend/Go 测试规范.md 为准
HTTP API 规范冲突 → specs/backend/HTTP API 设计规范.md 为准
前端技术选型冲突 → specs/frontend/前端技术栈.md 为准
前端架构规范冲突 → specs/frontend/前端应用架构规范.md 为准
前端认证集成冲突 → specs/frontend/前端 API 与认证集成规范.md 为准
前端交互与可访问性冲突 → specs/frontend/前端交互与可访问性规范.md 为准
前端测试规范冲突 → specs/frontend/前端测试规范.md 为准
设计系统冲突     → specs/frontend/Vue 组件设计系统方案.md 为准
Vue 性能优化冲突 → specs/frontend/Vue 性能与缓存优化.md 为准
品牌相关         → brand/naming.md 为准
```

## 整理约定

- 后写的文档可以修正前面已过时的设计，但不要让同一个决策长期存在两套说法——修改上游结论时，同步检查并更新引用了它的下游文档（交叉引用即显式依赖）。
- `product/prd.md` 只保留产品概念，不保留公式与 schema；公式与 schema 只出现在技术文档里。
- 词典术语已统一：`dictionary_entries.raw_meanings`（原始释义）、`dictionary_entries.review_meanings`（词典层默认复习释义）、`user_words.custom_review_meaning`（用户自定义，可空）。展示时 `custom ?? review_meanings ?? raw_meanings`。
- 用户自定义只写 `user_words`，任何文档都不应出现「用户修改 `dictionary_entries`」的说法。
- 权重公式的唯一权威在 `review/algorithm.md`；修改公式时同步更新其中的新词规则与最终示例，避免公式内部不一致。
- 表字段按领域归属：`dictionary_entries` / `user_words` 只在 `dictionary/data-model.md` 列出，`review_sessions` / `review_items` 只在 `review/data-model.md` 列出，两处不要重复维护同一份 schema。

## 冻结决策索引

完整决策表（含权威文档与一句话摘要）见 [docs/decisions/README.md](docs/decisions/README.md)，本文件只列索引，不重复规则细节：

```text
D001  ECDICT 全量本地化            → dictionary/overview.md（细节见 enrichment.md）
D002  Lookup / Enrich 分离          → dictionary/enrichment.md
D003  lemma 不确定不归并            → dictionary/normalization.md
D004  生词软删除（user_words.deleted_at）→ dictionary/data-model.md
D005  Review Submit 幂等 + 归属校验 + 事务原子性 → api/reviews.md
D006  词典数据 ≠ 用户学习数据        → dictionary/data-model.md
D007  释义三层取值 custom ?? review ?? raw → dictionary/data-model.md
D008  四表链路，概念 1:N、MVP 表现为 1:0..1 → architecture/data-model.md
D009  复习数量截断 min(count, available) → api/reviews.md
D010  Session 恢复由用户选择；同一时间仅一个 active → review/data-model.md
D011  权重 / mastery 不落库动态计算   → review/algorithm.md
D012  MVP 只做随机复习（到期复习后置） → product/roadmap.md
```

裁决口径：被索引的决策已冻结（2026-09-03 确认，勿回退为旧设计）；改动某条决策时，先修改其权威文档正文，再同步本索引与 `docs/decisions/README.md` 的描述。
