# @lexi-loop/web

LexiLoop（词环）的 Vue 3 + TypeScript + Vite 前端应用。产品与技术文档见 [docs/](../../docs/README.md)，前端工程规范见 [docs/specs/frontend/](../../docs/specs/frontend/)。

## 命令

```bash
pnpm dev             # 开发服务器（/api 代理到本地 Go 服务，默认 127.0.0.1:8080）
pnpm build           # 类型检查 + 生产构建
pnpm preview         # 预览生产构建
pnpm typecheck       # vue-tsc 类型检查
pnpm lint            # ESLint Flat Config（前端技术栈 §11.1）
pnpm format          # Prettier 格式化（含 Tailwind class 排序）
pnpm test            # vitest（watch，全部 project）
pnpm test:run        # vitest 单次执行（web + storybook project）
pnpm test:coverage   # vitest + v8 覆盖率
pnpm test:storybook  # 仅 storybook project（Story 即真实浏览器测试）

pnpm test:e2e        # Playwright E2E（构建 + vite preview，本地 4173）
pnpm storybook        # Storybook 组件工作台（localhost:6006）
pnpm storybook:build  # Storybook 静态构建（产物 storybook-static/，不入库）
```

## API Client

API 客户端是 workspace 共享包 [`packages/api-client`](../../packages/api-client/)（`docs/architecture/overview.md` §5.2）：请求函数与 DTO 类型由 Orval 从 `docs/openapi/` 自动生成（包内 `pnpm generate`，**不手改生成物**）；统一传输（`credentials: 'omit'`、超时 / 取消 / 网络错误分类）、`{code, message, data}` 解包与 `ApiError` 分类由包内手写 mutator 层承担。Feature 层直接从 `@lexi-loop/api-client` 导入端点函数（`importWords` / `listWords` / `startReviewSession` 等）与 DTO 类型。

本应用在 `src/lib/api.ts` 装配（由 `main.ts` 引入）：注入 `VITE_API_BASE_URL`（`lib/env.ts` 校验，缺省同源走 Vite 代理）与认证注入点，并重导出 `ApiError` 等错误工具供页面取用。

MVP 服务端暂无认证端点；认证落地后在 `src/lib/api.ts` 注入 `getAccessToken`，刷新与 401 重放按《前端 API 与认证集成规范》§10 在包内 mutator 实现，对调用方透明。

## 结构

按《前端应用架构规范》§3 组织：`src/app/` 应用装配（根组件、显式路由表、providers / pinia / query client 一次性安装），`src/pages/` 路由页面（页面结构见 [docs/frontend/review-flow.md](../../docs/frontend/review-flow.md) §1），`src/components/ui/` 业务无关基础组件——每个组件独立文件夹（`button/Button.vue` + 同目录 Story），经 `components/ui/index.ts` 统一导出（Button 配方见《Vue 组件设计系统方案》§8.1），`src/components/layout/` 跨页面布局（AppShell、ThemeSwitcher），`src/stores/` 跨页面客户端状态，`src/lib/` 基础设施适配（env / api / storage），`src/styles/` 设计系统主题层（Tailwind CSS v4 + Radix Colors，见《Vue 组件设计系统方案》）。`src/features/` 按业务能力组织（当前 `words/`、`review/`、`meta/`），Query Key / Query / Mutation 置于各 Feature 的 `api/`（《前端应用架构规范》§8.1），DTO 直接使用 `@lexi-loop/api-client` 生成类型，queryFn 透传 `signal`；`src/utils/` 存放无状态纯函数（《前端应用架构规范》§4.5 / §4.8，当前 `parseImportText`、生词库的 `buildPageItems` / `parsePositiveInt` / `formatMeanings`、生词详情的 `meaningText` / `formatDateTime`、复习的 `review-snapshot` / `review-resume` / 复习状态机迁移 `review-state-machine` / 键盘映射 `review-keyboard` / 开始复习错误文案 `review-errors`、路由页面级导航判定 `isPageLevelNavigation`）。

主题偏好为 Light / Dark / System（《Vue 组件设计系统方案》§11）：偏好持久化在 `localStorage`（`lexi-loop.theme`），解析后的主题类互斥挂在 `<html>`，`main.ts` 在应用挂载前初始化。

测试设施在应用根目录外层：`vitest.config.ts`（Vitest Projects：`web` 为 jsdom 单元 / 组件 / 集成测试，`storybook` 为真实浏览器 Story 测试）、`tests/`（`setup.ts` 全局设施 + `mocks/` 的 MSW Handler 工厂与 server + `integration/` 页面集成测试——`helpers.ts` 提供真实 Router + 全新 QueryClient + MSW 的装配，覆盖生词库 / 详情 / 导入 / 复习结果跨页行为）、`e2e/`（Playwright 用例，webServer 面向生产构建 preview）。

## 业务与维护入口

阶段状态与后续规划见 [Roadmap](../../docs/product/roadmap.md)。六个业务页面、复习恢复、键盘操作和结果页行为统一见 [review-flow.md](../../docs/frontend/review-flow.md)，版本信息契约见 [Meta API](../../docs/api/meta.md)。页面实现位于 `src/pages/`，数据查询与变更位于 `src/features/`，跨页复习快照与状态逻辑位于 `src/utils/`。

页面集成测试位于 `tests/integration/`，使用真实 Router、独立 QueryClient 与 MSW；端到端用例位于 `e2e/`。新增行为按 [前端测试规范](../../docs/specs/frontend/前端测试规范.md) 补充对应层级验证。

## 质量与品牌

质量设施已就位（2026-09-07）：ESLint Flat Config（类型感知 TS / Vue / vuejs-accessibility / TanStack Query / import-x 规则）+ Prettier（含 Tailwind class 排序，配置与脚本见《前端技术栈》§11）；组件测试环境 jsdom + Testing Library（`vitest.config.ts` 按《前端测试规范》§15 合并 vite 配置，`tests/` 为跨页面集成与共享设施）；MSW 在 HTTP 边界 Mock（`tests/mocks/`，`onUnhandledRequest: 'error'`，测试指向保留假主机）；Storybook 项目接入 `@storybook/addon-vitest`（Story 在真实 Chromium 中执行）与 `addon-a11y`（axe 检查为测试门禁，全局 `test: 'error'`）；Playwright E2E（`playwright.config.ts`，webServer 跑生产构建 + preview，`e2e/` 存放用例）。Story 以菜单打开收尾时的 `aria-hidden-focus` 豁免等例外均以注释登记在对应 Story。

品牌 favicon 已从 `docs/brand/logo.png` 的两个实心轮廓直接描摹为 `public/favicon.svg`：保留原图的双色开环、内嵌 L、开口位置与浅色模式主色（`#242a34` / `#1771e4`），并裁去不适合标签页尺寸的外围留白；SVG 内通过 `prefers-color-scheme: dark` 跟随系统深色偏好，将深色半环提亮为 `#eeeeee`，品牌蓝保持不变。favicon 独立于页面内的主题切换；`index.html` 已声明 SVG favicon。
