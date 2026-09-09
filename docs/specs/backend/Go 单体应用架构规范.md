# Go 单体应用架构规范

## 1. 概述

> 本规范在 LexiLoop 中的具体落地见 [Go 工程结构](../../backend/structure.md)。

本规范定义 Go 单体应用的分层结构：依赖方向清晰，不为形式上的”解耦”引入无收益的接口、版本目录和转换层。

核心思路：**Repo 是仅依赖 `bun.IDB` 的无状态临时适配器，由 Service 在方法内按需构造；Repo 内部维护 `repo/internal/schema` 并完成 `schema` ↔ `domain` 转换，利用 Go 的 `internal` 规则阻止其他层导入。Domain 封装不依赖外部资源的业务规则；`service` 包提供业务接口与用例类型，`service/v1` 子包实现用例编排、事务边界和预期错误映射；Handler 依赖接口，负责 HTTP 协议适配，可注入 mock 独立测试。Service 实现版本与 HTTP API 版本独立，默认不随传输协议版本复制。预期错误通过类型化应用错误传递，任何层都不得依赖 `err.Error()` 文本进行分支。缓存默认关闭，仅在测量证明有收益后作为允许未命中、允许被淘汰的优化加入。**

---

## 2. 分层架构

项目采用多层结构（`schema` 是 Repo 内部的私有子层，不算独立对外层；`infra` 是纯基础设施层，不含业务逻辑）：

| 层          | 职责                                      | 依赖                            |
| ---------- | --------------------------------------- | ----------------------------- |
| middleware | HTTP 横切关注点（auth、cors、recovery、logger 等），handler 的子包 | Service（按需）                  |
| handler    | 处理 HTTP 请求和响应；按 API 版本组织 Handler 与 DTO          | Service 接口               |
| service    | 业务接口、用例请求/结果类型                  | 标准库、Domain、稳定值类型          |
| service/v1 | Service 接口的具体实现：用例编排、事务、预期错误映射 | service、`*bun.DB`、Repo、Domain、apperr、外部端口接口 |
| repo       | 纯粹的数据访问，通过 `bun.IDB` 执行数据库操作            | `bun.IDB`、Domain、schema（私有）   |
| schema     | bun ORM 映射的数据结构，位于 `repo/internal/schema`  | bun                            |
| domain     | 领域模型：数据字段 + 操作自身字段的业务方法；sentinel error 定义于此 | 无 Web、ORM、数据库或基础设施依赖       |
| apperr     | 类型化应用错误：稳定 code、错误 kind、安全 message、原始 cause | 标准库                            |
| infra      | 基础设施：config/database/redis/mail/logger/storage/asynq 等 | 无业务依赖，由组合根组装          |

### 2.1 internal/ 目录结构

```
internal/
├── app/                        // 组合根（显式组装、Server 生命周期）
│   ├── provider.go             // 组件构造函数（由组合根显式组装）
│   └── server.go
├── handler/                    // HTTP 层
│   ├── router.go               // 路由入口：注册中间件、兜底与 v1 路由组
│   ├── v1/                     // 业务 handler 与 DTO（按 API 版本组织）
│   │   ├── user.go
│   │   ├── order.go
│   │   └── dto/                // 请求/响应 DTO 与转换函数
│   │       ├── user.go
│   │       └── order.go
│   └── middleware/             // HTTP 横切（handler 子包，不版本化）：auth、cors、recovery、logger
│       ├── auth.go
│       ├── cors.go
│       └── logger.go
├── service/                    // 业务接口 + 用例请求/结果类型
│   ├── user.go
│   ├── order.go
│   └── v1/                     // 具体实现（实现版本独立于 HTTP API 版本）
│       ├── user.go
│       └── order.go
├── apperr/                     // 类型化应用错误，不包含 HTTP/Gin 依赖
│   └── error.go
├── repo/                       // 数据访问（无状态临时适配器）
│   ├── user.go
│   ├── order.go
│   └── internal/
│       └── schema/             // 持久化模型，仅 repo 子树可导入
│           ├── user.go
│           └── order.go
├── domain/                     // 领域模型（数据 + 行为，无基础设施依赖）
│   ├── user.go
│   ├── order.go
│   └── order_item.go
└── infra/                      // 基础设施（config/database/redis/mail/logger/storage/asynq）
    ├── config/
    ├── database/
    ├── redis/
    ├── mail/
    ├── logger/
    ├── storage/
    └── asynq/
```

各层之间的依赖方向：

```
app → handler/v1 → service → domain
app → service/v1 → service
service/v1 → repo → domain
service/v1 → apperr
handler/v1 → apperr
app → infra（组装）
handler/middleware → service（按需）
repo → repo/internal/schema（由 Go internal 规则限制导入）
```

依赖方向是单向的：Handler 只依赖 `service` 接口包，不导入 `service/v1`；具体实现依赖接口包、Repo、Domain 与 `apperr`。`service` 接口包不得反向导入实现子包、Repo 或基础设施，避免循环依赖和持久化细节泄漏；Repo 知道 Domain 以及私有 schema。Domain 不依赖应用层。Middleware 是 Handler 的子包，与具体业务 Handler 互不依赖。`app` 是组合根，其他包不得导入 `app`。

### 2.2 组合根管理的组件

组合根管理**长生命周期**组件：

- Handler（v1 业务 handler 与路由入口）
- Service 实现（在 `service/v1` 中构造，按 `service.User` 等接口注入 Handler）
- `*bun.DB`、`*redis.Client` 等基础设施组件（infra 层产物）
- 邮件、时钟、对象存储、第三方 API 等需要替换的外部端口实现

以下组件不作为长生命周期依赖注入：

- **Repo**：构造成本接近于零且无长期状态，在 Service 方法内按需创建。若未来确有多个实现或明确的独立替换需求，再引入接口，不提前抽象
- **schema**：Repo 的私有实现细节，随 Repo 构造而存在
- **Domain**：纯数据结构，由 Repo 查询返回或 Service 中直接构造
- **Middleware**：在 `handler/middleware/` 子包中定义，由 `handler/router.go` 构造并注入路由链，生命周期等同于应用生命周期

