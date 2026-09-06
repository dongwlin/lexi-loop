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

## 待办

清理脚手架（2026-09-06）时留下的后续工作，完成后删除对应条目：

- [ ] 设计系统落地时建立全局样式入口 `src/styles/main.css`（reset / 全局字体 / Design Token，见《前端应用架构规范》§4.8；脚手架模板的 `src/style.css` 已删除，当前应用无全局样式）
- [ ] 品牌设计产出 favicon 后放入 `public/`，并在 `index.html` 补 `<link rel="icon">`（create-vite 模板 favicon 已删除）
- [ ] 按需引入规范依赖并落地目录结构：Vue Router、Pinia、TanStack Query、Tailwind CSS v4、Reka UI + Radix Colors、ESLint、MSW 等（见《前端技术栈》《前端应用架构规范》§3；Vitest 已随 API Client 引入）
- [ ] `src/App.vue` 目前是最小占位根组件，待按《前端应用架构规范》§4.1 建立 `app/` 应用装配（根组件 / router / pinia / 插件安装）后替换
