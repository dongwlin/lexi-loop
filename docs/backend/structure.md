# Go 工程结构（Backend Structure）

> 本文档描述**领域设计如何映射到 Go 分层架构**：CLI 入口、目录组织、职责边界、核心接口与算法模块。它不定义业务规则本身——「权重公式是什么」见 [review/algorithm.md](../review/algorithm.md)，「权重公式实现放哪里」才在本文档。
> 通用约束分别见 [Go 单体应用架构规范](../specs/backend/Go%20单体应用架构规范.md)、[Go 技术栈](../specs/backend/Go%20技术栈.md)、[Go 测试规范](../specs/backend/Go%20测试规范.md) 与 [HTTP API 设计规范](../specs/backend/HTTP%20API%20设计规范.md)；本文档是这些规范在 LexiLoop 中的具体落地。

## 1. 概述

项目采用 **pnpm monorepo** 架构，后端应用位于 `apps/server/`。`apps/server/main.go` 是唯一可执行入口，使用 Cobra 将 HTTP 服务、ECDICT 导入和数据库迁移组织成不同 CLI 命令；`cmd/` 只负责命令路由、参数绑定与退出码，不承载业务逻辑。

后端内部采用分层架构，依赖方向单向流动：

```text
main → cmd → app / importer / migrations（迁移）
app → handler / service / infra（组装）
handler/router → handler/v1 / handler/middleware
handler/v1 → service / httpresp / apperr
handler/middleware → service（按需）/ httpresp
service → repo / domain / apperr / 外部端口（按需）
repo → domain / repo/internal/schema
importer → repo / domain
```

核心思路：Domain 封装不依赖外部资源的业务规则；Repo 是仅依赖 `bun.IDB` 的无状态临时适配器，由 Service 或 Importer 在方法内按需构造；Service 负责业务用例编排、事务边界和预期错误映射；Importer 负责编排离线数据导入；Handler 负责 HTTP 协议适配，Middleware 作为 Handler 子包处理 HTTP 横切逻辑。API 版本只作用于 Handler 与 DTO，Service 默认不随传输协议版本复制。

## 2. `apps/server/` 目录结构

```text
apps/server/
├─ main.go                       Cobra 程序入口：只调用 cmd.Execute()
├─ cmd/                          CLI 路由层，不放业务逻辑
│  ├─ root.go                    根命令、全局参数与退出码
│  ├─ serve.go                   启动 HTTP 服务
│  ├─ import_ecdict.go           调用 importer/ecdict 执行离线导入
│  └─ migrate.go                 执行数据库迁移
├─ migrations/                   PostgreSQL 版本化迁移
│  ├─ migrations.go              迁移执行入口：embed 内嵌 SQL + golang-migrate
│  ├─ 000001_dictionary.up.sql
│  ├─ 000001_dictionary.down.sql
│  ├─ 000002_review.up.sql
│  └─ 000002_review.down.sql
├─ internal/
│  ├─ app/                       组合根（显式组装、Server 生命周期）
│  │  ├─ provider.go             长生命周期组件的构造与显式组装
│  │  └─ server.go               调用 handler.RegisterRoutes、HTTP Server 启停与优雅关闭
│  ├─ domain/                    领域模型与纯业务规则，无 ORM/基础设施依赖
│  │  ├─ dictionary.go           DictionaryEntry、释义取值策略
│  │  ├─ normalization.go        纯字符串归一规则
│  │  ├─ user_word.go            UserWord、复习累计状态、weight/mastery
│  │  ├─ review.go               ReviewSession、ReviewItem 及状态迁移
│  │  └─ errors.go               领域 sentinel / typed errors
│  ├─ repo/                      无状态数据访问适配器（仅依赖 bun.IDB）
│  │  ├─ dictionary.go           查 / 写 dictionary_entries、lemma 查询
│  │  ├─ user_word.go            查 / 写 user_words
│  │  ├─ review.go               查 / 写 review_sessions / review_items
│  │  ├─ errors.go               Repo 可识别错误
│  │  └─ internal/
│  │     └─ schema/              bun ORM 映射（仅 repo 子树可导入）
│  │        ├─ dictionary.go
│  │        ├─ user_word.go
│  │        ├─ review.go
│  │        └─ jsonb.go          jsonb 列的 Valuer / Scanner 通用包装
│  ├─ service/                   业务用例编排（具体类型，不按 API 版本复制）
│  │  ├─ word.go                 WordService
│  │  ├─ dictionary.go           DictionaryService（MVP：本地 Lookup）
│  │  ├─ review.go               ReviewService
│  │  ├─ tx.go                   可重放事务的有上限重试与内部错误包装
│  │  └─ sampler.go              具体的 WeightedSampler
│  ├─ importer/
│  │  └─ ecdict/                 ECDICT CSV 适配器与导入用例
│  │     ├─ parser.go             CSV 解析与源字段映射
│  │     └─ importer.go           批处理、事务与进度编排
│  ├─ handler/
│  │  ├─ router.go               全局中间件、HTTP 兜底及 `/api/v1` 业务路由的唯一挂载入口
│  │  ├─ httpresp/               统一响应与唯一的 apperr.Kind → HTTP 状态映射
│  │  │  └─ response.go
│  │  ├─ v1/
│  │  │  ├─ word.go              词 Handler
│  │  │  ├─ review.go            复习 Handler
│  │  │  └─ dto/
│  │  │     ├─ word.go            词请求 / 响应 DTO
│  │  │     └─ review.go          复习请求 / 响应 DTO
│  │  └─ middleware/             HTTP 横切（Handler 子包，不版本化）
│  │     ├─ cors.go
│  │     ├─ logger.go
│  │     ├─ recovery.go
│  │     └─ request_id.go
│  ├─ apperr/                    类型化应用错误
│  │  └─ error.go
│  └─ infra/                     纯技术组件，由组合根组装
│     ├─ config/
│     ├─ database/
│     │  └─ database.go          连接池构造
│     └─ logger/
│        └─ logger.go            zerolog ConsoleWriter 日志初始化（含全局接管）
└─ go.mod
```

