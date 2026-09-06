# 前端 API 与认证集成规范

## 1. 适用范围

本规范适用于基于 Vite、TypeScript 的 Vue 单页应用（SPA），规定前端调用 HTTP API、管理长短双 token、恢复会话、协调刷新和处理认证错误的统一方式。

本规范只规定前端集成与前后端协作边界：

- HTTP Client、API 生成器和数据层的技术选型以《[[前端技术栈]]》为准
- HTTP 顶层响应、错误码、状态码、分页和字段命名以《[[HTTP API 设计规范]]》为唯一权威
- 页面、Feature、`lib` 和状态归属以《[[前端应用架构规范]]》为准
- 表单错误、Toast、Focus、会话失效提示和认证表单体验以《[[前端交互与可访问性规范]]》为准
- 测试工具、测试层级和 HTTP Mock 以《[[前端测试规范]]》为准
- ETag 生成与服务端重验证语义以《[[ETag 设计规范]]》为准

项目使用 Vue，不得改变本规范规定的传输、token 轮换、并发刷新和错误处理语义。

---

## 2. 核心决策

### 2.1 使用显式 Bearer Token，不依赖 Cookie 会话

认证统一采用短期 access token 与长期 refresh token：

- access token 仅保存在当前运行时内存中，通过 `Authorization: Bearer <token>` 调用业务 API
- refresh token 由客户端存储适配器持久化，仅发送给刷新、退出和撤销等认证端点
- refresh token 不使用 Cookie，不依赖前端与 API 同站，也不依赖浏览器自动附带凭据
- API 请求默认使用 `credentials: 'omit'`；只有某个独立集成明确需要 Cookie 时，才能为该端点单独覆盖
- access token 和 refresh token 都不得出现在 URL、Query String、路由状态、日志、埋点或错误上报中

前端与 API 可以属于不同站点，例如：

```text
Frontend: https://app.example.com
API:      https://api.example.net
```

跨站只改变 CORS 配置，不改变 token 传递方式。

### 2.2 每次刷新都轮换 refresh token

刷新成功后，后端必须同时签发新的 access token 与 refresh token，并立即使旧 refresh token 失效。前端必须先可靠保存新 refresh token，再把刷新视为成功。

同一个 token family 中的 refresh token 具有单调递增的 `tokenVersion`。前端不得用较旧响应覆盖较新 token；后端保留 token family 关系，用于撤销会话和识别旧 token 重放。

### 2.3 认证刷新只有一个协调入口

登录恢复、主动刷新、access token 临近过期以及业务请求返回认证 `401`，都必须进入同一个 Refresh Coordinator。不得在 API 拦截器、Provider、路由守卫和不同 Feature 中分别实现刷新逻辑。

### 2.4 写请求允许在认证刷新后重放一次

业务请求因 access token 无效而收到认证 `401` 时，刷新成功后允许原请求重放一次，包括 `POST`、`PUT`、`PATCH` 和 `DELETE`。

这一规则成立的后端前提是：

> 认证中间件必须在进入业务 Handler、开启业务事务或产生任何业务副作用前返回认证 `401`。

网络中断、超时、连接重置和未知 `5xx` 不能证明业务 Handler 未执行，因此写请求遇到这些错误时不得自动重放。

### 2.5 服务端数据仍由 TanStack Query 管理

Session Manager 只管理：

- access token
- refresh token 生命周期
- 会话恢复状态
- 刷新、退出和跨标签页协调

当前用户 `/me`、权限集合和其他服务端数据仍由 TanStack Query 管理，不复制进 Pinia。

---

## 3. 分层与目录职责

推荐目录：

```text
src/
├── app/
│   ├── providers.ts
│   ├── query-client.ts
│   └── router.ts
├── features/
│   └── auth/
│       ├── api/
│       ├── components/
│       ├── queries.ts
│       └── use-auth-session.ts    # Vue: useAuthSession.ts
└── lib/
    ├── api.ts                     # 组合 packages/api-client：注入 Base URL 与 access token
    ├── auth/
    │   ├── refresh-coordinator.ts
    │   ├── session-manager.ts
    │   ├── token-storage.ts
    │   └── token-types.ts
    └── env.ts
```

契约 API Client 位于 monorepo 共享包 `packages/api-client`（见 `docs/architecture/overview.md` §5.2 与《[[前端技术栈]]》§7）；`lib/api` 只做应用侧组合。项目较小时可以合并文件，但职责不能混合。

| 模块 | 职责 | 不负责 |
| --- | --- | --- |
| `packages/api-client` transport | 发送请求、解析 Header 和响应体、支持 Abort | token 刷新、业务错误展示 |
| `packages/api-client` client | 附加 access token、统一解包、认证 401 后重放 | 登录表单、路由跳转、Toast |
| `packages/api-client` errors | 统一错误类型和协议错误转换 | 决定具体页面文案 |
| `lib/api`（应用侧组合） | 注入 Base URL 与 access token，导出端点实例 | 登录表单、路由跳转、Toast |
| `lib/auth/token-storage.ts` | 持久化 refresh token 与元数据 | 调用认证 API、保存 `/me` |
| `lib/auth/refresh-coordinator.ts` | 单飞刷新、跨标签页互斥和结果通知 | 业务权限判断 |
| `lib/auth/session-manager.ts` | 会话状态、登录结果提交、恢复和退出 | 保存当前用户 DTO |
| `features/auth` | 登录 UI、`/me` Query、权限视图和认证业务交互 | 自行实现底层 token 轮换 |
| `pages` | 编排页面状态和导航 | 直接读取 token、调用 `fetch` |