默认使用手写构造函数完成组装。只有当依赖图的实际复杂度证明代码生成有净收益时，才单独讨论引入 DI 工具；规范不要求所有组件都创建接口。

---

## 3. Domain 层设计

### 3.1 纯领域模型 + 充血模型

Domain 层采用**纯领域结构 + 充血模型（Rich Domain Model）**：结构体只定义数据字段，**不携带任何 ORM 映射**（无 bun struct tag、无 `bun.BaseModel`），同时承载操作自身数据的业务方法。

数据持久化映射由 `repo/internal/schema` 承担，Repo 负责在两者之间转换。Domain 不依赖 Web、ORM、数据库客户端或基础设施；允许使用 UUID 等稳定值类型库。

```go
// internal/domain/user.go
package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

// 领域不变量错误以 sentinel error 定义在 domain 包内，供 Service 层 errors.Is 断言
var (
    ErrAlreadyActive   = errors.New("user already active")
    ErrAlreadyInactive = errors.New("user already inactive")
    ErrUserBanned      = errors.New("user is banned")
    ErrNameRequired    = errors.New("name is required")
    ErrEmailRequired   = errors.New("email is required")
)

type User struct {
    ID        uuid.UUID
    Name      string
    Email     string
    Status    string // "active", "inactive", "banned"
    Level     int
    Version   int64
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Domain 与 schema 字段可以一一对应，也可以由 Repo 在转换时做类型提升（如 `Status` 字符串 ↔ 枚举类型）。转换细节全部收敛在 Repo 内部，Domain 不感知持久化形态。

### 3.2 模型方法的四类职责

模型上的方法限定于**不依赖外部资源**的操作，分以下四类：

#### 状态转换

封装实体状态机，确保状态变更必须通过方法执行不变量检查，失败返回 sentinel error：

```go
// Activate 激活用户
func (u *User) Activate() error {
    if u.Status == "banned" {
        return ErrUserBanned
    }
    if u.Status == "active" {
        return ErrAlreadyActive
    }
    u.Status = "active"
    return nil
}

// Deactivate 停用用户
func (u *User) Deactivate() error {
    if u.Status == "banned" {
        return ErrUserBanned
    }
    if u.Status == "inactive" {
        return ErrAlreadyInactive
    }
    u.Status = "inactive"
    return nil
}

// Ban 封禁用户
func (u *User) Ban() error {
    if u.Status == "banned" {
        return ErrUserBanned
    }
    u.Status = "banned"
    return nil
}
```

#### 工厂 / 构造函数

创建实体时校验初始状态的合法性，并生成 UUID v7 主键与 UTC 时间戳：

```go
// NewUser 创建用户，校验必填字段
func NewUser(name, email string) (*User, error) {
    if name == "" {
        return nil, ErrNameRequired
    }
    if email == "" {
        return nil, ErrEmailRequired
    }
    // 此处可加邮箱格式等基本校验
    return &User{
        ID:        uuid.Must(uuid.NewV7()),
        Name:      name,
        Email:     email,
        Status:    "active",
        Level:     1,
        Version:   1,
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
    }, nil
}
```

#### 查询 / 计算

基于自身字段的派生值：

```go
// IsVIP 是否 VIP 用户
func (u *User) IsVIP() bool {
    return u.Level >= 10
}

// CanPlaceOrder 是否可以下单
func (u *User) CanPlaceOrder() bool {
    return u.Status == "active"
}
```

#### 字段变更

对具有业务规则的字段，通过方法封装变更逻辑：

```go
// SetEmail 变更邮箱
func (u *User) SetEmail(email string) error {
    if email == "" {
        return ErrEmailRequired
    }
    // 此处可加邮箱格式校验
    u.Email = email
    return nil
}
```

#### 错误定义约定

- **调用方需要识别的预期错误**（状态机、规则冲突、必填字段、格式）：使用 sentinel error 或携带结构化字段的自定义错误，Service 层通过 `errors.Is` / `errors.As` 判断
- **不可作为业务分支依据的内部错误**：可以包装原始 cause，但不得要求上层比较错误文本
- `Error()` 文本用于日志和调试，可以调整；稳定契约是错误类型、sentinel 身份和应用错误 `code`

### 3.3 模型方法的边界

**模型方法只操作自身 struct 的字段，绝不访问数据库、缓存或任何外部依赖。**

需要查库的逻辑（如邮箱唯一性校验）留在 Service 层：

```go
// ❌ 错误：模型方法里做数据库查询
func (u *User) ChangeEmail(email string, repo *UserRepo) error { ... }

// ✅ 正确：Service 做外部资源校验，Domain 做自身数据校验
// Service 中：
func (s *UserService) ChangeEmail(ctx context.Context, id uuid.UUID, newEmail string) error {
    repo := NewUserRepo(s.db)
    // 外部依赖校验 — Service 的职责
    if exists, _ := repo.ExistsByEmail(ctx, newEmail); exists {
        return ErrEmailTaken
    }
    user, _ := repo.FindByID(ctx, id)
    // 自身状态校验 — Domain 的职责
    if err := user.SetEmail(newEmail); err != nil {
        return err
    }
    return repo.Update(ctx, user)
}
```

分界线很明确：**是否需要访问外部资源（DB / 缓存 / 其他模块）。需要 → Service；不需要 → Domain。**

---

## 4. Repo 层设计

### 4.1 无状态临时适配器

Repo 不持有任何长期状态，仅在 Service 方法内按需创建，生命周期仅限于当前调用：

```go
// internal/service/v1/user.go（完整结构见第 5 章）
func (s *User) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
    userRepo := repo.NewUserRepo(s.db)
    return userRepo.FindByID(ctx, id)
}
```

### 4.2 只依赖 bun.IDB

Repo 构造函数只接收 `bun.IDB` 接口，不依赖具体实现：

```go
type UserRepo struct {
    db bun.IDB
}

func NewUserRepo(db bun.IDB) *UserRepo {
    return &UserRepo{db: db}
}
```

这使 Repo 可以透明地接受 `*bun.DB`（普通查询）或 `bun.Tx`（事务），无需任何修改。

### 4.3 schema 子包（repo 私有）

Repo 内部维护一个名为 `schema` 的子包，专门承载 bun ORM 映射的持久化结构：

```go
// internal/repo/internal/schema/user.go
package schema