所有实体主键统一使用 **UUID v7**，由 Domain 工厂 / 构造函数在创建实体时生成，并初始化 UTC 时间戳；Domain 使用 UUID 值类型，schema 使用 PostgreSQL `uuid`。API 将 UUID 序列化为字符串。纯关联表若后续出现，可依其领域模型使用联合主键。

`migrations/` 是数据库结构的唯一可执行落点：字段语义仍以领域数据模型文档为准，迁移负责把它们实现为 PostgreSQL 类型、外键、唯一约束、CHECK 与索引。应用启动不隐式执行迁移；由 `migrate` 命令显式执行，ECDICT 导入前必须先迁移到所需版本。

## 3. CLI 与运行入口

`main.go` 只调用 `cmd.Execute()`。Cobra 的 `cmd/` 是 CLI 协议层，与 HTTP Handler 类似，只做参数解析、命令选择、上下文和退出码处理：

```text
lexi-loop serve                 启动 HTTP API
lexi-loop migrate up            执行数据库迁移
lexi-loop migrate down          回退明确指定的迁移步数
lexi-loop import-ecdict <file>  将 ECDICT CSV 导入 dictionary_entries
```

- `serve` 调用 `internal/app` 完成组件组装并管理服务生命周期。
- `migrate` 调用 `migrations` 包的迁移执行入口（embed 内嵌 SQL + golang-migrate），只操作 `migrations/` 中的版本化 SQL，不在 Go 代码中另存一份 schema。
- `import-ecdict` 调用 `internal/importer/ecdict`；Cobra 命令文件不解析 CSV、不直接构造 SQL。
- `cmd` 可以依赖 `app`、`importer` 和基础设施构造函数；任何业务包不得反向依赖 `cmd`。
- `app` 只是最外层组合根；Handler、Service、Repo、Domain、Importer 和 Infra 都不得反向导入 `app`。

组合根只管理长生命周期组件：`*bun.DB`、配置与日志等基础设施、具体 Service、版本化 Handler，以及未来真实需要替换的外部端口实现。Repo、Domain 和 schema 不进入长生命周期依赖图；Middleware 在 `handler/middleware/` 中定义，由 `handler/router.go` 构造并挂载。默认手写构造函数，不为了 DI 预先给每个组件创建接口。

## 4. 各层职责边界

### 4.1 domain

Domain 结构体不携带 bun struct tag 或 `bun.BaseModel`，并承载不依赖外部资源的规则。字段定义见 [dictionary/data-model.md](../dictionary/data-model.md) 与 [review/data-model.md](../review/data-model.md)。可被多层引用的 session / item / result 状态使用 Domain 类型化常量，不在 Service、Repo 和 Handler 中散落字符串字面量。

LexiLoop 的规则归属明确如下：

