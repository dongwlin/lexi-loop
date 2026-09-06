# AGENTS.md — lexi-loop server

`apps/server` 是 LexiLoop 的 Go 后端应用。本文件是 server 子树内 Agent 的补充工作指引；仓库级纪律（改动前读 agent-log、冲突裁决、改动后记录）以根目录 [AGENTS.md](../../AGENTS.md) 为准，代码目录与各层职责权威见 [docs/backend/structure.md](../../docs/backend/structure.md)。本文件不维护 schema、公式或决策结论，只引用。

## 当前状态：MVP 业务链路可用

后端已实现 Word / Review 全部业务用例与 `/api/v1` 端点：`internal/domain`（四表领域模型、`NewXxx` 工厂、ApplyReview / Submit / Complete / Abandon 状态迁移、weight / mastery 纯函数，含单元测试）、`internal/infra/config`（viper + `LEXI_*` 环境变量）、`internal/infra/database`（bun + pgx/v5 连接池构造）、`internal/infra/logger`（zerolog ConsoleWriter 日志初始化，root 命令 `PersistentPreRun` 接管全局日志，`migrations` 等使用全局 log 的包随之输出控制台格式）、`migrations` 包（embed + golang-migrate 迁移执行入口，`migrate` 命令已接入，迁移 SQL 为四表真实 DDL）与 `internal/repo`（DictionaryRepo / UserWordRepo / ReviewRepo 数据访问适配器：schema ↔ domain 转换、SQLSTATE 错误归一化、软删除、遇词累计 upsert、advisory lock / FOR UPDATE / 条件 UPDATE；`internal/repo`、`infra/database` 与 `migrations` 均含 testcontainers 集成测试）、`internal/apperr`（Kind / 稳定业务 code / `New` / `Internal` 类型化应用错误）与 `internal/handler/httpresp`（统一 `code` / `message` / `data` 响应结构与唯一的 `apperr.Kind → HTTP 状态` 映射，两者均含单元测试）、`internal/service`（`WordService` / `DictionaryService` / `ReviewService` / `WeightedSampler` / `tx.go` 可重放事务重试，按 structure.md §5 的事务与并发边界实现，含覆盖 §5.4 必测场景的 testcontainers 集成测试）、`internal/handler/v1`（版本化 Handler + DTO，含 httptest 组件测试）与 `internal/handler/router.go`（`/api/v1` 业务路由已挂载，全局中间件为 `internal/handler/middleware` 自定义实现：request_id / logger / recovery + 封装官方 gin-contrib/cors 的 CORS，经 `RegisterRoutes` 的 `Options` 按 request_id → recovery → cors → logger 挂载）、`internal/app`（组合根已接入 DB / Service / Handler；`serve` 需要 `LEXI_DATABASE_URL`，根命令将 `logger.Init` 构造的 zerolog logger 注入 `app.Run` 并传给 `RegisterRoutes`）、`internal/importer/ecdict`（ECDICT CSV 流式解析与字段映射、按固定批次分事务的幂等导入，含单元与集成测试）与 `import-ecdict` 命令均已实现。分层现状与下一步清单以 [README.md](README.md) 为准——完成里程碑后同步更新两处，避免状态失真。

改动本子树任何代码前：先读 [docs/agent-log/](../../docs/agent-log/) 当月文件的最近记录了解上下文，再核对下表对应文档与当前代码。

## 权威文档速查（改动对象 → 先读）

| 改动对象 | 先读 |
| --- | --- |
| 目录结构 / 职责边界 / 并发用例 | [docs/backend/structure.md](../../docs/backend/structure.md)（§1–§5） |
| `dictionary_entries` / `user_words` 字段与释义取值 | [docs/dictionary/data-model.md](../../docs/dictionary/data-model.md) |
| `review_sessions` / `review_items` 字段与状态 | [docs/review/data-model.md](../../docs/review/data-model.md) |
| 权重 / mastery / 抽样 | [docs/review/algorithm.md](../../docs/review/algorithm.md)（公式唯一权威，实现与测试不得另写一套） |
| Word / Review HTTP 契约 | [docs/api/words.md](../../docs/api/words.md)、[docs/api/reviews.md](../../docs/api/reviews.md) |
| 响应结构 / 错误码 / 分页 / 路由版本 | [docs/specs/backend/HTTP API 设计规范.md](../../docs/specs/backend/HTTP%20API%20设计规范.md) |
| 分层 / 事务 / 缓存通用规范 | [docs/specs/backend/Go 单体应用架构规范.md](../../docs/specs/backend/Go%20单体应用架构规范.md) |
| 测试方法与并发测试要求 | [docs/specs/backend/Go 测试规范.md](../../docs/specs/backend/Go%20测试规范.md)、structure.md §5.4 |

## 常用命令

