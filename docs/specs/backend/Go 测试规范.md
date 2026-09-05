# Go 测试规范

## 1. 概述

测试驱动开发（Test-Driven Development，TDD）是一种以测试驱动代码设计的开发方法，核心流程为 **红 → 绿 → 重构**：

1. **红**：先编写一个会失败的测试（此时被测代码尚不存在或未实现）
2. **绿**：用最小代价让测试通过（只求能跑，不追求优雅）
3. **重构**：在测试的保护下消除重复、改善设计，保证所有测试仍然通过

TDD 的三个收益：

- **可测性驱动设计**：写测试会暴露隐藏状态、过大的职责和不可控外部副作用，但不意味着每个类型都要创建接口
- **回归保护**：每一次重构都有测试兜底，改代码不再心惊胆战
- **文档即测试**：测试本身就是可执行的规格说明，比注释更可信

> [!note]
> TDD 不是"多写几个测试"，而是"先写测试、再写代码"。如果测试只是补在已有代码后面，得到的是覆盖，而不是驱动。

---

## 2. 测试金字塔与分层测试策略

按照 [[Go 单体应用架构规范]] 的分层架构（handler → service → repo → domain，middleware 为 handler 子包），测试策略应与分层对应：

| 层 | 测试类型 | 依赖 | 速度 |
| --- | --- | --- | --- |
| Domain | 单元测试 | 无运行时基础设施 | 极快 |
| Service + Repo | 集成测试 | testcontainers（真实 PostgreSQL） | 中等 |
| Handler | 组件测试 | httptest + 真实 Service/Test DB | 中等 |
| CLI | 组件测试 | Cobra command + 可控输入输出 | 快 |
| 离线导入 / Migration | 集成测试 | 小型 fixture + testcontainers | 中等 |
| 外部端口 | 单元测试 | fake/stub 邮件、时钟、对象存储、第三方 API | 快 |

测试金字塔原则：**底层测试多而快，顶层测试少而全**。

- Domain 层的业务规则不依赖运行时基础设施，纯逻辑测试成本很低
- Service 默认与具体 Repo 一起通过 testcontainers 验证编排、事务和错误映射，不为 mock 强行创建 Repo 接口
- Handler 的绑定、状态码和响应契约使用 `httptest` 验证；纯响应辅助函数可以单独做单元测试
- Cobra command 测试只验证命令路由、参数绑定、退出码与输出，不在 CLI 测试中重复业务规则
- 离线导入使用小型固定 CSV 与真实 PostgreSQL 验证幂等重跑、批次回滚和字段映射；迁移至少验证空库可完整 up
- 只有真实外部边界使用窄接口和测试替身

---

## 3. Go 测试基础

### 3.1 命名与约定

- 测试文件必须以 `_test.go` 结尾，与被测代码放在同一目录
- 测试函数签名必须是 `func TestXxx(t *testing.T)`
- 测试失败用 `t.Errorf` / `t.Fatalf`（或 testify 断言）报告，不要在测试里 `panic`
- 单元测试不依赖执行顺序，每个测试独立可运行

```go
package domain

import "testing"

func TestUser_Activate(t *testing.T) {
    // ...
}
```

### 3.2 常用命令

| 命令 | 作用 |
| --- | --- |
| `go test ./...` | 运行所有测试 |
| `go test -run TestUser_Activate ./internal/domain/` | 运行指定测试 |
| `go test -v ./...` | 显示每个测试的通过/失败详情 |
| `go test -cover ./...` | 输出测试覆盖率 |
| `go test -race ./...` | 开启竞态检测（Service 层并发测试必开） |
| `go test -bench . -benchmem` | 运行基准测试 |

---

## 4. TDD 工作流：红-绿-重构

以一个 Domain 状态转换为例演示完整流程。数据库用例也遵循同样循环，但红灯阶段由 testcontainers 集成测试驱动。

### 4.1 红：先写失败的测试