`lib/api` 与 `lib/auth` 不得 import `features`、`pages` 或 `app`。框架 Provider、Router 和 Query Client 在 `app/` 组合这些底层能力。

---

## 4. Token 数据模型与存储

### 4.1 Token Pair

登录和刷新成功后，认证 API 的 `data` 应提供完整 token pair 及服务端时间信息：

```ts
interface TokenPair {
  accessToken: string
  accessTokenExpiresAt: string
  refreshToken: string
  refreshTokenExpiresAt: string
  sessionId: string
  tokenVersion: number
}
```

- 时间使用带时区的 RFC 3339 字符串
- `sessionId` 标识当前登录会话或设备会话
- `tokenVersion` 在同一 `sessionId` 内单调递增
- 前端使用服务端返回的过期时间，不通过解析 PASETO/JWT 内容推断刷新计划
- 轮换不得无限延长初次登录建立的绝对会话期限

响应仍须遵循《[[HTTP API 设计规范]]》的统一顶层结构，本节只定义认证端点 `data` 的集成要求。

### 4.2 内存中的 access token

access token：

- 只保存在 Session Manager 的私有内存中
- 不进入 Pinia、TanStack Query Cache 或组件 state
- 不持久化到 `localStorage`、`sessionStorage` 或 IndexedDB
- 只通过窄接口 `getAccessToken()` 提供给 API Client
- 页面刷新、进程重启或 WebView 重建后通过 refresh token 恢复

不要把 token 暴露为 Vue 响应式 state。UI 只需要 `sessionStatus`，不需要读取凭据。

### 4.3 持久化 refresh token

所有平台必须实现同一个异步存储端口：

```ts
interface RefreshTokenStorage {
  read(): Promise<StoredRefreshToken | null>
  saveLogin(token: StoredRefreshToken): Promise<void>
  beginRotation(
    expected: Pick<StoredRefreshToken, 'sessionId' | 'tokenVersion'>,
  ): Promise<{ status: 'started'; requestId: string } | { status: 'stale' }>
  commitRotation(
    expected: Pick<StoredRefreshToken, 'sessionId' | 'tokenVersion'>,
    requestId: string,
    next: StoredRefreshToken,
  ): Promise<'committed' | 'stale'>
  clear(sessionId?: string): Promise<void>
}
```

```ts
interface StoredRefreshToken {
  refreshToken: string
  refreshTokenExpiresAt: string
  sessionId: string
  tokenVersion: number
  pendingRotation?: {
    requestId: string
    startedAt: string
  }
}
```

默认适配规则：

| 平台 | 默认存储 |
| --- | --- |
| 普通 Web / PWA | IndexedDB |
| 原生或混合移动应用 | iOS Keychain / Android Keystore 对应的安全存储插件 |
| 自动化测试 | 进程内 Fake Storage |

`localStorage` 只能作为经过项目风险评估后的兼容性降级方案，不能成为通用默认值。使用时仍须封装在同一接口后，禁止业务代码直接读写。

`beginRotation()` 必须在一个存储事务中比较当前 `sessionId` 与 `tokenVersion`：当前 token 没有待完成轮换时，生成并持久化新的请求 ID；已经存在待完成轮换时，返回原请求 ID。这样页面重载、移动网络中断或另一个标签页接管后，仍会重试同一次逻辑轮换。

`commitRotation()` 必须在一个存储事务中同时核对当前 `sessionId`、`tokenVersion` 和待完成的 `requestId`，全部匹配时才能写入 `next` 并清除待完成记录。返回 `stale` 表示另一个调用者已提交更新，当前响应及其中的 access token 都必须丢弃，不能只比较新旧数字后强行覆盖。

浏览器持久化存储仍可能被用户清除、隐私模式隔离或被系统回收。读取不到 refresh token 时按匿名会话处理，不能把“永不丢失”当作浏览器能力保证。PWA 可以按需申请持久存储权限，但不能依赖申请一定成功。

默认同一个前端 Origin 只维护一个活动会话。某标签页登录新账号时，`saveLogin()` 替换旧会话，其他标签页必须同步清除旧账号的内存 access token 和私有 Query Cache。若产品确实允许同一 Origin 并行登录多个账号，必须按 `sessionId` 为存储、锁、广播频道和 Query Key 建立完整命名空间，不能只给组件增加一个“当前账号”字段。

### 4.4 存储写入失败

登录或刷新得到新 token pair 后，如果新 refresh token 无法写入持久化存储：

1. 不提交新的 access token
2. 清除内存中的旧 access token
3. 尽力调用撤销端点废弃刚签发的会话
4. 抛出可恢复的本地存储错误
5. 提示用户检查浏览器隐私模式、存储权限或设备空间

不得继续运行在“access token 已更新、refresh token 仍是旧值”的半提交状态。

### 4.5 JavaScript 可访问凭据的安全边界

