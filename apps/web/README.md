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

pnpm storybook        # Storybook 组件工作台（localhost:6006）
pnpm storybook:build  # Storybook 静态构建（产物 storybook-static/，不入库）
```

## API Client

API 客户端是 workspace 共享包 [`packages/api-client`](../../packages/api-client/)（`docs/architecture/overview.md` §5.2）：请求函数与 DTO 类型由 Orval 从 `docs/openapi/` 自动生成（包内 `pnpm generate`，**不手改生成物**）；统一传输（`credentials: 'omit'`、超时 / 取消 / 网络错误分类）、`{code, message, data}` 解包与 `ApiError` 分类由包内手写 mutator 层承担。Feature 层直接从 `@lexi-loop/api-client` 导入端点函数（`importWords` / `listWords` / `startReviewSession` 等）与 DTO 类型。

本应用在 `src/lib/api.ts` 装配（由 `main.ts` 引入）：注入 `VITE_API_BASE_URL`（`lib/env.ts` 校验，缺省同源走 Vite 代理）与认证注入点，并重导出 `ApiError` 等错误工具供页面取用。

MVP 服务端暂无认证端点；认证落地后在 `src/lib/api.ts` 注入 `getAccessToken`，刷新与 401 重放按《前端 API 与认证集成规范》§10 在包内 mutator 实现，对调用方透明。

## 结构

按《前端应用架构规范》§3 组织：`src/app/` 应用装配（根组件、显式路由表、providers / pinia / query client 一次性安装），`src/pages/` 路由页面（页面结构见 [docs/frontend/review-flow.md](../../docs/frontend/review-flow.md) §1），`src/components/ui/` 业务无关基础组件——每个组件独立文件夹（`button/Button.vue` + 同目录 Story），经 `components/ui/index.ts` 统一导出（Button 配方见《Vue 组件设计系统方案》§8.1），`src/components/layout/` 跨页面布局（AppShell、ThemeSwitcher），`src/stores/` 跨页面客户端状态，`src/lib/` 基础设施适配（env / api / storage），`src/styles/` 设计系统主题层（Tailwind CSS v4 + Radix Colors，见《Vue 组件设计系统方案》）。`src/features/` 按业务能力组织（当前 `words/`、`review/`），Query Key / Query / Mutation 置于各 Feature 的 `api/`（《前端应用架构规范》§8.1），DTO 直接使用 `@lexi-loop/api-client` 生成类型，queryFn 透传 `signal`；`src/utils/` 存放无状态纯函数（《前端应用架构规范》§4.5 / §4.8，当前 `parseImportText`、生词库的 `buildPageItems` / `parsePositiveInt` / `formatMeanings`、生词详情的 `meaningText` / `formatDateTime`、复习的 `review-snapshot` / `review-resume` / 复习状态机迁移 `review-state-machine` / 键盘映射 `review-keyboard`、路由页面级导航判定 `isPageLevelNavigation`）。

主题偏好为 Light / Dark / System（《Vue 组件设计系统方案》§11）：偏好持久化在 `localStorage`（`lexi-loop.theme`），解析后的主题类互斥挂在 `<html>`，`main.ts` 在应用挂载前初始化。

## 待办

MVP 完成度评估（2026-09-06）：工程骨架、设计系统主题层、API Client（Orval 生成 + 统一 mutator）、Feature 层（words / review 的 api keys / queries / mutations）与后端 8 个端点已就绪；`src/pages/` 五个路由页面（导入、生词库、生词详情、复习、复习结果）已全部落地，页面基础设施（路由 meta → document.title、skip link、main landmark、h1 与状态分支）已就绪。以下按依赖顺序登记，完成后删除对应条目。

### MVP 业务落地（review-flow.md §2–§10）

- [x] 导入页 `/import`（review-flow §2–§3）：多行粘贴解析（trim / lowercase / 去空行，重复单词聚合成 count 不丢弃）、`importWords` 提交、导入结果反馈（聚合口径，见下方契约缺口）
- [x] 生词库 `/words`（review-flow §4）：搜索与分页写入 URL（《前端应用架构规范》§6.2，默认值不写入）、语义表格（单词 / 释义 / 遇到 / 复习 / 记得 / 忘记，无掌握度 / 优先级列——D011 契约缺口）、Loading（Skeleton + `keepPreviousData` 翻页不闪）/ Empty（区分空词库与搜索无结果）/ Error（错误说明 + 重试）状态；纯逻辑（`buildPageItems` / `parsePositiveInt` / `formatMeanings`）在 `src/utils/` 配单测
- [x] 生词详情 `/words/:id`（review-flow §5）：学习统计（无掌握度 / 优先级——D011 契约缺口）、生效释义逐条展示（D007 三层取值，自定义来源标注「自定义」）、编辑复习释义（Dialog，`meaningText` 文本 ⇄ 结构化释义，清空保存 = PATCH null 回退词典层，另设「清除自定义释义」）、删除生词（AlertDialog 确认，软删除语义 D004，删除成功回生词库）；已删 / 不存在 id 呈 404 态；生词库行内单词链接到详情；对话框提交中拦截关闭、失败聚焦错误摘要（§5.3 / §7.2）
- [x] 复习页 `/review`（review-flow §6–§8、§10）：数量选择（10 / 20 / 30 / 50 + 自定义，自定义输入未确认直接开始时先应用输入；开始前以 GET /words 分页 total 提示可用量不足，D009 截断仍由服务端保证）、active session 恢复（「继续复习 / 放弃本轮」由用户选择，D010；恢复检查经 Feature 层查询 `fetchQuery` 强制取新，瞬时失败呈错误态可重试、仅 404 清快照；「继续复习」以 GET session 逐词结果按 word join 本地快照，跳到首个未答项续答）、状态机 idle → recalling → revealed → answered、键盘操作绑定复习区域（Space 揭示释义，1 / ← 不记得，2 / → 记得；焦点在按钮等控件上时 Space / Enter 保留原生激活）、完成跳转结果页。备注：无 abandon 端点，「放弃本轮」MVP 只清除本地快照，服务端旧 session 在下一轮开始时自动 abandon
- [x] 复习结果页 `/review/result/:session`（review-flow §9）：汇总（总计 / 记得 / 不记得 / 正确率）、「需要加强」列表、再来一轮 / 回到生词库；不存在 id 呈 404 态（对齐详情页），未完成 / 已放弃分支区分文案
- [x] 页面基础设施：路由 meta → `document.title` 随路由更新、SPA 页面级导航完成后焦点移到新页面主标题（h1 带 `tabindex="-1"`，兜底主内容；初次加载与搜索 / 分页等仅 query 变化不移动焦点，《前端交互与可访问性规范》§5.3）、AppShell skip link（目标 `main` 带 `tabindex="-1"` 保证跳转后焦点落入主内容）与 `main` landmark、各页面 h1 与 Loading / Empty / Error 分支（《前端交互与可访问性规范》§6.1）

### 依赖服务端契约扩展（先行登记，前端不自行复算公式）

- [ ] 列表 / 详情 DTO 返回掌握程度与复习优先级（PRD §11 统计、review-flow §4 表格列；权重 / mastery 公式权威在 review/algorithm.md，由服务端动态计算，见 D011）——契约扩展前生词库与详情不展示这两项
- [ ] 导入接口返回逐词结果（review-flow §3 的「ambiguous 新增 / constrain 已存在 +2」反馈）——契约扩展前导入反馈只用聚合统计（created / updated / encounters）

### 质量与品牌

- [ ] 引入剩余质量设施：ESLint + Prettier、MSW、Playwright、组件测试环境（jsdom + Testing Library）、`@storybook/addon-vitest` 与 `addon-a11y` 等（见《前端技术栈》§10/§11；Vitest 与 Vue Router / Pinia / TanStack Query / Tailwind CSS v4 / Reka UI + Radix Colors 已随骨架引入，Storybook 10 + `@storybook/vue3-vite` 与 Button Story 已落地）
- [ ] 品牌设计产出 favicon 后放入 `public/`，并在 `index.html` 补 `<link rel="icon">`（create-vite 模板 favicon 已删除）
