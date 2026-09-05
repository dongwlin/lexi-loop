# Go 后端技术栈

本文件是 Go 后端项目的技术选型权威清单。新项目默认采用下列组件；替换任一默认选型前，应先结合实际约束讨论。架构、接口、数据库与测试的具体用法由对应规范定义，本文件不重复展开。

## 选型总览

| 领域     | 默认选型                     | 定位                                            |
| ------ | ------------------------ | --------------------------------------------- |
| Web 框架 | gin                      | HTTP JSON API、参数绑定、校验与中间件链                    |
| 数据库    | PostgreSQL               | 关系型数据的默认持久化方案                                 |
| 数据访问   | bun + pgx/v5/stdlib      | SQL-first ORM，通过 `database/sql` 接入 PostgreSQL |
| 数据库迁移 | golang-migrate/v4        | 版本化 SQL 迁移，SQL 经 embed 内嵌于 `apps/server/migrations/` |
| 日志     | zerolog                  | 低分配的结构化 JSON 日志                               |
| 依赖组装   | 手写组合根                    | 显式构造并组装长生命周期组件                                |
| 认证     | go-paseto                | 本地验证的 access token + refresh token            |
| 配置     | viper                    | 统一读取配置文件、环境变量与 flag                           |
| 测试     | testify + testcontainers | 断言与真实 PostgreSQL 集成测试                         |
| API 文档 | huma                     | 从代码生成 OpenAPI 3.1 文档，同时提供请求校验与错误处理     |
| CLI    | cobra                    | 组织服务启动、数据库迁移和离线导入等子命令                       |
| 进程内缓存  | ristretto（按需）            | 经测量证明有收益后启用的性能优化                              |

## 选型约束

### 数据访问

- **bun**：采用查询构建器风格，SQL 由开发者显式构造。相比 gorm，更贴近 SQL、行为更可预测，也能减少隐式行为带来的排查成本。
- **pgx/v5/stdlib**：通过 `database/sql` 适配器接入 bun。连接参数与错误类型统一使用 pgx v5；数据库错误通过 `*pgconn.PgError` 的 SQLSTATE 判断，不比较错误文本。
- **PostgreSQL**：作为默认关系型数据库，提供 JSONB、部分索引等完整能力，并适合使用 UUID 主键。

数据访问的分层方式、事务边界和 Repo 约定见《[[Go 单体应用架构规范]]》；表结构与字段约定见《[[PostgreSQL 数据库设计规范]]》。

### 依赖组装

默认使用手写组合根，通过构造函数显式组装长生命周期组件，不为每个组件预先创建接口。只有实际依赖图的复杂度证明代码生成能带来净收益时，才讨论引入依赖注入工具。

### 认证

采用 go-paseto 实现长短双 token：

- **access token**：短期有效，参考有效期约 15 分钟，用于业务接口鉴权。
- **refresh token**：长期有效，参考有效期约 7 天，仅用于换取新的 token 对。
- **轮换**：刷新成功后立即废弃旧 refresh token，并签发新的 refresh token；旧 token 不得再次使用，以降低重放风险。

不选 JWT 的主要原因是其算法协商曾长期成为错误配置与漏洞来源；PASETO 使用固定的协议版本和算法组合，约束更明确，且可在本地完成验证。

### 配置与日志

- **viper**：统一管理配置文件、环境变量与 flag，并允许环境变量覆盖其他来源。
- **zerolog**：统一输出结构化日志；默认使用 JSON，便于日志采集与检索。

### 测试

testify 用于断言，testcontainers 用于启动真实 PostgreSQL，执行 Service + Repo 集成测试；数据库访问不使用 mock。测试分层与替身边界见《[[Go 测试规范]]》。

### API 文档

huma 基于代码定义生成 OpenAPI 3.1 文档，同时为请求参数提供自动校验和统一的错误响应格式。生成的 OpenAPI spec 统一放在 `docs/openapi/`，不挂载到线上系统。

### CLI

使用 cobra 组织同一个后端二进制的子命令。应用根目录的 `main.go` 只调用 `cmd.Execute()`；`cmd/` 只负责命令注册、参数绑定与退出码，实际工作委托给组合根或专用用例。不得在 Cobra command 中解析业务数据或直接访问数据库。

## 按需能力

ristretto 不是默认依赖。只有指标或压测证明缓存能解决实际性能问题时才启用；缓存未命中、写入被拒绝或条目被淘汰都不得影响业务正确性。具体策略见《[[Go 单体应用架构规范]]》。