客户端显式保存 refresh token 后，XSS 成为主要凭据泄漏风险。项目至少必须：

- 使用严格 CSP，避免 `unsafe-inline` 和 `unsafe-eval`
- 对用户内容做上下文正确的转义或清洗
- 不使用 `innerHTML` / `v-html` 渲染不可信内容
- 限制第三方脚本和依赖来源，审查高权限 SDK
- 禁止 token 进入日志、Pinia Devtools、Query Devtools、埋点、Sentry Breadcrumb 和 Storybook args
- 让 refresh token 只在 Token Storage 与 Refresh Coordinator 的私有作用域中出现

客户端加密不能替代上述措施：如果解密密钥与密文都能被同一段前端 JavaScript 使用，单纯“加密后存储”不能解决 XSS 窃取问题。

---

## 5. API Client 契约

### 5.1 请求配置

统一 API Client 至少支持：

```ts
type AuthMode = 'required' | 'optional' | 'none'

interface ApiRequestOptions {
  auth?: AuthMode
  signal?: AbortSignal
  timeoutMs?: number
  headers?: HeadersInit
  idempotencyKey?: string
}
```

- 业务 API 默认为 `auth: 'required'`
- 登录、注册和刷新使用 `auth: 'none'`
- 公共内容接口可以使用 `auth: 'optional'`
- URL 必须从 `lib/env.ts` 中经过校验的 Base URL 构造
- API Client 默认发送 `Accept: application/json`
- 有 JSON Body 时发送 `Content-Type: application/json`
- 所有请求默认 `credentials: 'omit'`

`auth: 'required'` 且内存中没有 access token 时，不发送一个注定失败的业务请求；先进入会话恢复，恢复失败后抛出认证错误。

### 5.2 响应解包

普通 JSON 响应统一按《[[HTTP API 设计规范]]》解包：

```text
HTTP Response
  ↓
读取 X-Request-Id / Retry-After / ETag
  ↓
解析 JSON 顶层结构
  ↓
code = OK → 返回 data
  ↓
code ≠ OK → 抛出 ApiError
```

程序分支使用稳定的 `code`；`message` 只作为经过后端安全处理的默认展示文本。前端不得解析 `message` 判断登录失效、字段错误或重试策略。

以下响应单独处理：

- `204 No Content`：仅在具体端点契约允许时返回 `undefined`
- `304 Not Modified`：交给 ETag/缓存适配器处理，不按普通 JSON 响应解包
- 非 JSON 或顶层结构不合法：转换为 `ProtocolError`，不得伪装成业务错误

### 5.3 统一错误类型

```ts
type ApiErrorKind =
  | 'http'
  | 'network'
  | 'timeout'
  | 'abort'
  | 'protocol'
  | 'storage'

interface ApiError {
  kind: ApiErrorKind
  httpStatus?: number
  code?: string
  message: string
  data?: unknown
  fieldErrors?: Record<string, string[]>
  requestId?: string
  retryAfterSeconds?: number
  cause?: unknown
}
```

- `requestId` 读取响应头 `X-Request-Id`
- `retryAfterSeconds` 由 `Retry-After` 解析
- Abort 是预期控制流，默认不展示错误、不记录为故障
- Timeout 与 Network Error 必须区分，不能都转换成“服务器错误”
- 未知服务端字段错误显示为表单级错误，不能因字段名不匹配而丢失

### 5.4 错误展示归属

API Client 只负责分类，不直接弹 Toast 或导航：

| 错误 | 默认处理位置 |
| --- | --- |
| 字段校验错误 | Feature Form 对应字段 |
| 可预期业务冲突 | 当前 Feature 或 Page |
| `403` | Page 的 Forbidden 状态 |
| `404` | Page 的 Not Found 状态 |
| `429` | 当前操作附近，并告知可重试时间 |
| 网络或超时 | 当前 Query/Mutation 的可恢复错误状态 |
| 未知 `5xx` | Page 错误状态或应用级兜底 |
| Abort | 不展示 |

禁止为每一个失败请求弹全局 Toast。后台刷新失败、用户已离开页面导致的 Abort 和表单字段错误尤其不能产生全局噪声。

---

## 6. 取消、超时与普通重试

### 6.1 请求取消

- Query Function 必须把 TanStack Query 提供的 `signal` 传给 API Client
- Page 卸载、Query 失效或用户主动取消时，中止不再需要的读取请求
- 退出登录时取消全部私有 Query 和正在等待刷新重放的请求
- 已经进入后端事务的 Mutation 不能靠前端 Abort 推断操作已取消

### 6.2 超时

不设置过短的全局统一超时。读取接口可以按产品体验设置超时；上传、导出和移动网络请求应使用各自策略。

Mutation 超时只表示前端没有及时收到结果，不表示后端没有执行。界面应提示“结果未知”，并通过查询资源状态或幂等结果接口恢复，不能直接再次提交。

### 6.3 普通重试只在 TanStack Query 配置

API Client 不实现网络和 `5xx` 的通用重试，避免与 TanStack Query 叠加。

默认策略：

- Query 只对网络错误和明确可恢复的 `5xx` 做少量退避重试
- `400`、`401`、`403`、`404`、`409` 和 `422` 不做普通重试
- `429` 遵循 `Retry-After`，没有有效值时不自动高频重试
- Mutation 默认不做普通自动重试
- 只有后端明确支持 Idempotency Key 的写端点，Mutation 才能复用同一个 key 重试

