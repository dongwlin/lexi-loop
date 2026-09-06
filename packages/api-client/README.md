# @lexi-loop/api-client

LexiLoop 的 TypeScript API 客户端 workspace 共享包（`docs/architecture/overview.md` §5.2），供 `apps/web` 使用。基于原生 `fetch`。

## 结构

- `src/generated/` — Orval 生成物：`client.ts` 请求函数 + `model/` DTO 类型。**不手工修改**，通过重新生成覆盖
- `src/mutator.ts` — 手写统一层：Orval 自定义 mutator，全部生成请求经由它发出——Base URL 前缀、Bearer 注入、`{code, message, data}` 统一解包、`ApiError` 转换
- `src/transport.ts` — fetch 薄封装：超时 / 取消 / 网络错误分类、`credentials: 'omit'`、`X-Request-Id` / `Retry-After` 解析
- `src/errors.ts` — `ApiError` 错误模型（kind 六分类 + 稳定业务 `code`，程序分支只判断 `code`）
- `spec/openapi.json` — 派生的 Orval 输入 spec（可随时重建，勿手改）

## 重新生成

后端契约变更后（`docs/openapi/` 由 `lexi-loop openapi` 重新生成）：

```bash
pnpm -F @lexi-loop/api-client generate
```

流程：`scripts/unwrap-envelope.mjs` 从 `docs/openapi/openapi.json` 派生 `spec/openapi.json`——把每个操作 200 响应的 `EnvelopeXxx` 外壳剥掉、直接引用 `data` 的 schema（运行时解包由 mutator 统一负责，生成类型才与 mutator 返回值一致），并移除错误响应与 `Error*` 组件（错误模型由 `errors.ts` 承载）。之后 Orval 以 fetch client + mutator 生成。CI 以「重新生成后 `git diff` 为空」校验生成物与后端契约同步。

## 使用

包不读取宿主应用的环境变量；Base URL 与 access token 由应用启动时注入一次（`apps/web` 在 `src/lib/api.ts` 装配）：

```ts
import { configureApiClient, listWords } from '@lexi-loop/api-client'

configureApiClient({ baseUrl, getAccessToken })

const page = await listWords({ page: 1, search: 'amb' }) // Promise<ListWordsResponse>，已是解包后的 data
```

认证刷新与 401 重放（《前端 API 与认证集成规范》§10）在 mutator 内实现，对调用方透明。

## 命令

```bash
pnpm generate   # 派生 spec + Orval 生成
pnpm typecheck  # tsc --noEmit
pnpm test:run   # vitest（mutator 与生成端点的单测）
```