| 规则 | 实现位置 |
| --- | --- |
| trim / lowercase 等纯字符串归一 | `domain/normalization.go` |
| lemma 候选查询与“不确定不归并”流程 | `DictionaryService` 编排，查询交给 Dictionary Repo |
| `custom ?? review ?? raw` 释义取值 | `domain/dictionary.go` 的纯领域函数 |
| review / remember / forget / streak 更新 | `UserWord.ApplyReview(...)` |
| weight / mastery 动态计算 | `UserWord.ReviewWeight(now)` / `UserWord.MasteryScore()` |
| item 的 pending → 已作答 | `ReviewItem.Submit(...)` |
| session 的完成 / 放弃 | `ReviewSession.Complete(...)` / `ReviewSession.Abandon(...)` |
| 实体初始不变量、UUID v7 与 UTC 时间戳 | 对应的 `domain.NewXxx(...)` 工厂 / 构造函数 |

领域不变量错误定义在 domain 包内，供 Service 通过 `errors.Is` / `errors.As` 识别。Domain 方法只操作自身字段，不访问数据库、缓存或其他外部资源。数据库查询、抽样随机源以及同一用例需共享的 `now` 由 Service 取得并显式传入；实体首次创建所需的 UUID v7 与 UTC 创建时间则由 Domain 构造函数初始化。

### 4.2 repo

Repo 不持有长期状态，仅在 Service 或 Importer 方法内按需创建，构造函数只接收 `bun.IDB`。因此同一个 Repo 可以透明接受 `*bun.DB` 或 `bun.Tx`。

Repo 返回 domain 类型；数据库操作使用 schema 类型，`schema` ↔ `domain` 转换只在 Repo 内完成。数据访问错误归一化为 `ErrNotFound`、`ErrConflict`、`ErrAlreadyExists` 等可识别错误，数据库约束通过 SQLSTATE 判断，不比较驱动错误文本；未预期的基础设施错误保留原始 cause 交给 Service 包装。

并发所需的条件 UPDATE、行锁和 PostgreSQL advisory transaction lock 也封装在 Repo；Service 只表达“锁定 active-session 作用域”“提交仍为 pending 的 item”等意图，不直接写 SQL。

### 4.3 service

Service 使用具体类型，构造函数返回具体指针。请求/结果类型与 Service 放在同一个包中且不含 `json` 标签；JSON 契约只由版本化 DTO 定义。

Service 从 Repo 获取领域模型，调用 Domain 方法，再通过 Repo 写回。所有 PostgreSQL 操作委托给 Repo，Service 不直接构造 bun 查询、也不绕过 Domain 方法直接修改有不变量的字段。Service 负责用例事务、外部资源校验、随机抽样以及 Domain/Repo 错误到 `apperr.Error` 的映射。

数据访问分为两类：PostgreSQL 关系数据一律通过具体 Repo 访问；Redis、时钟、对象存储或第三方 API 等外部设施，只在出现真实需求时由调用方按业务能力定义窄端口，实现放在 `infra` 中。不把 Redis 或第三方客户端的通用 API 直接扩散到 Service。

跨 Service 的数据库用例不得开启彼此独立的嵌套事务。需要加入调用方事务的内部流程，在 `service` 包内提供接收 `bun.IDB` 的非导出方法；例如 `ImportWords` 在自己的事务中调用 `DictionaryService.lookup(ctx, tx, word)`，由该方法用同一个 `tx` 构造 Dictionary Repo。对外的 `Lookup` 再用 `s.db` 委托给同一内部实现，避免维护两套规则。

`DictionaryService` 的 MVP 职责是纯字符串归一 → lemma 解析 → 本地词典 Lookup → 未命中时创建最小词条。在线 Provider 的 Lookup 分支与 Enrich 均为 V2 能力，届时通过窄外部端口注入；ECDICT 始终不是运行时 Provider。

### 4.4 handler、router 与 httpresp

`app/server.go` 作为组合根创建 Gin Engine，并将日志、具体 Service 和版本化 Handler 传给 `handler.RegisterRoutes`。`handler/router.go` 是 HTTP 挂载的唯一入口：它构造并挂载全局 Middleware、注册 HTTP 兜底，再建立 `/api/v1` 业务路由组。这保留 LexiLoop 已有 API 契约的 `/api/v1/...` 路径，同时遵循通用规范中“中间件定义与挂载分离”的边界。

版本化 Handler 持有具体 Service，只负责参数绑定、基础格式校验、DTO 转换和 Service 调用；Domain 可作为 DTO 转换的读取来源，但不是 JSON 契约。Middleware 在 `handler/middleware/` 中定义，可按需依赖具体 Service，但不得包含业务规则，也不与具体业务 Handler 相互依赖。

所有成功/失败响应都经 `handler/httpresp` 输出；只有该包可以定义 `apperr.Kind → HTTP status` 映射。Handler 与 Middleware 不得各自复制响应结构或状态映射。