认证 `401` 后的一次刷新与重放属于认证恢复，不计入 TanStack Query 的普通 retry 次数。

---

## 7. Session 状态模型

### 7.1 状态定义

```ts
type SessionStatus =
  | 'restoring'
  | 'authenticated'
  | 'anonymous'
  | 'recoverable-error'
```

| 状态 | 含义 |
| --- | --- |
| `restoring` | 正在读取 refresh token 或恢复 access token，受保护路由等待 |
| `authenticated` | 内存中存在可用 access token |
| `anonymous` | 没有 refresh token，或后端明确判定会话无效 |
| `recoverable-error` | 存在本地会话，但因网络、服务或存储故障暂时无法恢复 |

网络错误和 `5xx` 不得把状态直接改为 `anonymous`。只有以下情况才结束会话：

- refresh token 不存在或已过期
- 后端明确返回 refresh token 无效、已撤销或检测到重放
- 用户主动退出
- token 数据损坏且无法安全恢复

### 7.2 应用启动

```text
应用启动
  ↓
Session = restoring
  ↓
读取 refresh token
  ├── 不存在 → anonymous
  └── 存在
        ↓
      Refresh Coordinator
        ├── 成功 → 提交 token pair → authenticated → 获取 /me
        ├── 明确认证失败 → 清除 token → anonymous
        └── 网络/5xx/存储错误 → recoverable-error
```

路由守卫必须等待 `restoring` 结束。`recoverable-error` 应展示重试或离线反馈，不能重定向到登录页形成误退出。

### 7.3 登录

登录成功后的提交顺序：

1. 校验完整 token pair 和过期时间
2. 清理旧会话后，通过 `saveLogin()` 持久化新 refresh token
3. 持久化成功后，把 access token 放入内存
4. 更新 Session 为 `authenticated`
5. 获取或失效 `/me` Query
6. 广播当前会话已登录
7. 跳转到经过校验的返回地址

不能先跳转到受保护页面，再异步保存 refresh token。

### 7.4 主动刷新

前端应同时使用两类刷新触发器：

- 按 `accessTokenExpiresAt` 在过期前留出时间窗口主动刷新
- 业务 API 返回认证 `401` 时被动刷新兜底

主动刷新应考虑页面不可见、设备休眠、时钟偏差和离线状态。定时器只用于减少失败请求，不能代替 `401` 兜底。

页面重新获得焦点或网络恢复时，如果 access token 已过期或即将过期，可以进入同一个 Refresh Coordinator。不得另建一套刷新实现。

### 7.5 退出

退出流程：

1. 读取并暂存当前 refresh token，仅用于撤销请求
2. 立即清除内存 access token 和持久化 refresh token
3. Session 改为 `anonymous`
4. 取消进行中的私有 Query、Mutation 后续回调和等待重放请求
5. 清除私有 Query Cache
6. 向其他标签页广播退出事件
7. 尽力调用后端撤销端点
8. 导航到登录页或公开页面

本地退出不等待网络。撤销请求失败时不得把本地登录态恢复回来，但应留下可观测日志；高风险产品可以提供“退出所有设备”端点。

---

## 8. Refresh Token 轮换协议

### 8.1 刷新请求

刷新端点必须通过 HTTPS 调用，且不携带 access token：

```http
POST /api/v1/auth/refresh
Content-Type: application/json
X-Refresh-Request-Id: <uuid>
```

请求体携带当前 refresh token。Refresh Coordinator 在调用 API 前通过 `beginRotation()` 获得 `X-Refresh-Request-Id`；在因“尚未收到响应”而重试同一次刷新时保持不变，只有上一轮明确完成并提交新 token 后，下一次正常轮换才生成新 ID。

请求 ID 不得由 refresh token 哈希生成，也不能包含用户、设备或 token 内容。

网络错误、超时和无法证明轮换未发生的 `5xx` 必须保留待完成轮换记录。页面刷新或用户点击重试后，Coordinator 先读取该记录并复用请求 ID，不能把同一个旧 refresh token 配上新的请求 ID。

### 8.2 后端原子轮换前提

前端依赖后端满足以下语义：

1. 校验当前 refresh token、会话、绝对过期时间和 token family
2. 在同一事务中废弃旧 token、递增 `tokenVersion` 并记录新 token 哈希
3. 返回完整的新 token pair
4. 同一个旧 token 被不同 `X-Refresh-Request-Id` 再次使用时，按重放风险处理
5. 同一个请求 ID 因响应丢失被重试时，返回第一次轮换的同一逻辑结果，而不是再次轮换

第 5 条用于解决“服务端已轮换，但客户端在收到响应前断网”的不确定结果。后端可以通过受保护的幂等记录实现，但不得以长期明文保存 refresh token 为代价。幂等记录的可恢复期限必须作为认证契约明确配置；客户端待完成记录超过该期限后，不得用新的请求 ID 重放旧 token，而应结束本地会话并要求重新登录。

### 8.3 前端提交顺序