import (
    "time"

    "github.com/google/uuid"
    "github.com/uptrace/bun"
)

// User 是 users 表的 bun ORM 映射，只描述持久化形态，不含任何业务逻辑
type User struct {
    bun.BaseModel `bun:"table:users,alias:u"`

    ID        uuid.UUID `bun:",pk,type:uuid"`
    Name      string    `bun:"name"`
    Email     string    `bun:"email"`
    Status    string    `bun:"status"` // "active", "inactive", "banned"
    Level     int       `bun:"level"`
    Version   int64     `bun:"version,notnull"`
    CreatedAt time.Time `bun:",notnull,type:timestamptz,default:now()"`
    UpdatedAt time.Time `bun:",notnull,type:timestamptz,default:now()"`
}
```

schema 子包的约束：

- **私有性**：schema 位于 `repo/internal/schema`，Go 编译器会阻止 `repo` 目录树之外的 service、handler、domain 导入
- **无业务逻辑**：schema 结构只描述数据映射，不包含业务方法、不定义领域错误
- **转换内聚**：`schema` ↔ `domain` 的双向转换只在 Repo 内部完成，可将转换封装为 Repo 的私有方法（如 `toDomain` / `toSchema`）避免重复

### 4.4 Repo 返回 Domain 类型

Repo 的查询方法返回 `domain` 包中定义的领域模型指针；写入方法接收 `*domain.User`。数据库操作使用 `schema` 类型，转换在 Repo 内部完成：

```go
func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
    s := new(schema.User)
    if err := r.db.NewSelect().Model(s).Where("id = ?", id).Scan(ctx); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, err
    }
    return &domain.User{
        ID:        s.ID,
        Name:      s.Name,
        Email:     s.Email,
        Status:    s.Status,
        Level:     s.Level,
        Version:   s.Version,
        CreatedAt: s.CreatedAt,
        UpdatedAt: s.UpdatedAt,
    }, nil
}

func (r *UserRepo) Insert(ctx context.Context, u *domain.User) error {
    s := &schema.User{
        ID:        u.ID,
        Name:      u.Name,
        Email:     u.Email,
        Status:    u.Status,
        Level:     u.Level,
        Version:   u.Version,
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
    }
    _, err := r.db.NewInsert().Model(s).Exec(ctx)
    return err
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
    result, err := r.db.NewUpdate().
        Model((*schema.User)(nil)).
        Set("name = ?", u.Name).
        Set("email = ?", u.Email).
        Set("status = ?", u.Status).
        Set("level = ?", u.Level).
        Set("version = version + 1").
        Set("updated_at = ?", u.UpdatedAt).
        Where("id = ?", u.ID).
        Where("version = ?", u.Version).
        Exec(ctx)
    if err != nil {
        return err
    }
    affected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if affected == 0 {
        return ErrConflict
    }
    u.Version++
    return nil
}
```

Repo 是对外唯一的数据访问入口，Service 只通过 Repo 与数据库交互，Service 感知不到 `schema` 的存在。

### 4.5 数据访问错误归一化

Repo 不把 `sql.ErrNoRows`、驱动错误文本或 PostgreSQL 约束名直接交给 Service 判断。Repo 包集中定义少量稳定错误，例如：

```go
var (
    ErrNotFound          = errors.New("repository: not found")
    ErrConflict          = errors.New("repository: concurrent update conflict")
    ErrAlreadyExists     = errors.New("repository: unique constraint violation")
    ErrReferenceViolation = errors.New("repository: foreign key violation")
)
```

- 查无记录统一包装为 `repo.ErrNotFound`
- 乐观锁版本不匹配统一返回 `repo.ErrConflict`
- 唯一约束、外键错误在 Repo 内按错误类型或 SQLSTATE 分别归一化为 `ErrAlreadyExists`、`ErrReferenceViolation`，不比较驱动错误文本
- 未预期的基础设施错误保留 cause 并向上返回，由 Service 包装为内部应用错误

---

## 5. Service 层设计

### 5.1 Service 提供接口，v1 子包提供实现

`internal/service` 只定义对外业务接口与用例请求/结果类型，不保存数据库连接或实现业务流程。接口按业务能力划分，避免把所有用例合成一个大接口；涉及 I/O 或可取消工作的用例方法接收 `context.Context`；纯内存只读快照等操作可省略上下文。参数和返回值使用 Domain 或本包用例类型，不暴露 Gin、HTTP DTO、bun 或 Repo 类型。请求/结果类型不含 `json` 标签，JSON 契约由 Handler 的 DTO 定义。

```go
// internal/service/user.go
package service

import (
    "context"

    "yourproject/internal/domain"

    "github.com/google/uuid"
)

type User interface {
    GetProfile(ctx context.Context, id uuid.UUID) (*domain.User, error)
    ActivateUser(ctx context.Context, id uuid.UUID) error
}

type UpdateProfileRequest struct {
    Nickname *string
    Avatar   *string
}
```

具体结构体、构造函数与方法放在 `internal/service/v1`。构造函数返回具体指针，组合根通过接口注入调用方；使用编译期断言确保实现满足接口。用例类型只在父包定义，子包通过 `service.XxxRequest` / `service.XxxResult` 引用，不复制定义。

```go
// internal/service/v1/user.go
package v1

import (
    "context"
    "errors"

    "yourproject/internal/apperr"
    "yourproject/internal/domain"
    "yourproject/internal/repo"
    "yourproject/internal/service"

    "github.com/google/uuid"
    "github.com/uptrace/bun"
)

type User struct {
    db *bun.DB
}

var _ service.User = (*User)(nil)

func NewUser(db *bun.DB) *User {
    return &User{db: db}
}

func (s *User) GetProfile(ctx context.Context, id uuid.UUID) (*domain.User, error) {
    user, err := repo.NewUserRepo(s.db).FindByID(ctx, id)
    if errors.Is(err, repo.ErrNotFound) {
        return nil, apperr.New(apperr.NotFound, apperr.CodeUserNotFound, "user not found", err)
    }
    if err != nil {
        return nil, apperr.Internal(err)
    }
    return user, nil
}

