# @lexi-loop/web

LexiLoop（词环）的 Vue 3 + TypeScript + Vite 前端应用。产品与技术文档见 [docs/](../../docs/README.md)，前端工程规范见 [docs/specs/frontend/](../../docs/specs/frontend/)。

## 命令

```bash
pnpm dev        # 开发服务器（/api 代理到本地 Go 服务，默认 127.0.0.1:8080）
pnpm build      # 类型检查 + 生产构建
pnpm preview    # 预览生产构建
pnpm typecheck  # vue-tsc 类型检查
pnpm test       # vitest（watch）
pnpm test:run   # vitest（单次执行）
```

## API Client

API 客户端是 workspace 共享包 [`packages/api-client`](../../packages/api-client/)（`docs/architecture/overview.md` §5.2）：请求函数与 DTO 类型由 Orval 从 `docs/openapi/` 自动生成（包内 `pnpm generate`，**不手改生成物**）；统一传输（`credentials: 'omit'`、超时 / 取消 / 网络错误分类）、`{code, message, data}` 解包与 `ApiError` 分类由包内手写 mutator 层承担。Feature 层直接从 `@lexi-loop/api-client` 导入端点函数（`importWords` / `listWords` / `startReviewSession` 等）与 DTO 类型。

本应用在 `src/lib/api.ts` 装配（由 `main.ts` 引入）：注入 `VITE_API_BASE_URL`（`lib/env.ts` 校验，缺省同源走 Vite 代理）与认证注入点，并重导出 `ApiError` 等错误工具供页面取用。

MVP 服务端暂无认证端点；认证落地后在 `src/lib/api.ts` 注入 `getAccessToken`，刷新与 401 重放按《前端 API 与认证集成规范》§10 在包内 mutator 实现，对调用方透明。

## 结构

按《前端应用架构规范》§3 组织：`src/app/` 应用装配（根组件、显式路由表、providers / pinia / query client 一次性安装），`src/pages/` 路由页面（当前为骨架占位，页面结构见 [docs/frontend/review-flow.md](../../docs/frontend/review-flow.md) §1），`src/components/layout/` 跨页面布局（AppShell），`src/stores/` 跨页面客户端状态，`src/lib/` 基础设施适配（env / api / storage），`src/styles/` 设计系统主题层（Tailwind CSS v4 + Radix Colors，见《Vue 组件设计系统方案》）。业务能力出现时按 Feature 组织进 `src/features/`。

主题偏好为 Light / Dark / System（《Vue 组件设计系统方案》§11）：偏好持久化在 `localStorage`（`lexi-loop.theme`），解析后的主题类互斥挂在 `<html>`，`main.ts` 在应用挂载前初始化。

## 待办

清理脚手架（2026-09-06）时留下的后续工作，完成后删除对应条目：

- [ ] 品牌设计产出 favicon 后放入 `public/`，并在 `index.html` 补 `<link rel="icon">`（create-vite 模板 favicon 已删除）
- [ ] 引入剩余质量设施：ESLint + Prettier、MSW、Storybook、Playwright、组件测试环境（jsdom + Testing Library）等（见《前端技术栈》§10/§11；Vitest 与 Vue Router / Pinia / TanStack Query / Tailwind CSS v4 / Reka UI + Radix Colors 已随骨架引入）
