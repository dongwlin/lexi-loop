# lexi-loop server

LexiLoop（词环）英语生词复习系统后端。工程结构规范见 [docs/backend/structure.md](../../docs/backend/structure.md)，技术选型见 [docs/specs/backend/Go 技术栈.md](../../docs/specs/backend/Go%20技术栈.md)。

## 当前状态：MVP 业务链路可用

后端已按分层规范实现 **Word / Review 全部业务用例与 `/api/v1` 端点**：`internal/domain` 领域模型与纯业务规则（四实体工厂、ApplyReview / Submit / Complete / Abandon 状态迁移、weight / mastery 动态计算，含单元测试）、`internal/infra/database` 数据库连接池（bun + pgx）、`migrations/` 真实迁移 DDL（四表，含「全库至多一个 active session」部分唯一索引）、`internal/repo` 数据访问适配器（advisory lock / FOR UPDATE / 条件 UPDATE、SQLSTATE 错误归一化）、`internal/apperr` 类型化应用错误与 `internal/handler/httpresp` 统一响应（唯一的 `apperr.Kind → HTTP 状态` 映射）均已实现并含测试；`internal/service`（`WeightedSampler` 加权随机不放回抽样、`DictionaryService` 本地 Lookup——归一 → 词形变化表归并 → 未命中创建最小词条、`WordService` 五用例、`ReviewService` 三用例按 structure.md §5 的事务与并发边界，`tx.go` 提供可重放事务的有上限重试；含 testcontainers 集成测试，覆盖 §5.4 必测的并发与回滚场景）、`internal/handler/v1`（版本化 Handler + DTO，httptest 组件测试驱动完整 HTTP 栈）与 `internal/app` 组合根（显式组装 DB / Service / Handler）均已落地；`serve` 现在需要 `LEXI_DATABASE_URL`。`import-ecdict` 命令与 `handler/middleware` 自定义实现仍为占位。

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
   ├─ importer/ecdict/           ECDICT 离线导入（占位）
   ├─ handler/                   router.go（/api/v1 已挂载业务路由）+ httpresp / v1（已实现，含组件测试）/ middleware（占位）
   ├─ apperr/                    类型化应用错误（已实现，含单元测试）
   └─ infra/                     config（viper，已实现）/ database（连接池已实现）
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

数据库连接池参数同样通过 `LEXI_*` 环境变量配置（0 值表示采用 pgx 默认，见 `internal/infra/config`）：

| 环境变量 | 说明 |
| --- | --- |
| `LEXI_DATABASE_URL` | PostgreSQL 连接串，必填 |
| `LEXI_DATABASE_MAX_CONNS` / `LEXI_DATABASE_MIN_CONNS` | 连接池上下限 |
| `LEXI_DATABASE_MAX_CONN_LIFETIME` / `LEXI_DATABASE_MAX_CONN_IDLE_TIME` | 连接复用与空闲回收时长（Go duration，如 `5m` / `90s`） |

数据库迁移由 `migrate` 命令显式执行（应用启动不隐式迁移，见 [docs/backend/structure.md](../../docs/backend/structure.md) §2）：

```bash
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . migrate up        # 应用全部未执行的迁移
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . migrate down       # 回退 1 步
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . migrate down 2     # 回退指定步数（超过可用数时截断）
```

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
go test -race ./internal/infra/... ./internal/repo/... ./internal/service/... ./internal/handler/... ./migrations
```

`internal/infra/database`、`migrations`、`internal/repo`、`internal/service` 与 `internal/handler` 的集成测试各自在包级 `TestMain` 中启动一次共享的真实 PostgreSQL 容器（镜像 `postgres:18-alpine`，全部用例复用；各包 `TestMain` 会先应用全部迁移）；Docker 不可用或传 `-short` 时集成用例跳过、纯单元测试仍运行（见 [Go 测试规范](../../docs/specs/backend/Go 测试规范.md)）。structure.md §5.4 的并发 / 回滚必测场景（单 active session、并发提交幂等、最后两题并发提交必完成、注入失败整体回滚、items 中途失败不留半成品）已在 Service 层用真实 PostgreSQL 验证；`internal/handler` 组件测试经 httptest 驱动完整 HTTP 栈验证端点契约。

## 下一步（按依赖顺序）

1. `internal/importer/ecdict` + `import-ecdict` 命令：ECDICT 离线导入
2. 中间件替换为 `handler/middleware` 自定义实现（request_id / logger / recovery / cors）

## 模块

module 路径：`github.com/dongwlin/lexi-loop/apps/server`（GitHub 远程仓库 `dongwlin/lexi-loop`）。