// ActivateUser 的实现见 §5.3，同样位于本包。
```

组合根同时导入 Handler 与 Service 实现时使用别名区分：

```go
// internal/app/provider.go
package app

import (
    handlerv1 "yourproject/internal/handler/v1"
    servicev1 "yourproject/internal/service/v1"

    "github.com/uptrace/bun"
)

func NewUserHandler(db *bun.DB) *handlerv1.UserHandler {
    userSvc := servicev1.NewUser(db)
    return handlerv1.NewUserHandler(userSvc)
}
```

`service/v1` 的 `v1` 表示业务实现版本，不与 HTTP `/v1` 绑定。`handler/v1` 和未来的 `handler/v2` 可以注入同一个实现；只有业务行为确实需要长期并存时，才引入新的实现子包，由组合根选择。父包接口保持独立，不因新增 API 版本机械复制。

Service 接口是 Handler mock 测试的替换边界，属于明确的测试需求。Repo 仍使用具体类型，不为 mock Handler 额外抽象 Repo。邮件、时钟、对象存储等外部端口仍按所需能力声明窄接口；仅供实现使用的端口可定义于实现包，避免加入面向 Handler 的业务接口。

### 5.2 职责边界

下文的 Service 编排、事务与缓存职责及具体方法示例均属于 `service/v1` 实现包；`service` 父包仅提供契约。

| 职责 | 说明 |
|------|------|
| 业务流程编排 | 组合多个 Repo 操作完成一个业务动作 |
| 领域规则委托 | 将操作自身数据的规则委托给 Domain 方法，Service 不做字段级校验 |
| 预期错误映射 | 用 `errors.Is` / `errors.As` 识别 Domain/Repo 错误，映射为带稳定 code/kind 的 `apperr.Error` |
| 事务边界控制 | 通过 `s.db.RunInTx` 管理事务 |
| 缓存策略 | 读写缓存、缓存失效均在 Service 方法内完成 |
| 外部资源校验 | 唯一性检查、权限校验等需要访问外部资源的逻辑 |
| 不直接操作 SQL | 所有关系型数据库操作委托给 Repo |
| 不直接改 domain 字段 | 字段变更通过 Domain 方法执行，防止绕过业务规则 |

#### 两类数据访问

- **关系型数据**（PostgreSQL）：一律通过 Repo（`bun.IDB`）访问，Service 不直接构造 bun 查询
- **Redis 型数据**（验证码、限流计数等键值数据）：不经过关系型 Repo；Service 依赖按业务能力命名的窄接口，Redis 实现在 `infra/redis` 中完成

判定标准：数据存储于 PostgreSQL → 走具体 Repo；外部设施 → 通过能表达业务所需能力的窄端口访问，不把通用客户端 API 扩散到 Service：

```go
// internal/service/v1/verification.go：实现所需的外部端口
type VerificationCodeStore interface {
    Save(ctx context.Context, key, code string, ttl time.Duration) error
    Consume(ctx context.Context, key, code string) (bool, error)
}

type Verification struct {
    codes VerificationCodeStore
}
```

### 5.3 充血模型的 Service 写法

Service 从 Repo 获取领域模型 → 调用 Domain 方法执行业务规则 → 通过 Repo 写回：

```go
// ActivateUser 激活用户（internal/service/v1/user.go）
func (s *User) ActivateUser(ctx context.Context, id uuid.UUID) error {
    userRepo := repo.NewUserRepo(s.db)
    user, err := userRepo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, repo.ErrNotFound) {
            return apperr.New(apperr.NotFound, apperr.CodeUserNotFound, "user not found", err)
        }
        return apperr.Internal(err)
    }
    if err := user.Activate(); err != nil {
        if errors.Is(err, domain.ErrUserBanned) {
            return apperr.New(apperr.FailedPrecondition, apperr.CodeUserDisabled, "user is disabled", err)
        }
        return apperr.New(apperr.Conflict, apperr.CodeUserStateConflict, "user state conflict", err)
    }
    user.UpdatedAt = time.Now().UTC()
    if err := userRepo.Update(ctx, user); err != nil {
        if errors.Is(err, repo.ErrConflict) {
            return apperr.New(apperr.Conflict, apperr.CodeConcurrentUpdate, "resource was modified", err)
        }
        return apperr.Internal(err)
    }
    return nil
}
```

对比贫血模型的写法（Service 直接操作字段 + if-else 校验），充血模型中 Service 不关心"什么条件下可以激活"——这个知识由 `User.Activate()` 封装。

---

## 6. Handler 层设计

### 6.1 结构体定义

业务 Handler 位于 `handler/v1/`，持有 `service` 接口，负责 HTTP 协议适配。Handler 与 DTO 按 API 版本组织；通用 Middleware 不参与版本化：

```go
// internal/handler/v1/user.go
package v1

import (
    "yourproject/internal/service"

    "github.com/gin-gonic/gin"
)

type UserHandler struct {
    svc service.User
}