```text
收到新 token pair
  ↓
校验 sessionId / tokenVersion / expiresAt
  ↓
使用 expected version + requestId 原子提交新 refresh token
  ↓
更新内存 access token
  ↓
通知等待者
  ↓
重放原业务请求
```

任一步失败时不得释放一个“刷新成功”的结果。`commitRotation()` 返回 `stale` 时，当前 token pair 不得进入内存或广播；调用方应采用已经提交的较新会话结果。

### 8.4 Refresh 失败分类

| 失败 | Session 处理 | 原请求处理 |
| --- | --- | --- |
| refresh token 过期、撤销或无效 | 清除会话，转 `anonymous` | 抛出认证错误 |
| 检测到 refresh token 重放 | 清除整个 token family 对应会话 | 抛出安全认证错误 |
| 网络错误、超时、无法确认结果的 `5xx` | 保留 refresh token 和待完成请求 ID，转 `recoverable-error` | 返回可恢复错误，重试时复用请求 ID |
| `429` | 保留会话和请求 ID，遵循 `Retry-After` | 不立即循环刷新 |
| 新 token 存储失败 | 清除本地会话并尽力撤销新会话 | 抛出 Storage Error |

Refresh Coordinator 不得在失败后无上限递归调用自己。

---

## 9. 并发刷新协调

### 9.1 单个标签页：Single Flight

同一 JavaScript 运行时内只允许一个进行中的刷新 Promise：

```ts
let refreshPromise: Promise<AccessToken> | null = null
```

后续刷新调用复用该 Promise。Promise 完成后必须在 `finally` 中清空引用；等待者各自继续或失败，不得再次触发刷新。

### 9.2 多标签页：Web Locks + BroadcastChannel

Web 应用默认采用：

- Web Locks：同一前端 Origin 下的排他刷新锁
- BroadcastChannel：通知其他标签页登录、刷新和退出结果
- IndexedDB：保存 refresh token、`sessionId`、`tokenVersion` 和降级租约

推荐流程：

```text
Tab A / Tab B 同时需要刷新
          ↓
请求同名 Web Lock
          ↓
Tab A 获得锁，重新读取 IndexedDB 中的最新 token
          ↓
调用 refresh 并写入新 refresh token
          ↓
广播 REFRESH_SUCCEEDED
          ↓
Tab B 采用广播中的短期 access token，不再刷新
          ↓
Tab A 释放锁
```

锁名称必须包含应用标识，不能使用过于通用的全局名称：

```text
<app-name>:auth-refresh
```

持锁后必须重新读取 refresh token，不能继续使用排队前保存在闭包中的旧值。

### 9.3 广播消息

允许的消息：

```ts
type AuthChannelMessage =
  | {
      type: 'REFRESH_SUCCEEDED'
      sessionId: string
      tokenVersion: number
      accessToken: string
      accessTokenExpiresAt: string
    }
  | { type: 'SESSION_LOGGED_OUT'; sessionId: string }
  | {
      type: 'SESSION_LOGGED_IN'
      sessionId: string
      tokenVersion: number
      accessToken: string
      accessTokenExpiresAt: string
    }
```

- 广播可以携带短期 access token，以避免每个活动标签页依次轮换
- 广播不得携带 refresh token
- 消息只在同一受信任前端 Origin 内使用，不通过 `postMessage('*')` 转发
- 接收方必须校验 `sessionId`、`tokenVersion` 和过期时间
- 旧版本、其他会话和已经过期的消息必须忽略
- 消息对象不得写日志、埋点或持久化

默认单会话模式下，收到不同 `sessionId` 的 `SESSION_LOGGED_IN` 表示另一个标签页已经切换账号。接收方必须先取消旧请求、清除旧账号私有 Query Cache 和内存 token，再采用新会话；不能让新凭据继续显示旧账号数据。`SESSION_LOGGED_OUT` 只清除与消息 `sessionId` 匹配的当前会话，防止迟到消息退出一个更新的会话。

新打开或休眠期间错过广播的标签页，通过持久化 refresh token 正常恢复，不依赖历史广播。

### 9.4 Web Locks 不可用时

降级使用 IndexedDB 租约锁：

```ts
interface RefreshLease {
  ownerId: string
  expiresAt: number
}
```

- 在单个 IndexedDB read-write transaction 中检查并写入租约
- 获锁者使用随机 `ownerId`
- 网络调用期间按需续租
- 释放时只删除属于当前 `ownerId` 的租约
- 进程崩溃后，其他标签页等待租约过期并加入随机抖动再竞争
- 等待期间监听 BroadcastChannel，收到有效刷新结果即可停止竞争

不得把一个没有所有权校验的 `localStorage` 布尔值当作互斥锁。

### 9.5 原生与混合移动应用

原生或混合应用使用进程级 Mutex/Single Flight，并通过安全存储插件原子替换 refresh token。若应用可能同时运行多个进程或多个 WebView，必须提供与 Web Locks 等价的跨进程协调器，不能假设只有一个调用者。

---

## 10. 认证 401 与请求重放

### 10.1 识别认证 401

只有同时满足以下条件时才触发刷新：

- HTTP 状态为 `401`
- 响应业务 `code` 表示认证信息失效
- 原请求 `auth` 为 `required` 或携带过期 access token
- 原请求不是登录、刷新、退出或其他认证控制端点
- 原请求尚未因认证刷新重放过

