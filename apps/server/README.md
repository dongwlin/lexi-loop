# lexi-loop server

LexiLoop（词环）英语生词复习系统后端。工程结构规范见 [docs/backend/structure.md](../../docs/backend/structure.md)，技术选型见 [docs/specs/backend/Go 技术栈.md](../../docs/specs/backend/Go%20技术栈.md)。

## 实现入口

阶段状态见 [Roadmap](../../docs/product/roadmap.md)。服务端提供 Word / Review / Meta API、本地词典导入、迁移和离线 OpenAPI 生成；工程目录与职责见 [backend/structure.md](../../docs/backend/structure.md)，接口行为分别见 [Word API](../../docs/api/words.md)、[Review API](../../docs/api/reviews.md) 与 [Meta API](../../docs/api/meta.md)。

OpenAPI 产物由下方命令生成，不手改，也不挂载在线文档路由。启动 `serve` 需要 `LEXI_DATABASE_URL`。

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

OpenAPI 3.1 spec 由 `openapi` 命令离线生成（不需要数据库；产物落盘仓库根 `docs/openapi/`，由生成器维护、请勿手改）：

```bash
cd apps/server
go run . openapi                      # 生成 docs/openapi/openapi.json 与 openapi.yaml
go run . openapi --out /tmp/openapi   # 自定义输出目录（相对当前工作目录）
```

版本信息由 `version` 命令输出（不需要数据库；`Version` / `BuildTime` 由 `go build -ldflags "-X"` 注入，未注入即开发构建、以 `dev` 占位，契约见 [docs/api/meta.md](../../docs/api/meta.md)）：

```bash
go run . version                      # 只输出版本号（如 dev）
go run . version -b                   # version / build_time / go_version 三行（buildTime 空显示 -）
```

验证：

```bash
curl http://localhost:8080/healthz   # -> {"status":"ok"}

# 业务端点示例（先 migrate up；完整契约见 docs/api/words.md、docs/api/reviews.md、docs/api/meta.md）
curl -X POST http://localhost:8080/api/v1/words/import \
  -H 'Content-Type: application/json' \
  -d '{"words":[{"word":"ambiguous","count":1}]}'
curl -X POST http://localhost:8080/api/v1/reviews -H 'Content-Type: application/json' -d '{"count":30}'
```

`/healthz` 是基础设施端点，挂在根路径、不参与业务版本（[HTTP API 设计规范 §2.4](../../docs/specs/backend/HTTP%20API%20设计规范.md)）。业务路由统一挂在 `/api/v1` 下，路径契约见 [docs/api/words.md](../../docs/api/words.md)、[docs/api/reviews.md](../../docs/api/reviews.md) 与 [docs/api/meta.md](../../docs/api/meta.md)。

## 测试

```bash
go test ./...                       # 单元 + 集成测试（集成需本机 Docker）
go test -short ./...                # 只跑单元测试（跳过 testcontainers 集成测试）
go test -race ./internal/infra/... ./internal/repo/... ./internal/service/... ./internal/handler/... ./internal/importer/... ./migrations
```

`internal/infra/database`、`migrations`、`internal/repo`、`internal/service`、`internal/handler` 与 `internal/importer/ecdict` 的集成测试各自在包级 `TestMain` 中启动一次共享的真实 PostgreSQL 容器（镜像 `postgres:18-alpine`，全部用例复用；各包 `TestMain` 会先应用全部迁移）；Docker 不可用或传 `-short` 时集成用例跳过、纯单元测试仍运行（见 [Go 测试规范](../../docs/specs/backend/Go 测试规范.md)）。structure.md §5.4 的并发 / 回滚必测场景（单 active session、并发提交幂等、最后两题并发提交必完成、注入失败整体回滚、items 中途失败不留半成品）已在 Service 层用真实 PostgreSQL 验证；`internal/handler` 组件测试经 httptest 驱动完整 HTTP 栈验证端点契约，OpenAPI spec 单元测试（离线构造，不依赖 Docker）验证操作路由与 Error 模型 schema；`internal/importer/ecdict` 以小型固定 CSV 验证字段映射、幂等重跑与批次回滚续传。

## 模块

module 路径：`github.com/dongwlin/lexi-loop/apps/server`（GitHub 远程仓库 `dongwlin/lexi-loop`）。