func NewUserHandler(svc service.User) *UserHandler {
    return &UserHandler{svc: svc}
}
```

### 6.2 职责边界

| 职责 | 说明 |
|------|------|
| 参数绑定与校验 | HTTP 请求 → 请求 DTO，基础格式校验 |
| DTO 转换 | 请求 DTO → service 参数；service 返回的 domain → 响应 DTO |
| 调用 Service | 传入业务参数，获取结果或错误 |
| 错误映射 | 读取类型化应用错误的 kind/code，统一映射 HTTP 状态码（见第 7 章） |
| HTTP 响应渲染 | 经统一响应辅助函数返回 JSON |
| 不写业务逻辑 | 不做任何业务判断 |
| 不把 domain 当 API 契约 | 可读 domain 仅用于 DTO 转换，序列化形状一律由 DTO 定义 |

```go
func (h *UserHandler) ActivateUser(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        response.BadRequest(c, err)
        return
    }
    if err := h.svc.ActivateUser(c.Request.Context(), id); err != nil {
        response.Error(c, err)
        return
    }
    response.Success(c, nil)
}
```

### 6.3 DTO 子包设计

- 位置：`internal/handler/v1/dto/`，按实体/模块命名（`user.go`、`order.go`）
- 命名：请求 DTO `XxxRequest`（带 `json` 标签与 binding 校验），响应 DTO `XxxResponse` / `XxxItem`
- 转换：`toXxxItem(*domain.Xxx) XxxResponse` 等转换函数，字段逐个拷贝（可选字段用指针）
- 约束：DTO 是纯数据载体，不含业务逻辑

```go
// internal/handler/v1/dto/user.go
type UserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func ToUserResponse(u *domain.User) UserResponse {
    return UserResponse{
        ID:    u.ID.String(),
        Name:  u.Name,
        Email: u.Email,
    }
}
```

---

## 7. API 响应与错误码

API 响应结构与错误码是全站统一契约，Handler 与 Middleware 一律遵循。统一响应结构（`code` / `message` / `data`）、错误码格式与分类、HTTP 状态码映射、分页与列表等响应细节见《[HTTP API 设计规范](HTTP%20API%20设计规范.md)》。

### 7.1 响应实现约束

- 响应一律经统一辅助函数返回（如 `response.Success(c, data)` / `response.Error(c, httpStatus, code, message)`），不直接手写 `c.JSON`；流式/文件响应（如导出下载）不受此约束
- 稳定应用错误码集中在 `apperr` 包维护；HTTP 状态映射和响应写入集中在 handler 的 `httpresp` 子包

### 7.2 错误映射规则

跨层稳定契约是错误类型、`kind` 与 `code`，不是错误文本：

```
Domain / Repo（sentinel 或 typed error）
  → Service（errors.Is / errors.As，映射 apperr.Error）
  → Handler（按 kind 映射 HTTP 状态，原样返回稳定 code）
```

- **Domain / Repo**：返回可由 `errors.Is` / `errors.As` 识别的预期错误，并保留底层 cause
- **Service**：将预期错误映射为 `apperr.Error`；未预期错误包装为 `Internal`
- **Handler**：只识别 `apperr.Error`；非应用错误统一按 `500` 处理并记录 cause，不向客户端暴露内部细节
- **message**：只用于安全、可读的默认提示，可以调整或本地化，不得参与程序分支

`apperr` 不依赖 Gin 或 HTTP：

```go
package apperr

import "time"

type Kind uint8

const (
    InvalidArgument Kind = iota + 1
    Unauthenticated
    PermissionDenied
    NotFound
    Conflict
    FailedPrecondition
    RateLimited
    InternalKind
)

const (
    CodeInternal          = "ERROR"
    CodeTokenMissing      = "BASE.AUTH.TOKEN_MISSING"
    CodeUserNotFound      = "BASE.NOT_FOUND.USER"
    CodeUserDisabled      = "BASE.BIZ.USER_DISABLED"
    CodeUserStateConflict = "BASE.BIZ.USER_STATE_CONFLICT"
    CodeConcurrentUpdate  = "BASE.BIZ.CONCURRENT_UPDATE"
)

type Error struct {
    Kind       Kind
    Code       string
    Message    string
    RetryAfter time.Duration
    Err        error
}

func New(kind Kind, code, message string, cause error) *Error {
    return &Error{Kind: kind, Code: code, Message: message, Err: cause}
}

func Internal(cause error) *Error {
    return New(InternalKind, CodeInternal, "internal server error", cause)
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }
```

Handler 通过 `errors.As(err, &appErr)` 取得结构化信息；`httpresp` 按《[HTTP API 设计规范](HTTP%20API%20设计规范.md)》§11 实现唯一的 `kind → HTTP status` 映射。业务 `code` 一经发布保持稳定，新增或废弃时同步接口文档。

---

## 8. Middleware 层设计

### 8.1 定位与职责

Middleware 是 HTTP 层的横切关注点，作为 `handler` 的子包组织（`handler/middleware/`），独立于具体业务域，不参与版本化，处理到达 Handler 前的通用逻辑：

| 职责 | 说明 |
|------|------|
| 请求拦截 | 在请求到达 Handler 前执行（认证、限流等） |
| 上下文注入 | 将解析结果写入 `gin.Context`（如 `userID`） |
| 响应控制 | 校验不通过时直接中断请求链 |
| 不写业务逻辑 | 不做业务规则判断，业务逻辑一律委托给 Service |

### 8.2 按需依赖 Service 接口

Middleware 接收 `service` 包中需要的业务接口，不导入 `service/v1`；不要为了中间件额外复制一套接口。与外部设施交互的可替换端口仍由 Service 自己声明：

```go
// internal/handler/middleware/auth.go
package middleware

import (
    "strings"

    "yourproject/internal/apperr"
    "yourproject/internal/handler/httpresp"
    "yourproject/internal/service"

    "github.com/gin-gonic/gin"
)

// Auth 认证中间件 — 依赖 service.Token 接口（声明 ValidateToken 方法）
func Auth(tokenSvc service.Token) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if header == "" || !strings.HasPrefix(header, "Bearer ") {
            httpresp.AbortError(c, apperr.New(
                apperr.Unauthenticated,
                apperr.CodeTokenMissing,
                "missing token",
                nil,
            ))
            return
        }

        userID, err := tokenSvc.ValidateToken(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
        if err != nil {
            httpresp.AbortError(c, err)
            return
        }

        c.Set("userID", userID)
        c.Next()
    }
}
```

```go
// internal/handler/middleware/logger.go
package middleware

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog"
)

// Logger 请求日志中间件 — 纯技术组件，不依赖任何业务包
func Logger(log zerolog.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        log.Info().
            Str("method", c.Request.Method).
            Str("path", c.Request.URL.Path).
            Int("status", c.Writer.Status()).
            Dur("latency", time.Since(start)).
            Msg("request")
    }
}
```

### 8.3 定义与挂载分离

Middleware 的定义在 `handler/middleware/` 子包中，路由注册由 `handler/router.go` 完成。组合根在 `app/server.go` 中构造所有组件并调用 `handler.RegisterRoutes` 完成挂载：

```go
// internal/handler/router.go
package handler

