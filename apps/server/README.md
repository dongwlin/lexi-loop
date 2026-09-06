# lexi-loop server

LexiLoop（词环）英语生词复习系统后端。工程结构规范见 [docs/backend/structure.md](../../docs/backend/structure.md)，技术选型见 [docs/specs/backend/Go 技术栈.md](../../docs/specs/backend/Go%20技术栈.md)。

## 当前状态：MVP 骨架

代码框架已按分层规范搭好，**业务用例尚未实现**（service / handler/v1 / middleware / importer 仍为注释占位，见各文件 TODO）。当前交付物是「可启动的 Gin 服务器 + `/healthz` 健康检查」、`internal/domain` 领域模型与纯业务规则（按 dictionary / review 数据模型落四实体：工厂生成 UUID v7 与 UTC 时间戳、ApplyReview / Submit / Complete / Abandon 状态迁移、weight / mastery 动态计算，含单元测试）、`internal/infra/database` 数据库连接池（bun + pgx，含 testcontainers 集成测试）、`migrations/` 真实迁移 DDL（dictionary_entries / user_words / review_sessions / review_items 四表，含外键、唯一约束、CHECK、索引与「全库至多一个 active session」部分唯一索引）、`internal/repo` 数据访问适配器（DictionaryRepo / UserWordRepo / ReviewRepo：CRUD、软删除、遇词累计 upsert、advisory lock / FOR UPDATE / 条件 UPDATE、SQLSTATE 错误归一化，均含 testcontainers 集成测试）、`internal/apperr` 类型化应用错误（Kind / 稳定业务 code / `New` / `Internal`，含单元测试）与 `internal/handler/httpresp` 统一响应（全站唯一的 `code` / `message` / `data` 结构与 `apperr.Kind → HTTP 状态` 映射，含 httptest 单元测试）；`import-ecdict` 命令已注册但报「尚未实现」。

```text
apps/server/
├─ main.go                       可执行入口：仅调用 cmd.Execute()
├─ cmd/                          CLI 路由层（serve / migrate / import-ecdict）
├─ migrations/                   版本化迁移（migrations.go 执行入口 + 四表真实 DDL）
└─ internal/
   ├─ app/                       组合根：Server 生命周期、优雅关闭
   ├─ domain/                    领域模型与纯业务规则（已实现，含单元测试）
   ├─ repo/                      数据访问适配器 + internal/schema（已实现，含集成测试）
   ├─ service/                   业务用例编排（占位）
   ├─ importer/ecdict/           ECDICT 离线导入（占位）
   ├─ handler/                   router.go（/api/v1 挂载点）+ httpresp（已实现，含单元测试）/ v1 / middleware
   ├─ apperr/                    类型化应用错误（已实现，含单元测试）
   └─ infra/                     config（viper，已实现）/ database（连接池已实现）
```

目录与职责对齐 [docs/backend/structure.md](../../docs/backend/structure.md) §2；与通用分层规范的关系见 [docs/specs/backend/Go 单体应用架构规范.md](../../docs/specs/backend/Go%20单体应用架构规范.md)。

## 运行

```bash
cd apps/server

# 直接运行（监听 :8080）
go run . serve

# 指定监听地址（viper 读取 LEXI_HTTP_ADDR；也支持 --http-addr）
LEXI_HTTP_ADDR=:9090 go run . serve

# 指定 PostgreSQL 连接串（viper 读取 LEXI_DATABASE_URL）
LEXI_DATABASE_URL='postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable' go run . serve
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
```

`/healthz` 是基础设施端点，挂在根路径、不参与业务版本（[HTTP API 设计规范 §2.4](../../docs/specs/backend/HTTP%20API%20设计规范.md)）。业务路由统一挂在 `/api/v1` 下，路径契约见 [docs/api/words.md](../../docs/api/words.md) 与 [docs/api/reviews.md](../../docs/api/reviews.md)。

## 测试

```bash
go test ./...                       # 单元 + 集成测试（集成需本机 Docker）
go test -short ./...                # 只跑单元测试（跳过 testcontainers 集成测试）
go test -race ./internal/infra/... ./internal/repo/... ./migrations  # infra + repo + 迁移全量（含集成测试）
```

`internal/infra/database`、`migrations` 与 `internal/repo` 的集成测试各自在包级 `TestMain` 中启动一次共享的真实 PostgreSQL 容器（镜像 `postgres:18-alpine`，全部用例复用；repo 的 `TestMain` 会先应用全部迁移）；Docker 不可用或传 `-short` 时集成用例跳过、纯单元测试仍运行（见 [Go 测试规范](../../docs/specs/backend/Go%20测试规范.md)）。structure.md §5.4 的并发 / 回滚用例（单 active session、并发提交幂等、整体回滚）将在 Service 层按同一方式验证。

## 下一步（按依赖顺序）

1. `internal/service` / `internal/handler/v1`：业务用例与版本化 Handler（structure.md §5 的事务与并发边界）
2. `internal/importer/ecdict` + `import-ecdict` 命令：ECDICT 离线导入
3. 中间件替换为 `handler/middleware` 自定义实现，`app/provider.go` 完成组合根组装

## 模块

module 路径：`github.com/dongwlin/lexi-loop/apps/server`（GitHub 远程仓库 `dongwlin/lexi-loop`）。