### 4.5 apperr 与 infra

`apperr` 不依赖 HTTP 或 Gin，集中维护稳定 code、错误 kind、安全 message 和原始 cause。Service 用 `errors.Is` / `errors.As` 将 Domain / Repo 的可识别错误映射为 `apperr.Error`；Handler 只识别类型化应用错误，未预期错误统一按内部错误处理，不向客户端暴露 cause。任何层都不得根据 `err.Error()` 文本分支。

`infra` 只包含 config、database 等技术组件，无业务逻辑，由组合根组装。未来增加 Redis、邮件、对象存储或第三方 Provider 时，具体适配器也归入 `infra`；业务能力窄端口仍由调用方定义。

## 5. 复习用例的事务与并发边界

### 5.1 `ReviewService.StartSession`

开始一轮必须在**一个事务**内完成：

```text
获取 active-session 作用域锁
→ 查找并锁定旧 active session
→ 如存在，调用 Abandon 并写回
→ 读取可复习候选词并按同一个 now 计算权重
→ 加权随机不放回抽样
→ 创建 active ReviewSession
→ 批量创建全部 pending ReviewItem
→ 提交事务后返回
```

MVP 单用户使用 Repo 封装的 PostgreSQL transaction-level advisory lock 串行化该流程；`migrations/` 同时创建“全库至多一个 active session”的部分唯一索引作为最终防线。V3 引入用户后，锁键和唯一约束都改为按 `user_id` 隔离。唯一约束冲突只能对整个无外部副作用的事务做有上限重试，不能只重试最后一条 INSERT。数据库返回死锁或序列化失败时，也只能对明确可重放的整个事务进行有上限、带抖动的重试。

候选数为 0 时事务不创建 session；请求数量超过候选数时仍按权威 API 契约截断。

### 5.2 `ReviewService.SubmitResult`

一次提交的 item 状态、词级累计数据和 session 汇总必须在**一个事务**内完成：

```text
先锁定 URL 指定的 ReviewSession，串行化同一 session 的提交
→ 按 session_id + item_id 锁定 ReviewItem
→ 不存在或归属不匹配：按 API 契约处理，不修改任何统计
→ 已非 pending：幂等返回当前状态，不重复计数
→ 调用 ReviewItem.Submit(result, now)
→ 条件 UPDATE ... WHERE id=? AND session_id=? AND result='pending'
  RETURNING user_word_id
→ 锁定对应 UserWord，调用 ApplyReview(result, now)，写回累计字段
→ 若已无 pending item，调用 Complete 并写回汇总
→ 提交事务
```

固定锁顺序为 `review_session → review_item → user_word`，所有提交路径都必须遵守。session 行锁使同一轮不同 item 的并发提交串行化，后一个事务能看到前一个事务已提交的 item 状态，因此不会发生“两个请求都看到另一个 item 仍 pending，最终无人完成 session”的竞态。

Repo 的条件 UPDATE 是持久化层并发保护，Domain 方法是业务状态迁移；两者必须同时保留。任一步骤失败都回滚，因此不会出现 item 已作答但 `user_words` 未计数、或最后一题已提交但 session 仍未完成的状态。相同 item 的重复请求若在并发中到达，会等待首个事务结束，随后命中“已非 pending”分支并幂等返回。

### 5.3 其他写用例

- `ImportWords`：单个请求使用一个事务；词条创建依赖 `headword` 唯一约束处理并发，任一项失败则整批回滚。
- `UpdateReviewMeaning`、`DeleteWord`：使用带业务条件的单条 UPDATE；若改为先读后写，必须增加行锁或乐观锁。
- ECDICT 导入：按固定批次提交，失败只回滚当前批次；Importer 记录可安全重跑的进度，写入以词条唯一键保持幂等。

### 5.4 LexiLoop 必测并发与回滚场景

通用分层测试方法以 [Go 测试规范](../specs/backend/Go%20测试规范.md) 为准；本项目额外必须使用 testcontainers + 真实 PostgreSQL 验证：并发开始两轮后全库只有一个 active session；同一 item 并发提交只累计一次；同一 session 的最后两个 item 并发提交后 session 必为 completed；在 item 更新后注入失败会回滚 item 与全部累计字段；创建 items 中途失败不会留下半成品 session。

## 6. 抽样与算法模块

权重和 mastery 是只依赖 `UserWord` 与显式时间参数的纯业务计算，因此按分层规范放在 Domain。`service/sampler.go` 只保留具体 `WeightedSampler`，负责把权重用于加权随机不放回抽样。