import (
    "yourproject/internal/handler/v1"
    "yourproject/internal/handler/middleware"
    "yourproject/internal/service"

    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog"
)

func RegisterRoutes(
    r *gin.Engine,
    log zerolog.Logger,
    tokenSvc service.Token,
    userH *v1.UserHandler,
    orderH *v1.OrderHandler,
) {
    // 全局中间件 — 不需要 Service 依赖的
    r.Use(middleware.Logger(log))
    r.Use(middleware.Recovery())

    // 公开路由
    v1 := r.Group("/v1")
    v1.POST("/login", userH.Login)
    v1.POST("/register", userH.Register)

    // 需认证路由
    authorized := v1.Group("")
    authorized.Use(middleware.Auth(tokenSvc))
    {
        authorized.GET("/users/:id", userH.GetUser)
        authorized.POST("/orders", orderH.CreateOrder)
    }
}
```

### 8.4 依赖方向

Middleware 按需依赖 Service 接口，但 Service 不感知 Middleware。app 作为组合根，知道双方并完成装配。

```
handler/middleware → service → domain
app → service/v1 → service
service/v1 → repo → domain
```

Middleware 是 Handler 的子包，与具体业务 Handler 互不依赖；两者都可以按需依赖 Service 接口。

---

## 9. 事务处理

### 9.1 按原子性选择，而不是按 SQL 条数选择

是否开启显式事务取决于业务原子性和并发控制，不取决于“最终只有一条写 SQL”。以下场景需要事务或等价的原子方案：

- 多个数据库操作必须全部成功或全部失败
- 先读取旧状态，再根据旧状态决定写入，并且需要锁住该状态
- 同时维护余额、库存、幂等记录等跨行不变量
- 需要固定锁顺序，避免并发请求互相覆盖

Service 通过 `s.db.RunInTx` 开启事务闭包，并把同一个 `tx` 传给事务内所有 Repo：

```go
func (s *Order) CreateOrder(ctx context.Context, req service.CreateOrderRequest) error {
    return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
        orderRepo := repo.NewOrderRepo(tx)
        inventoryRepo := repo.NewInventoryRepo(tx)

        // Domain 构造函数执行初始校验
        order, err := domain.NewOrder(req.UserID, req.Items)
        if err != nil {
            return err
        }
        // Domain 方法执行自身逻辑（如计算总价）
        order.CalculateTotal()

        if err := orderRepo.Insert(ctx, order); err != nil {
            return err
        }
        return inventoryRepo.Deduct(ctx, req.Items)
    })
}
```

事务闭包内不得混用 `s.db` 和 `tx`，否则部分操作会逃逸到事务外。Domain 只操作内存字段，不感知事务。

### 9.2 显式事务外的安全写入

满足以下任一条件时，可以不显式开启事务：

- 单条 SQL 本身已经完整表达业务条件和写入
- 读改写通过版本号乐观锁完成，更新使用 `WHERE id = ? AND version = ?`
- 操作天然幂等，且不依赖另一次查询结果

下面的读取与更新虽然是两条 SQL，但 Repo 的 `Update` 带 version 条件，因此并发修改不会静默覆盖：

```go
func (s *User) ActivateUser(ctx context.Context, id uuid.UUID) error {
    userRepo := repo.NewUserRepo(s.db)
    user, err := userRepo.FindByID(ctx, id)
    if err != nil {
        return err
    }
    if err := user.Activate(); err != nil {
        return err
    }
    user.UpdatedAt = time.Now().UTC()
    if err := userRepo.Update(ctx, user); errors.Is(err, repo.ErrConflict) {
        return apperr.New(
            apperr.Conflict,
            apperr.CodeConcurrentUpdate,
            "resource was modified",
            err,
        )
    } else {
        return err
    }
}
```

冲突后只有在用例可安全重放时才允许有限重试；不得对扣款、发券、发送消息等副作用盲目自动重试。

### 9.3 悲观锁与锁顺序

当业务必须基于最新值完成一组修改，且冲突概率较高时，在事务内使用 `SELECT ... FOR UPDATE`。同时锁定多行时按稳定键排序后获取锁，降低死锁概率。数据库返回序列化失败或死锁时，只对明确可重放的事务做有上限、带抖动的重试。

### 9.4 事务与缓存

缓存失效只能发生在事务成功提交后；事务内删除缓存可能在回滚后造成不必要的失效，也不能证明数据库已经提交：

```go
func (s *User) TransferPoints(ctx context.Context, fromID, toID uuid.UUID, amount int) error {
    err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
        fromUser, err := repo.NewUserRepo(tx).FindByID(ctx, fromID)
        if err != nil {
            return err
        }
        toUser, err := repo.NewUserRepo(tx).FindByID(ctx, toID)
        if err != nil {
            return err
        }

        if err := fromUser.DeductPoints(amount); err != nil {
            return err
        }
        toUser.AddPoints(amount)

        if err := repo.NewUserRepo(tx).Update(ctx, fromUser); err != nil {
            return err
        }
        return repo.NewUserRepo(tx).Update(ctx, toUser)
    })
    if err != nil {
        return err
    }
    // 仅在提交成功后失效。缓存仍以 TTL 作为最终陈旧上限。
    s.cache.Del(fmt.Sprintf("user:id:%s", fromID))
    s.cache.Del(fmt.Sprintf("user:id:%s", toID))
    return nil
}
```

---

## 10. 缓存策略

### 10.1 默认关闭，按证据启用

Ristretto 是可选的进程内缓存，不是每个 Service 的默认依赖。只有同时满足以下条件才启用：

- 指标或压测证明数据库读取是实际瓶颈
- 数据允许在明确 TTL 内短暂陈旧
- 缓存未命中、写入被拒绝或条目被淘汰都不影响业务正确性
- 已定义容量 cost、TTL、失效键和观测指标

强一致数据、低频数据、写多读少数据默认不缓存。多实例应用不能把 Ristretto 当作共享缓存。

启用缓存的 Service 可以额外持有类型化 Ristretto 实例；未启用缓存的 Service 不保留空接口或空实现占位。

### 10.2 只缓存不可变快照

```
1. 查询缓存 → 命中则直接返回
2. 未命中 → 通过 Repo 查询数据库（返回 Domain）
3. 转换为不可变值快照并尝试写入缓存
4. 返回结果
```

不得缓存会在后续业务方法中被修改的 `*domain.User` 指针。使用值类型快照，并在每次读取时构造新的 Domain 值：

```go
type UserCacheValue struct {
    ID        uuid.UUID
    Name      string
    Email     string
    Status    string
    Level     int
    Version   int64
    UpdatedAt time.Time
}

