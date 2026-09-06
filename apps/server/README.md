# lexi-loop server

LexiLoop（词环）英语生词复习系统后端。工程结构规范见 [docs/backend/structure.md](../../docs/backend/structure.md)，技术选型见 [docs/specs/backend/Go 技术栈.md](../../docs/specs/backend/Go%20技术栈.md)。

## 当前状态：MVP 业务链路可用

后端已按分层规范实现 **Word / Review 全部业务用例与 `/api/v1` 端点**：`internal/domain` 领域模型与纯业务规则（四实体工厂、ApplyReview / Submit / Complete / Abandon 状态迁移、weight / mastery 动态计算，含单元测试）、`internal/infra/database` 数据库连接池（bun + pgx）、`migrations/` 真实迁移 DDL（四表，含「全库至多一个 active session」部分唯一索引）、`internal/repo` 数据访问适配器（advisory lock / FOR UPDATE / 条件 UPDATE、SQLSTATE 错误归一化）、`internal/apperr` 类型化应用错误与 `internal/handler/httpresp` 统一响应（唯一的 `apperr.Kind → HTTP 状态` 映射）均已实现并含测试；`internal/service`（`WeightedSampler` 加权随机不放回抽样、`DictionaryService` 本地 Lookup——归一 → 词形变化表归并 → 未命中创建最小词条、`WordService` 五用例、`ReviewService` 三用例按 structure.md §5 的事务与并发边界，`tx.go` 提供可重放事务的有上限重试；含 testcontainers 集成测试，覆盖 §5.4 必测的并发与回滚场景）、`internal/handler/v1`（版本化 Handler + DTO，httptest 组件测试驱动完整 HTTP 栈）、`internal/importer/ecdict`（ECDICT CSV 流式解析与字段映射、按批分事务的幂等导入，含单元与集成测试）与 `import-ecdict` 命令，以及 `internal/app` 组合根（显式组装 DB / Service / Handler）与 `internal/infra/logger` 日志初始化（zerolog ConsoleWriter，CLI 启动时接管全局日志）均已落地；`serve` 需要 `LEXI_DATABASE_URL`。`handler/middleware` 自定义中间件已实现并经 `handler/router.go` 全局挂载（顺序 request_id → recovery → cors → logger）：request_id 透传 / 生成 `X-Request-Id` 响应头与链路日志字段、logger 输出 zerolog 请求日志（method / path / status / bytes / client_ip / latency，按状态分级）、recovery 将 panic 统一渲染为内部错误（不向客户端暴露 panic 值与堆栈）、cors 封装官方 gin-contrib/cors 按显式 Origin 白名单放行（规范 §11.2 的方法 / 请求头 / 暴露响应头）；四个中间件均含单元测试。

```text
apps/server/
├─ main.go                       可执行入口：仅调用 cmd.Execute()
├─ cmd/                          CLI 路由层（serve / migrate / import-ecdict）
├─ migrations/                   版本化迁移（migrations.go 执行入口 + 四表真实 DDL）
└─ internal/
   ├─ app/                       组合根：组件显式组装、Server 生命周期、优雅关闭（已接入 DB / Service / Handler）
   ├─ domain/                    领域模型与纯业务规则（已实现，含单元测试）
   ├─ repo/                      数据访问适配器 + internal/schema（已实现，含集成测试）
   ├─ service/                   业务用例编排：word / dictionary / review / sampler / tx（已实现，含集成测试）
   ├─ importer/ecdict/           ECDICT 离线导入：parser 字段映射 + importer 批事务编排（已实现，含单元与集成测试）
   ├─ handler/                   router.go（/api/v1 已挂载业务路由）+ httpresp / v1（已实现，含组件测试）/ middleware（已实现，含单元测试）
   ├─ apperr/                    类型化应用错误（已实现，含单元测试）
   └─ infra/                     config（viper，已实现）/ database（连接池已实现）/ logger（zerolog ConsoleWriter 日志初始化，已实现）
```

目录与职责对齐 [docs/backend/structure.md](../../docs/backend/structure.md) §2；与通用分层规范的关系见 [docs/specs/backend/Go 单体应用架构规范.md](../../docs/specs/backend/Go%20单体应用架构规范.md)。

## 运行

```bash
cd apps/server

# 直接运行（监听 :8080；serve 必须提供数据库连接串）
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . serve

# 指定监听地址（viper 读取 LEXI_HTTP_ADDR；也支持 --http-addr）
LEXI_HTTP_ADDR=:9090 LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . serve
```

