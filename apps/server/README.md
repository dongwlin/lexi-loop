# lexi-loop server

LexiLoop（词环）英语生词复习系统后端。工程结构规范见 [docs/backend/structure.md](../../docs/backend/structure.md)，技术选型见 [docs/specs/backend/Go 技术栈.md](../../docs/specs/backend/Go%20技术栈.md)。

## 当前状态：MVP 骨架

代码框架已按分层规范搭好，**业务逻辑尚未实现**（各包为注释占位，见各文件 TODO）。当前交付物是「可启动的 Gin 服务器 + `/healthz` 健康检查」，命令形态（`serve` / `migrate` / `import-ecdict`）已就位，`migrate` 与 `import-ecdict` 报「尚未实现」。

```text
apps/server/
├─ main.go                       可执行入口：仅调用 cmd.Execute()
├─ cmd/                          CLI 路由层（serve / migrate / import-ecdict）
├─ migrations/                   版本化迁移占位（up/down SQL）
└─ internal/
   ├─ app/                       组合根：Server 生命周期、优雅关闭
   ├─ domain/                    领域模型与纯业务规则（占位）
   ├─ repo/                      数据访问适配器 + internal/schema（占位）
   ├─ service/                   业务用例编排（占位）
   ├─ importer/ecdict/           ECDICT 离线导入（占位）
   ├─ handler/                   router.go（/api/v1 挂载点）+ httpresp / v1 / middleware
   ├─ apperr/                    类型化应用错误（占位）
   └─ infra/                     config（viper，已实现）/ database（占位）
```

目录与职责对齐 [docs/backend/structure.md](../../docs/backend/structure.md) §2；与通用分层规范的关系见 [docs/specs/backend/Go 单体应用架构规范.md](../../docs/specs/backend/Go%20单体应用架构规范.md)。

## 运行

```bash
cd apps/server

# 直接运行（监听 :8080）
go run . serve

# 指定监听地址（viper 读取 LEXI_HTTP_ADDR；也支持 --http-addr）
LEXI_HTTP_ADDR=:9090 go run . serve
```

验证：

```bash
curl http://localhost:8080/healthz   # -> {"status":"ok"}
```

`/healthz` 是基础设施端点，挂在根路径、不参与业务版本（[HTTP API 设计规范 §2.4](../../docs/specs/backend/HTTP%20API%20设计规范.md)）。业务路由统一挂在 `/api/v1` 下，路径契约见 [docs/api/words.md](../../docs/api/words.md) 与 [docs/api/reviews.md](../../docs/api/reviews.md)。

## 下一步（按依赖顺序）

1. `internal/infra/database`：bun + pgx 连接池与迁移执行适配器（选型见 [Go 技术栈](../../docs/specs/backend/Go%20技术栈.md)）
2. `internal/domain`：按 [dictionary/data-model.md](../../docs/dictionary/data-model.md) 与 [review/data-model.md](../../docs/review/data-model.md) 落模型
3. `internal/repo` + `migrations/`：数据访问与真实迁移
4. `internal/apperr` + `httpresp`：类型化错误与统一响应（[HTTP API 设计规范](../../docs/specs/backend/HTTP%20API%20设计规范.md)）
5. `internal/service` / `internal/handler/v1`：业务用例与版本化 Handler
6. 中间件替换为 `handler/middleware` 自定义实现，`app/provider.go` 完成组合根组装

## 模块

module 路径：`github.com/dongwlin/lexi-loop/apps/server`（GitHub 远程仓库 `dongwlin/lexi-loop`）。