func (s *User) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
    key := fmt.Sprintf("user:id:%s", id)
    if val, ok := s.cache.Get(key); ok {
        return val.ToDomain(), nil // 每次返回新的值，不共享可变指针
    }
    user, err := repo.NewUserRepo(s.db).FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    snapshot := NewUserCacheValue(user)
    _ = s.cache.SetWithTTL(key, snapshot, snapshot.Cost(), 30*time.Second)
    return user, nil
}
```

Ristretto 允许新条目写入被拒绝或丢弃；这只会产生下一次 miss，不能影响正确性。生产请求不为了确认可见性而调用 `Wait()`；测试需要立即读取刚写入的条目时可以调用。

### 10.3 写后失效与陈旧窗口

数据库成功写入或事务成功提交后，立即删除所有相关键，并用短 TTL 限制失效遗漏和并发回填造成的最长陈旧时间：

```go
func (s *User) UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) error {
    userRepo := repo.NewUserRepo(s.db)

    // 查出旧记录并保存旧邮箱，用于清理旧索引的缓存键
    old, err := userRepo.FindByID(ctx, id)
    if err != nil {
        return err
    }
    oldEmail := old.Email

    // 应用变更（通过 Domain 方法）
    if err := old.SetEmail(email); err != nil {
        return err
    }
    old.Name = name // 简单字段直接赋值（无业务规则时）
    old.UpdatedAt = time.Now().UTC()
    if err := userRepo.Update(ctx, old); err != nil {
        return err
    }

    // 删除缓存：新索引 + 旧索引
    s.cache.Del(fmt.Sprintf("user:id:%s", id))
    s.cache.Del(userEmailCacheKey(email))
    if oldEmail != email {
        s.cache.Del(userEmailCacheKey(oldEmail))
    }
    return nil
}
```

Cache-Aside 存在“读请求查到旧值 → 写请求提交并删除缓存 → 读请求把旧值重新写回”的竞争窗口。若业务不能接受 TTL 内陈旧，不应使用这种本地缓存方案。

### 10.4 缓存键与敏感信息

对于同时存在按 ID 查询和按邮箱查询等多索引场景，为每种查询维度设计独立的缓存键：

| 查询维度 | 缓存键格式 | 示例 |
|----------|-----------|------|
| 按 ID | `user:id:{id}` | `user:id:01912345-...` |
| 按邮箱 | `user:email:{normalized-hash}` | `user:email:sha256:...` |

缓存键包含命名空间、实体、查询维度和规范化后的值。邮箱、手机号等个人信息不以明文写入缓存键或指标标签。

### 10.5 一致性边界

| 场景 | 策略 |
|------|------|
| 缓存 miss / Set 被拒绝 / 条目淘汰 | 回源数据库，业务结果不受影响 |
| 单进程写入 | DB 成功后删除相关键，TTL 限制遗漏影响 |
| 多写入事务 | 提交成功后删除相关键；回滚时不删除 |
| 旧索引清理 | 写前保存旧索引，写后同时删除旧键与新键 |
| 多实例部署 | 各实例缓存互不感知；要求共享失效机制，或明确接受 TTL 内陈旧 |
| 强一致读取 | 绕过缓存，直接读取数据库 |

至少观测 hit/miss、Set rejection、eviction、回源耗时和估算陈旧窗口；没有这些指标时无法判断缓存是否值得保留。

---

## 11. 测试策略

### 11.1 Domain 层单元测试

充血模型的业务规则脱离数据库即可独立测试，领域错误可用 `errors.Is` 精确断言：

```go
func TestUser_Activate(t *testing.T) {
    // 正常激活
    u := &domain.User{Status: "inactive"}
    assert.NoError(t, u.Activate())
    assert.Equal(t, "active", u.Status)

    // 重复激活应报错
    assert.ErrorIs(t, u.Activate(), domain.ErrAlreadyActive)

    // 封禁用户不能激活
    banned := &domain.User{Status: "banned"}
    assert.ErrorIs(t, banned.Activate(), domain.ErrUserBanned)
}
```

无运行时基础设施，不需要 mock 或 testcontainers，属于纯逻辑测试。

### 11.2 Repo 层集成测试

Repo 层使用 `testcontainers` 进行集成测试，验证数据库操作与真实 PostgreSQL 行为一致，同时验证 `schema` ↔ `domain` 转换（字段一一对应、类型一致、写入后回读不丢失数据）。

### 11.3 Service 层测试

`service/v1` 中的具体实现与 Repo 一起使用 testcontainers 测试，不为测试强行创建 Repo 接口。重点验证：

- 事务提交、回滚、锁顺序和乐观锁冲突
- Domain / Repo 错误是否映射为正确的 `apperr.Kind` 与稳定 `code`
- 未预期错误是否被包装为 `Internal`，且不会暴露内部 cause
- 启用缓存时，miss、淘汰、Set 被拒绝和失效遗漏均不影响数据库事实

邮件、时钟、对象存储和第三方 API 等已声明窄端口的外部边界，可以使用 fake/stub 测试失败路径。

### 11.4 Handler 接口 mock 测试

Handler 单元测试使用 `httptest`，注入实现 `service` 接口的 mock/stub，不构造 `service/v1`、Repo 或数据库。简单接口可手写替身；接口方法较多时可以生成 mock，但生成物限于测试用途。替身必须实现完整接口，并用编译期断言检查；未配置的方法调用应使测试失败，避免静默返回成功。

```go
// internal/handler/v1/user_test.go（与 §6.1、§6.2 配套；省略 import）
type userServiceStub struct {
    t        *testing.T
    activate func(context.Context, uuid.UUID) error
}