```go
// internal/domain/user_test.go
package domain

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestUser_Deactivate(t *testing.T) {
    u := &User{Status: "active"}
    err := u.Deactivate() // 编译不过：方法还不存在 —— 这就是“红”
    assert.NoError(t, err)
    assert.Equal(t, "inactive", u.Status)
}
```

### 4.2 绿：最小实现让测试通过

```go
func (u *User) Deactivate() error {
    u.Status = "inactive"
    return nil
}
```

### 4.3 重构：在测试保护下完善

补齐真实行为，并继续以测试驱动的方式添加边界用例：

```go
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
```

每一步都先写测试、再补实现，测试始终领先于代码半步。

---

## 5. 表驱动测试

表驱动（Table-Driven Test）是 Go 社区最推崇的测试组织方式：**把测试用例抽象成表格，用同一段逻辑遍历执行**。

### 5.1 基本结构

```go
func TestIsValidEmail(t *testing.T) {
    tests := []struct {
        name  string // 用例名，用于定位失败
        input string
        want  bool
    }{
        {"普通邮箱", "user@example.com", true},
        {"缺少@符号", "userexample.com", false},
        {"空字符串", "", false},
        {"无域名", "user@", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := IsValidEmail(tt.input)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 5.2 表驱动的好处

- **加用例零成本**：新增一行表格即可覆盖新场景，不用复制粘贴测试函数
- **失败信息可读**：`t.Run` 子测试名直接出现在失败输出里，一眼定位是哪条用例
- **结构清晰**：输入与期望成对出现，相当于一份可执行的规格表

### 5.3 常见字段约定

| 字段 | 含义 |
| --- | --- |
| `name` | 用例名称（必须），用中文短语描述场景 |
| `input` / `args` | 被测函数的输入 |
| `want` / `wantErr` | 期望结果 / 是否期望报错 |
| `setup` | 前置准备（可选，构造 mock 等） |

> [!tip]
> 期望报错的用例用 `assert.ErrorIs(t, err, domain.ErrUserBanned)` 而非 `assert.Error`，可以精确校验错误类型，而不是只校验"有没有错"。

---

## 6. 子测试与并行

### 6.1 t.Run 子测试

`t.Run` 让每个用例拥有独立的失败上下文，也支持只运行某一子测试：

```bash
go test -run 'TestIsValidEmail/空字符串' ./...
```

### 6.2 并行执行

```go
for _, tt := range tests {
    tt := tt // Go 1.22 之前的版本需要显式捕获循环变量
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        // ...
    })
}
```

> [!warning]
> `t.Parallel()` 只适合**互不共享状态**的用例。涉及共享变量（如数据库、单例缓存）的测试不要并行。

---

## 7. 断言风格：testify

项目测试栈为 `testify + testcontainers`（见 [[Go 技术栈]]），统一使用 testify 断言：

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

| API | 用途 |
| --- | --- |
| `assert.Equal(t, want, got)` | 值相等 |
| `assert.NoError(t, err)` / `assert.Error(t, err)` | 错误断言 |
| `assert.ErrorIs(t, err, domain.ErrXxx)` | 精确错误类型（常用） |
| `assert.Empty(t, got)` | 空值断言 |
| `require.NoError(t, err)` | 失败立即 `Fatal`，停止当前测试 |

`assert` 与 `require` 的区别：**assert 失败继续跑后续断言，require 失败立即终止**。前置条件（如初始化失败）用 `require`，普通断言用 `assert`。

---

## 8. 测试替身

测试替身只用于已经因真实替换需求而存在的窄外部端口，不用于模拟 Repo 或数据库。`bun.IDB` 是数据库执行接口，不是业务 Repo 接口。

### 8.1 替身类型

| 类型 | 定义 | 适用场景 |
| --- | --- | --- |
| Stub（桩） | 返回预设数据的假实现 | 让被测代码走通正常分支 |
| Mock（模拟） | 校验调用参数与调用次数 | 验证交互行为 |
| Fake（假实现） | 内存中的简化真实实现 | 替代邮件发送器、时钟、对象存储等外部边界 |

### 8.2 手写 Fake

例如验证码发送是外部副作用，Service 只依赖发送能力：

```go
type MessageSender interface {
    SendVerificationCode(ctx context.Context, target, code string) error
}