不能把所有 `401`、`403` 或任意网络错误都当成 access token 过期。

### 10.2 重放次数

每个请求保存内部元数据：

```ts
interface AuthReplayState {
  authReplayCount: 0 | 1
}
```

流程：

```text
业务请求 → 认证 401
  ↓
Refresh Coordinator
  ├── 成功 → 使用新 access token 重建请求并重放一次
  └── 失败 → 返回对应错误

重放后再次 401
  ↓
不再刷新、不再重放，结束当前会话
```

### 10.3 写请求重放

写请求可以在认证 `401` 后重放一次，但必须满足：

- 第一次 `401` 由认证中间件在业务 Handler 前产生
- 请求体可重新构造；不能复用已经消费的 `Request` 或 Stream
- JSON、FormData 和上传请求分别提供可重放的 Request Factory
- 原请求的业务 Idempotency Key、条件请求头和 Trace Header 保持不变
- 只替换 access token，不重新生成业务请求 ID
- 用户退出或 AbortSignal 已取消时不再重放

认证重放与业务幂等是两个不同问题。认证 `401` 保证业务未执行；网络错误不具备这一保证。

### 10.4 刷新循环保护

- Refresh 请求使用不带认证恢复逻辑的底层 Transport
- 认证端点不能被普通 API Client 的 `401` 拦截器再次包裹
- 一个逻辑请求最多刷新一次、重放一次
- Session 已转为 `anonymous` 后，后续等待者直接失败
- 多个失败请求只触发一次退出和一次导航

---

## 11. 跨站 API 与 CORS

### 11.1 前端要求

- 所有生产 API 使用 HTTPS
- API Base URL 由 `lib/env.ts` 校验协议和合法 Origin
- 请求使用显式 Bearer Header 和 `credentials: 'omit'`
- 不通过 `mode: 'no-cors'` 绕过 CORS；不透明响应无法用于应用 API
- 不把不可信 URL 参数直接拼成 API Origin

### 11.2 后端 CORS 要求

私有 API 必须维护明确的前端 Origin Allowlist，即使请求不使用 Cookie，也不得对任意 Origin 开放敏感响应。

至少允许：

```text
Methods:  GET, POST, PUT, PATCH, DELETE, OPTIONS
Headers:  Authorization, Content-Type, X-Refresh-Request-Id,
          Idempotency-Key, If-None-Match, If-Match
Expose:   X-Request-Id, ETag, Retry-After
```

动态返回允许 Origin 时必须同时返回：

```http
Vary: Origin
```

前端不使用 Cookie 认证，因此不要求 `Access-Control-Allow-Credentials: true`。预检失败属于部署配置错误，不能在前端伪装成普通网络错误后无限重试。

### 11.3 CSRF 与 XSS

Bearer token 由前端显式读取并放入 Header 或刷新请求体，浏览器不会像 Cookie 一样自动附带，因此传统的跨站表单 CSRF 不是主要风险。

仍须防止：

- CORS Allowlist 配置过宽
- XSS 读取或滥用 token
- 恶意浏览器扩展和第三方脚本
- refresh token 被复制到日志或监控系统
- 前端从攻击者控制的 URL 构造状态变更请求

CORS 不是认证和授权机制；后端仍必须验证每个 access token 的签名、有效期、Audience、Subject、会话状态和权限。

---

## 12. TanStack Query 集成

### 12.1 Query Function

Feature Query Function 必须：

- 调用统一 API Client
- 透传 Query 的 `signal`
- 返回解包后的 `data`
- 让 `ApiError` 保留稳定类型
- 不自行读取 refresh token 或处理 `401`

### 12.2 Query Key 与用户隔离

私有 Query Cache 不能跨用户复用。项目至少采用一种方式：

- 退出时取消并删除全部私有 Query
- Query Key 包含稳定的会话或用户命名空间

默认优先在退出时精确清除私有 Query；支持快速切换多账号的产品再把 `sessionId` 或用户 ID 纳入 Query Key。

### 12.3 `/me`

`/me` 是普通私有 Query：

- Session 恢复为 `authenticated` 后才能启用
- 登录、权限变化后精确失效
- 退出时清除
- 不复制到 Auth Context 或 Pinia

Session 为 `authenticated` 只表示凭据可用；`/me` Query 决定当前用户资料的 Loading、Error 和 Success。

### 12.4 Mutation

- Mutation 的业务成功后精确更新或失效相关 Query
- 认证刷新与一次重放由 API Client 完成，对 Mutation 调用方透明
- 普通网络失败不由 Mutation 自动重试
- 乐观更新必须能在最终错误时回滚
- 用户退出后，旧会话 Mutation 的回调不得更新新会话页面

### 12.5 ETag

ETag 由 API 传输或缓存适配层处理，不散落在 Feature 中：

- `If-None-Match` 与已缓存 representation 绑定
- `304` 复用现有数据，不尝试解析空响应体
- 私有资源不得跨用户共享 ETag 与响应数据
- Query 的 `staleTime` 决定何时重新获取，ETag 只优化重验证传输

---

## 13. OpenAPI 与生成客户端

生成器的默认选择见《[[前端技术栈]]》；本节只规定所有生成器都必须遵循的集成边界。