MVP 只有一个真实实现，不提前声明 `Sampler` 接口。若以后确有两种需要长期共存的抽样实现，再按调用方所需的最小能力抽取接口。为测试确定性而需要替换随机源时，将随机源作为 `WeightedSampler` 的构造参数，不为 Repo 或 Service 制造接口。

算法公式只在 [review/algorithm.md](../review/algorithm.md) 维护；实现及测试不得复制另一套说明。

## 7. 核心用例

```text
WordService：ImportWords()、ListWords()、GetWord()、UpdateReviewMeaning()、DeleteWord()
DictionaryService：Lookup()（MVP）；在线 Lookup 分支、Enrich()（V2）
ReviewService：StartSession()、SubmitResult()、GetSession()
ECDICT Importer：Import()
```

方法的 HTTP 契约见 [api/words.md](../api/words.md) 和 [api/reviews.md](../api/reviews.md)；Lookup / Enrich 的阶段边界见 [dictionary/enrichment.md](../dictionary/enrichment.md)。

## 8. ECDICT 导入

`internal/importer/ecdict` 负责 CSV 解析、源字段到 Domain 的映射、批处理和可重入导入；`cmd/import_ecdict.go` 只是它的 Cobra 入口。解析器属于离线输入适配器，不放进 Domain 或 Service。

ECDICT 是导入期数据源，运行时不读取 CSV。导入后 `dictionary_entries` 就是本地词典库本身，运行时查询只走数据库。详细设计见 [dictionary/enrichment.md](../dictionary/enrichment.md)。

## 9. MVP 与未来扩展

MVP 只实现本地词典 Lookup、Word API、随机复习、复习历史及上述 CLI 命令；不实现认证、在线 Provider、Enrich、缓存、队列或多用户。

- `auth.go`、Token Service 和 PASETO 认证属于 V3 多用户阶段，当前目录不创建占位文件。
- Free Dictionary Provider、在线 Lookup 分支和 Enrich 属于 V2，届时再增加具体实现与后台执行入口。
- `user_id` 属于 V3；当前通过 `UNIQUE(dictionary_entry_id)` 保证单用户的一词一行。
- tags、collections、decks、folders、user_word_relations、word_variants 不属于 MVP。

缓存仍默认关闭。只有指标或压测证明数据库读取是实际瓶颈，且业务能接受明确 TTL 内的陈旧时，才能在对应 Service 中引入 Ristretto；启用前必须定义容量 cost、TTL、失效键和 hit / miss / rejection / eviction 等观测指标。届时只缓存不可变值快照，每次命中都构造新的 Domain 值；事务成功提交后再失效相关键，并保证 miss、Set 被拒绝或条目被淘汰时仍以 PostgreSQL 为事实源。强一致读取不经过进程内缓存。

`dictionary_entries` / `user_words` 拆分、软删除、四表链路、Review Submit 幂等和单 active session 均是 MVP 必须实现的冻结决策，不能作为“后续优化”跳过。

## 10. 反模式提醒

| 反模式 | 正确做法 |
| --- | --- |
| 贫血模型，规则散落在 Service | 不依赖外部资源的规则放入 Domain 方法或纯领域函数 |
| schema 泄漏 | schema 限定在 `repo/internal/schema` |
| Handler / Cobra Command 写业务逻辑 | 委托 Service、Importer 或 Domain |
| Domain 方法查库或访问外部资源 | 外部资源校验由 Service 编排，Domain 只操作自身字段 |
| Repo 作为长生命周期单例注入 | 在 Service / Importer 方法内用当前 `bun.IDB` 即用即弃 |
| Service 直接构造 SQL | 所有 PostgreSQL 操作委托 Repo |
| 事务内混用 `s.db` 与 `tx` | 同一事务中的全部 Repo 都使用传入的 `bun.Tx` |
| Review item 与累计统计分开提交 | `SubmitResult` 的全部状态变化使用同一事务 |
| 先查 active 再无保护地创建 | advisory transaction lock + 部分唯一索引 |
| Service 直接修改 Domain 字段 | 调用 Domain 方法执行状态迁移和不变量检查 |
| 状态字符串散落在各层 | 在 Domain 定义类型化常量，Repo 集中转换 |
| 按错误文本分支 | `errors.Is` / `errors.As` + `apperr.Kind/Code` |
| 各 Handler 自行映射 HTTP 状态 | 统一经过 `handler/httpresp` |
| 仅为未来替换创建接口 | 出现真实多实现或窄外部边界时再抽取接口 |
| API 版本复制 Service | 只版本化 Handler/DTO，复用业务用例 |
| 缓存可变 Domain 指针 | 缓存不可变值快照，每次命中返回新值 |