数据库连接池与 CORS 白名单同样通过 `LEXI_*` 环境变量配置（连接池 0 值表示采用 pgx 默认，见 `internal/infra/config`）：

| 环境变量 | 说明 |
| --- | --- |
| `LEXI_DATABASE_URL` | PostgreSQL 连接串，必填 |
| `LEXI_DATABASE_MAX_CONNS` / `LEXI_DATABASE_MIN_CONNS` | 连接池上下限 |
| `LEXI_DATABASE_MAX_CONN_LIFETIME` / `LEXI_DATABASE_MAX_CONN_IDLE_TIME` | 连接复用与空闲回收时长（Go duration，如 `5m` / `90s`） |
| `LEXI_HTTP_CORS_ALLOWED_ORIGINS` | CORS 允许的前端 Origin 白名单，逗号分隔（不支持通配符）；缺省为本地开发源 `http://localhost:5173`、`http://127.0.0.1:5173`，生产部署须显式配置 |

数据库迁移由 `migrate` 命令显式执行（应用启动不隐式迁移，见 [docs/backend/structure.md](../../docs/backend/structure.md) §2）：

```bash
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . migrate up        # 应用全部未执行的迁移
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . migrate down       # 回退 1 步
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . migrate down 2     # 回退指定步数（超过可用数时截断）
```

ECDICT 离线导入由 `import-ecdict` 命令执行（导入前先 `migrate up`，字段映射与源列取舍见 `internal/importer/ecdict`）：

```bash
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . import-ecdict ecdict.csv           # 全量导入（默认每 1000 行一个事务）
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . import-ecdict ecdict.csv --batch-size 2000   # 自定义单事务行数
```

导入按固定批次分事务提交，失败只回滚当前批次并中止；写入按 headword 唯一键幂等，中断（SIGINT / SIGTERM）或失败后重跑同一文件即可续传。

验证：

```bash
curl http://localhost:8080/healthz   # -> {"status":"ok"}

# 业务端点示例（先 migrate up；完整契约见 docs/api/words.md、docs/api/reviews.md）
curl -X POST http://localhost:8080/api/v1/words/import \
  -H 'Content-Type: application/json' \
  -d '{"words":[{"word":"ambiguous","count":1}]}'
curl -X POST http://localhost:8080/api/v1/reviews -H 'Content-Type: application/json' -d '{"count":30}'
```

`/healthz` 是基础设施端点，挂在根路径、不参与业务版本（[HTTP API 设计规范 §2.4](../../docs/specs/backend/HTTP%20API%20设计规范.md)）。业务路由统一挂在 `/api/v1` 下，路径契约见 [docs/api/words.md](../../docs/api/words.md) 与 [docs/api/reviews.md](../../docs/api/reviews.md)。

## 测试

```bash
go test ./...                       # 单元 + 集成测试（集成需本机 Docker）
go test -short ./...                # 只跑单元测试（跳过 testcontainers 集成测试）
go test -race ./internal/infra/... ./internal/repo/... ./internal/service/... ./internal/handler/... ./internal/importer/... ./migrations
```

`internal/infra/database`、`migrations`、`internal/repo`、`internal/service`、`internal/handler` 与 `internal/importer/ecdict` 的集成测试各自在包级 `TestMain` 中启动一次共享的真实 PostgreSQL 容器（镜像 `postgres:18-alpine`，全部用例复用；各包 `TestMain` 会先应用全部迁移）；Docker 不可用或传 `-short` 时集成用例跳过、纯单元测试仍运行（见 [Go 测试规范](../../docs/specs/backend/Go 测试规范.md)）。structure.md §5.4 的并发 / 回滚必测场景（单 active session、并发提交幂等、最后两题并发提交必完成、注入失败整体回滚、items 中途失败不留半成品）已在 Service 层用真实 PostgreSQL 验证；`internal/handler` 组件测试经 httptest 驱动完整 HTTP 栈验证端点契约；`internal/importer/ecdict` 以小型固定 CSV 验证字段映射、幂等重跑与批次回滚续传。

## 下一步（按依赖顺序）

1. `app.Run` 的 stdlib log 输出切换为注入的 zerolog logger

## 模块

module 路径：`github.com/dongwlin/lexi-loop/apps/server`（GitHub 远程仓库 `dongwlin/lexi-loop`）。