- Swagger/OpenAPI 生成代码统一放在 monorepo 共享包 `packages/api-client`，应用内不得出现第二份生成物
- 生成文件不手工修改
- 页面和组件不得直接 import 生成客户端
- 生成客户端必须经过统一 Transport/API Client，确保 Header、响应解包、错误模型和认证刷新一致
- 登录与刷新响应应生成明确的 `TokenPair` 类型
- CI 应检查生成产物是否与后端契约同步
- TypeScript 类型不等于运行时校验；认证 token pair、环境变量和其他高风险边界仍需做必要的运行时校验

DTO 与视图类型一致时直接使用；只有字段组合、可空性、表单值或 UI 契约确实不同时才建立 Feature 适配。

---

## 14. 路由、权限与用户体验

### 14.1 受保护路由

路由守卫只判断 Session 状态：

- `restoring`：等待，不提前跳转
- `authenticated`：允许进入，再由页面 Query 获取数据
- `anonymous`：跳转登录页
- `recoverable-error`：展示恢复界面，不当作未登录

登录返回地址必须是经过校验的站内路径，禁止把任意外部 URL 当作重定向目标。

### 14.2 权限

- 前端权限判断用于隐藏不可用操作、减少无效请求和提供合理反馈
- 后端是认证与授权的唯一最终裁决者
- `401` 表示身份凭据失效，`403` 表示身份有效但权限不足
- `403` 不触发 token 刷新
- 权限 Query 更新后，相关页面和操作必须重新计算，不把权限复制到多个 Store

### 14.3 会话错误体验

必须区分：

- 登录凭据已失效，需要重新登录
- 当前离线或 API 暂时不可用，可以重试
- 本地存储不可用，需要调整浏览器或设备设置
- 当前账户无权限，需要返回或联系管理员

不得把上述情况全部显示为“登录已过期”。

---

## 15. 日志、监控与隐私

允许记录：

- `X-Request-Id`
- HTTP 状态码和稳定业务 `code`
- API 路径模板，不含敏感 Query
- 请求耗时、重试次数和刷新结果分类
- `sessionId` 的不可逆截断标识（确有排障需要时）

禁止记录：

- `Authorization` Header
- access token、refresh token 及其完整哈希
- 登录密码、验证码和恢复码
- 完整请求/响应体中的敏感字段
- BroadcastChannel 认证消息
- IndexedDB 认证记录内容

前端监控 SDK 必须配置 Header、Body、Breadcrumb 和 Storage Scrubber。开发环境调试也不能用 `console.log(token)`。

---

## 16. 测试要求

测试执行、MSW、Browser Mode 和 Storybook 规则见《[[前端测试规范]]》。本节只规定 API 与认证必须覆盖的行为。

### 16.1 API Client 单元与集成测试

- 成功响应正确解包 `data`
- 非 `OK` 响应转换为稳定 `ApiError`
- 捕获 `X-Request-Id`、`Retry-After` 和 ETag
- Network、Timeout、Abort 和 Protocol Error 正确分类
- Abort 不触发 Toast、监控故障或普通重试
- `304` 不解析空 JSON
- `auth: 'none'` 不附加 access token
- `credentials` 默认是 `omit`

### 16.2 Session 与轮换测试

- 启动时没有 refresh token → `anonymous`
- 启动刷新成功 → 先持久化 refresh token，再提交 access token
- refresh 明确认证失败 → 清除会话
- refresh 网络错误或结果不确定的 `5xx` → `recoverable-error`，保留 refresh token 与待完成请求 ID
- 存储写入失败 → 不保留半提交会话
- refresh 成功后旧 token 不再写回
- 较旧 `tokenVersion` 无法覆盖较新版本
- 同一逻辑刷新重试复用 `X-Refresh-Request-Id`
- 网络中断、页面重载和另一标签页接管后仍能读取待完成请求 ID
- 下一次正常刷新使用新的 Request ID
- 待完成请求超过后端可恢复期限后，不用新 ID 重放旧 token
- logout 即使撤销请求失败也完成本地清理

### 16.3 并发测试

- 同一标签页多个并发 `401` 只产生一次 refresh
- 等待者共享成功或失败结果
- 多标签页竞争时只有一个持锁者使用同一 refresh token
- 持锁后重新读取存储中的最新 token
- 刷新成功后先写存储，再广播和释放锁
- 其他标签页收到有效 access token 后不再次刷新
- 旧版本、错误 `sessionId` 和过期广播被忽略
- 跨标签页登录新账号会先清除旧账号私有 Query Cache
- 迟到的旧会话退出消息不会清除新会话
- 持锁标签页崩溃后，租约过期可恢复
- 跨标签页 logout 清除各自内存 access token

协调器纯逻辑使用 Vitest + Fake Storage/Fake Channel 测试；Web Locks、BroadcastChannel、IndexedDB 和真实浏览器生命周期至少保留一组 Vitest Browser Mode 集成测试。

### 16.4 请求重放测试

- 认证 `401` → refresh → 原请求恰好重放一次
- `POST`、`PUT`、`PATCH`、`DELETE` 的方法、Body 和业务 Idempotency Key 保持不变
- 重放使用新 access token
- 重放后再次 `401` 不再刷新
- Refresh 端点自身 `401` 不进入循环
- 写请求 Network Error、Timeout 和未知 `5xx` 不自动重放
- 用户退出或 Abort 后，等待请求不再重放