var _ service.User = (*userServiceStub)(nil)

func (s *userServiceStub) GetProfile(context.Context, uuid.UUID) (*domain.User, error) {
    s.t.Fatal("unexpected GetProfile call")
    return nil, nil
}

func (s *userServiceStub) ActivateUser(ctx context.Context, id uuid.UUID) error {
    if s.activate == nil {
        s.t.Fatal("unexpected ActivateUser call")
        return nil
    }
    return s.activate(ctx, id)
}

func TestUserHandler_ActivateUser_NotFound(t *testing.T) {
    id := uuid.Must(uuid.NewV7())
    calls := 0
    svc := &userServiceStub{t: t, activate: func(ctx context.Context, got uuid.UUID) error {
        calls++
        require.Equal(t, id, got)
        return apperr.New(apperr.NotFound, apperr.CodeUserNotFound, "user not found", nil)
    }}
    h := NewUserHandler(svc)
    r := gin.New()
    r.POST("/users/:id/activate", h.ActivateUser)

    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/users/"+id.String()+"/activate", nil)
    r.ServeHTTP(w, req)

    require.Equal(t, 1, calls)
    require.Equal(t, http.StatusNotFound, w.Code)
    var body struct {
        Code string `json:"code"`
    }
    require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
    require.Equal(t, apperr.CodeUserNotFound, body.Code)
}
```

覆盖参数绑定与 DTO 转换、非法参数不调用 Service、请求上下文与参数传递、成功响应、应用错误的 HTTP 状态与稳定 code、未知错误返回 500 且不泄漏 cause。Middleware 可同样注入接口替身验证放行与中断路径。少量真实 Service + PostgreSQL 的 HTTP 集成测试保留用于验证组装和整条链路；mock 测试不能替代 §11.3 的事务与业务错误映射验证。完整测试要求见 [Go 测试规范](Go%20测试规范.md)。

---

## 12. 反模式清单

以下反模式在实践中反复出现，明确禁止：

| 反模式 | 说明 | 正确做法 |
|--------|------|----------|
| 贫血模型 | 业务规则散落在 Service 的 if-else 里，直接改字段 | 规则收归 Domain 方法，状态变更只能经方法执行 |
| schema 泄漏 | handler/service/domain 试图导入持久化模型 | schema 放在 `repo/internal/schema`，由编译器限制 |
| Handler 绑定 Service 实现 | 字段或构造参数使用 `*servicev1.User`，测试被迫连接数据库 | 依赖 `service.User` 接口，注入 mock/stub |
| Service 接口反向依赖实现 | `service` 导入 `service/v1` 或暴露 bun/Repo 类型 | 接口包只声明业务契约，由组合根选择实现 |
| Handler 写业务逻辑 | handler 里做业务判断或直接访问 repo | 委托 Service，handler 只做协议适配 |
| 模型方法查库 | domain 方法接收 repo/db 参数做查询 | 外部资源校验留在 Service 层 |
| repo 无谓入 DI | repo 成为长生命周期单例，或仅为 mock 创建一一对应接口 | Service 方法内按需创建；需要替换时再抽象 |
| Service 直接改 domain 字段 | `u.Status = "active"` | 调用 `u.Activate()` 执行不变量检查 |
| 状态字符串散落 | `"active"` / `"disabled"` 字面量散落各层 | domain 类型化常量 + 统一转换 |
| 教条式 getter/setter | 每个字段都封装方法 | 只有状态机/不变量字段需要封装（见 §3.2） |
| 按错误文本分支 | `switch err.Error()` 或比较驱动错误文本 | `errors.Is` / `errors.As` + `apperr.Kind/Code` |
| API 版本复制 Service | 每新增 `/v2` 就复制整个 `service/v2` | API 版本与实现版本独立，由组合根选择并复用实现 |
| 读改写无并发控制 | 先查再更新，但无锁也无 version 条件 | 事务行锁或乐观锁 |
| 缓存可变指针 | 多个请求共享 `*domain.Xxx` 并原地修改 | 缓存不可变值快照，每次命中返回副本 |

---

## 13. 方案要点总结

- **分层清晰**：Handler / Middleware → service 接口；app 组装 service/v1 实现 → Repo → Domain，infra 独立成层，每层职责单一、依赖单向；repo → schema 仅在 Repo 内部发生
- **接口与实现分离**：service 提供接口与用例类型，service/v1 提供实现；实现版本独立于 Handler/DTO 的 API 版本
- **横切独立**：Middleware 作为 Handler 的子包处理 HTTP 横切关注点，定义与挂载分离，需要业务能力时直接依赖 Service 接口
- **Domain 纯净**：领域层不携带 ORM 映射或基础设施依赖，业务规则附着在 Domain 上而非散落在 Service 中
- **错误契约化**：Domain/Repo 返回可识别错误，Service 映射为 `apperr.Kind/Code`，Handler 不比较错误文本
- **Handler 易测**：注入 Service 接口的 mock/stub，用 httptest 验证协议适配，无需启动数据库
- **Domain 易测**：领域逻辑无运行时基础设施，单元测试不需要数据库或 mock，错误用 `errors.Is` 断言
- **schema 私有**：bun 持久化映射收敛在 Repo 的 schema 子包中，转换只在 Repo 内部发生，持久化细节不泄漏到其他层
- **依赖极简**：组合根显式组装长生命周期组件；Repo、Domain、Middleware、schema 不创建无收益的 DI 绑定
- **Repo 无状态**：构造即用即弃，天然支持事务注入
- **两类数据访问**：关系型数据走具体 Repo（bun.IDB）；Redis/第三方设施通过按业务能力命名的窄端口访问
- **主键统一**：实体使用 UUID v7，默认由应用层生成；纯关联表可以使用联合主键
- **缓存可选**：默认关闭；启用时只缓存不可变快照，任何 miss、淘汰或拒绝都不影响正确性
- **并发安全**：事务按原子性选择；Repo 通过 `bun.IDB` 接收 `*bun.DB` / `bun.Tx`，读改写使用锁或 version 条件
- **转换集中**：领域与持久化分离，转换点集中在 Repo 内部，schema 只服务 Repo 一家