type fakeMessageSender struct {
    target string
    code   string
    err    error
}

func (f *fakeMessageSender) SendVerificationCode(_ context.Context, target, code string) error {
    if f.err != nil {
        return f.err
    }
    f.target = target
    f.code = code
    return nil
}
```

### 8.3 testify/mock

交互较复杂、确实需要校验调用次数时才使用 `testify/mock`。简单端口优先手写 fake，失败信息更直观：

```go
type MockMessageSender struct {
    mock.Mock
}

func (m *MockMessageSender) SendVerificationCode(ctx context.Context, target, code string) error {
    args := m.Called(ctx, target, code)
    return args.Error(0)
}
```

> [!note]
> Repo 和 Service 的关系型数据路径使用 testcontainers 跑真实 PostgreSQL，同时验证 SQL、事务、约束和 `schema` ↔ `domain` 转换。不要为了获得快速 mock 测试而改变生产架构。

---

## 9. 覆盖率与质量

### 9.1 覆盖率的使用姿势

- `go test -cover` 看整体覆盖率，`go test -coverprofile=coverage.out` 导出后用 `go tool cover -html=coverage.out` 查看具体行覆盖
- **覆盖率不是目标而是信号**：100% 覆盖率不代表质量，关键分支没测到才是问题
- 优先保证 Domain 业务规则、Service 编排分支（事务回滚、缓存失效）的覆盖

### 9.2 测试质量自检

- 每个测试是否独立？删除任意一个测试，其余应该全部仍可通过
- 断言是否精确？能用 `ErrorIs` 就不要用 `Error`，能用 `Equal` 就不要只验证不为空
- 是否测了失败路径？只测正常路径的测试，覆盖率再高也是假保险

---

## 10. 基准测试

性能敏感的代码（如缓存策略、热点 SQL）用基准测试量化：

```go
func BenchmarkUserService_Deactivate(b *testing.B) {
    s := newBenchmarkUserService(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = s.Deactivate(context.Background(), benchmarkUserID)
    }
}
```

运行：

```bash
go test -bench BenchmarkUserService_Deactivate -benchmem ./internal/service/
```

> [!tip]
> 基准测试前用 `b.ResetTimer()` 排除初始化开销；对比两个实现时保证 `-benchmem` 输出内存分配数据，内存分配往往比耗时更敏感。

---

## 11. 规范约定

结合本项目分层架构，测试规范如下：

- **文件与函数命名**：`xxx_test.go`、`TestXxx`、表驱动用例名用中文场景短语
- **Domain 测试**：无运行时基础设施的纯逻辑，是覆盖率的主要来源
- **Service + Repo 测试**：testcontainers + 真实 PostgreSQL，覆盖提交、回滚、乐观锁冲突和错误 code 映射，不 mock 数据库
- **Handler 测试**：`httptest` 验证参数绑定、HTTP 状态、统一响应结构和 Middleware 中断路径
- **CLI 测试**：直接执行 Cobra command，验证子命令选择、参数绑定、退出码与错误输出
- **Importer / Migration 测试**：fixture + testcontainers，验证 ECDICT 幂等重跑、批次失败回滚，以及空库完整迁移
- **外部端口测试**：邮件、时钟、对象存储、第三方 API 等使用手写 fake/stub；复杂交互才用 mock
- **缓存测试**：使用真实内存缓存，覆盖 miss、淘汰、Set 被拒绝和 TTL；不得假设 Set 后立即可见
- **断言**：统一 testify；前置条件用 `require`，普通断言用 `assert`；错误断言用 `ErrorIs`
- **CI 必跑**：`go test -race ./...` + `go vet ./...`
- **修改公共行为先改测试**：需求变更时，先更新测试（红），再改实现（绿），最后重构