### 16.5 页面与 Storybook

页面集成测试使用真实 Router、Query Client、Session Manager 和 Store，只在 HTTP 边界使用 MSW，至少覆盖：

- 会话恢复期间等待
- 恢复成功后进入原目标页面
- 会话明确失效后进入登录页
- 网络故障进入可恢复状态而不是误退出
- `/me`、私有 Query 和退出清理协作
- `401` 与 `403` 的不同页面反馈

Storybook 只覆盖登录表单、会话失效提示、离线恢复界面等可见组件状态和交互；Single Flight、token 轮换和锁属于基础设施行为，不用 Story 代替 Vitest 测试。

---

## 17. 禁止事项

- 使用 Cookie 保存或自动发送 refresh token
- 把 access token 持久化到浏览器存储
- 在 Page、组件、Store 或 Query Function 中直接读取 refresh token
- 每个请求拦截器各自调用 refresh
- 让多个标签页并发消费同一个 refresh token
- 刷新成功前先释放锁、广播成功或重放请求
- 使用旧 `tokenVersion` 覆盖新 refresh token
- 把 refresh token 放入 BroadcastChannel、URL、日志或监控事件
- 用一个 `localStorage` 布尔值假装跨标签页互斥锁
- 把 Refresh 的网络失败当成退出登录
- 对 `403` 发起 token 刷新
- 对写请求的 Network Error 或 Timeout 自动重放
- 重用已经消费的 Request Body
- API Client 直接弹 Toast 或执行 Router 导航
- 同时在 API Client 与 TanStack Query 配置普通重试
- 把 `/me` 或权限数据复制进 Auth Store
- 让前端权限检查替代后端授权

---

## 18. 实现顺序

建议按以下顺序落地：

1. `ApiError`、底层 Transport 与响应解包
2. Token Storage 端口及 Web/移动端适配器
3. Session Manager 状态机
4. 单标签页 Single Flight Refresh Coordinator
5. 认证 `401` 后一次请求重放
6. Web Locks、BroadcastChannel 与 IndexedDB 租约降级
7. TanStack Query、Router 和 `/me` 集成
8. CORS、监控脱敏和 ETag 协作
9. Vitest、MSW 与 Browser Mode 风险矩阵

每完成一层都先通过其独立测试，再接入下一层。不要一开始把 Router、Query、存储、跨标签页和所有认证 UI 写进一个巨大 Provider 或 Store。

---

## 19. 速查清单

### API

- [ ] Page、组件和 Store 都只通过 Feature API 调用后端？
- [ ] API Client 统一处理 Base URL、Bearer Header、响应解包和 `ApiError`？
- [ ] 程序分支只判断稳定 `code`，没有解析 `message`？
- [ ] `X-Request-Id`、`Retry-After`、Abort 和 `304` 已正确处理？
- [ ] 普通重试只由 TanStack Query 管理？

### Token

- [ ] access token 只在内存中？
- [ ] refresh token 通过平台存储适配器持久化，不使用 Cookie？
- [ ] 每次刷新都轮换 refresh token 并递增 `tokenVersion`？
- [ ] 新 refresh token 持久化成功后才提交 access token？
- [ ] token 不会进入 URL、日志、监控、Devtools 或 Story？

### 并发与重放

- [ ] 单页内刷新使用 Single Flight？
- [ ] 多标签页使用 Web Locks + BroadcastChannel，并有 IndexedDB 租约降级？
- [ ] 持锁后会重新读取最新 refresh token？
- [ ] 写请求只在认证 `401` 后重放一次？
- [ ] Network Error、Timeout 和未知 `5xx` 不会自动重放写请求？
- [ ] Refresh 端点和重放请求不会形成循环？

### 会话与数据

- [ ] `restoring`、`anonymous` 与 `recoverable-error` 已明确区分？
- [ ] `/me` 和权限只由 TanStack Query 管理？
- [ ] 退出会立即清理本地凭据、取消请求并清除私有缓存？
- [ ] 新用户不会看见旧用户的 Query Cache？
- [ ] 前端守卫和权限展示没有替代后端授权？

### 部署与测试

- [ ] 生产 API 全部使用 HTTPS？
- [ ] CORS 使用明确 Origin Allowlist 并暴露必要响应头？
- [ ] API 请求默认 `credentials: 'omit'`？
- [ ] 并发刷新、token 轮换、写请求重放和失败恢复都有 Vitest 测试？
- [ ] Web Locks、BroadcastChannel 与 IndexedDB 有 Browser Mode 集成测试？

---

## 20. 参考

- [RFC 10017：OAuth 2.0 for Browser-Based Applications](https://www.rfc-editor.org/rfc/rfc10017.html)
- [RFC 9700：Best Current Practice for OAuth 2.0 Security](https://www.rfc-editor.org/rfc/rfc9700.html)
- [MDN：Web Locks API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Locks_API)
- [MDN：Broadcast Channel API](https://developer.mozilla.org/en-US/docs/Web/API/Broadcast_Channel_API)
- [MDN：IndexedDB API](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API)
- [MDN：Cross-Origin Resource Sharing](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS)