```bash
cd apps/server
go build ./...                 # 构建全部包
go vet ./...                   # 静态检查
go test ./...                  # 单元 + 集成测试（集成需本机 Docker）
go test -short ./...           # 只跑单元测试（跳过集成测试）
go run . serve                 # 启动 HTTP 服务（默认监听 :8080；需要 LEXI_DATABASE_URL）
LEXI_HTTP_ADDR=:9090 LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . serve
curl http://localhost:8080/healthz   # -> {"status":"ok"}
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . migrate up      # 应用全部未执行的迁移
LEXI_DATABASE_URL='...' go run . migrate down [steps]   # 回退指定步数（默认 1，超过可用数时截断）
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . import-ecdict ecdict.csv   # 导入 ECDICT CSV（先 migrate up；--batch-size 调整单事务行数）
```

`infra/config`、`infra/database`、`infra/logger`、`internal/repo`、`migrations`、`internal/apperr`、`internal/handler/httpresp`、`internal/service`、`internal/handler`、`internal/handler/middleware` 与 `internal/importer/ecdict` 已有测试（apperr / httpresp / sampler / ecdict 解析映射 / logger / middleware 为纯单元测试，其余含集成）：`infra/database`、`internal/repo`、`migrations`、`internal/service`、`internal/handler` 与 `internal/importer/ecdict` 的集成测试各自在包级 `TestMain` 中启动一次共享的真实 PostgreSQL 容器（镜像 `postgres:18-alpine`，全部用例复用；各包 `TestMain` 会先应用全部迁移）；无 Docker 或传 `-short` 时集成用例跳过、单元测试仍运行。structure.md §5.4 列出的并发 / 回滚必测场景（单 active session、同 item 并发提交只累计一次、最后两 item 并发提交后 session 必为 completed、提交中注入失败整体回滚、items 中途失败不留半成品 session）已在 Service 层验证，失败注入使用测试库临时触发器与真实 SQL 篡改，不触碰 migrations；`internal/importer/ecdict` 以小型固定 CSV 验证字段映射、幂等重跑与批次回滚续传。

## 分层实现要点

以 structure.md §4 各层完整描述为准，以下只记常见踩坑的速记指针：

- 依赖单向：业务包不得反向 import `cmd`；Handler / Service / Repo / Domain / Importer / Infra 不得反向 import `app`（§3）。
- Domain 结构体不带 bun tag / `bun.BaseModel`，方法只操作自身字段、不查库；实体创建用 `domain.NewXxx` 工厂生成 UUID v7 与 UTC 时间戳（§2、§4.1）。
- 状态迁移与累计更新走 Domain 方法（如 `ApplyReview` / `Submit` / `Complete`），Service 不直改有不变量的字段（§4.3）。
- Repo 无状态，由 Service / Importer 方法内以当前 `bun.IDB`（含 `bun.Tx`）按需构造，不做长生命周期注入；bun schema 只允许在 `repo/internal/schema`，`schema ↔ domain` 转换只在 Repo 内（§4.2）。
- Service 编排用例与事务，不直接构造 SQL；同一事务内的全部 Repo 都使用传入的 `bun.Tx`（§4.3、§10）。
- JSON 契约只由 `handler/v1/dto` 定义；Service 请求 / 结果类型不带 json tag。API 版本只作用于 Handler / DTO，不复制 Service（§4.3–§4.4）。
- 成功 / 失败响应与 `apperr.Kind → HTTP 状态` 映射只存在于 `handler/httpresp`；Handler 与 Middleware 不得自行复制（§4.4）。
- 错误用 `errors.Is` / `errors.As` + `apperr` 识别，禁止按 `err.Error()` 文本分支（§4.5）。
- 状态值使用 Domain 类型化常量，不在 Service / Repo / Handler 中散落字符串（§4.1）。
- `migrations/` 是数据库结构的唯一可执行落点，字段语义以数据模型文档为准；启动不隐式迁移，由 `lexi-loop migrate` 显式执行，执行入口为 `migrations` 包的 `Migrate`（embed + golang-migrate，down 步数按当前版本截断为可用数）（§2、§3）。
- 写用例（StartSession / SubmitResult / ImportWords 等）的锁顺序、幂等、有上限重试与回滚边界按 structure.md §5 实现。

## MVP 边界与冻结决策

冻结决策索引见根 [AGENTS.md](../../AGENTS.md) 与 [docs/decisions/README.md](../../docs/decisions/README.md)。后端落地时的硬约束（细节以对应权威文档为准，这里不展开）：

- D004 软删除：删除写 `user_words.deleted_at`；不存在「用户修改 `dictionary_entries`」的说法。
- D005：Submit 幂等 + 归属校验 + 事务原子性；D009：复习数量截断 `min(count, available)`。
- D010：全库至多一个 active session，advisory lock + 部分唯一索引双保险。
- 释义展示取值 `custom ?? review ?? raw` 三层语义由 Domain 纯函数实现，语义以 data-model 为准。
- 不属于 MVP（V2 / V3 阶段）：认证 / PASETO、在线 Provider / Enrich、缓存、队列、多用户；当前不为此创建占位文件（structure.md §9）。

## 变更时的文档同步

- 先改权威文档正文，再同步引用它的下游（本文件、README、decisions 索引、agent-log），避免同一决策长期存在两套说法（根 AGENTS.md 整理约定）。
- 完成产生仓库文件变化的工作单元后，由主 Agent 按 [docs/agent-log/README.md](../../docs/agent-log/README.md) 在当月文件记录一条，写明完成内容、涉及文件与验证方式。
