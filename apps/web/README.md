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

按《前端应用架构规范》§3 组织：`src/app/` 应用装配（根组件、显式路由表、providers / pinia / query client 一次性安装），`src/pages/` 路由页面（当前为骨架占位，页面结构见 [docs/frontend/review-flow.md](../../docs/frontend/review-flow.md) §1），`src/components/ui/` 业务无关基础组件（Button，配方见《Vue 组件设计系统方案》§8.1），`src/components/layout/` 跨页面布局（AppShell），`src/stores/` 跨页面客户端状态，`src/lib/` 基础设施适配（env / api / storage），`src/styles/` 设计系统主题层（Tailwind CSS v4 + Radix Colors，见《Vue 组件设计系统方案》）。业务能力出现时按 Feature 组织进 `src/features/`。

主题偏好为 Light / Dark / System（《Vue 组件设计系统方案》§11）：偏好持久化在 `localStorage`（`lexi-loop.theme`），解析后的主题类互斥挂在 `<html>`，`main.ts` 在应用挂载前初始化。

## 待办

MVP 完成度评估（2026-09-06）：工程骨架、设计系统主题层、API Client（Orval 生成 + 统一 mutator）与后端 8 个端点已就绪；`src/pages/` 四个路由页面仍为占位卡片，MVP 业务未落地。以下按依赖顺序登记，完成后删除对应条目。

### MVP 业务落地（review-flow.md §2–§10）

- [ ] Feature 层：`features/words/`、`features/review/` 的 api keys / queries / mutations（《前端应用架构规范》§8.1；DTO 直接使用 `@lexi-loop/api-client` 生成类型，queryFn 透传 `signal`）
- [ ] 导入页 `/import`（review-flow §2–§3）：多行粘贴解析（trim / lowercase / 去空行，重复单词聚合成 count 不丢弃）、`importWords` 提交、导入结果反馈（聚合口径，见下方契约缺口）
- [ ] 生词库 `/words`（review-flow §4）：搜索与分页写入 URL（《前端应用架构规范》§6.2）、语义表格（单词 / 释义 / 遇到 / 复习 / 记得 / 忘记）、Loading / Empty / Error 状态
- [ ] 生词详情 `/words/:id`（review-flow §5）：学习统计展示、编辑复习释义（Dialog，PATCH `customReviewMeaning`，null 清除并回退三层取值 D007）、删除生词（AlertDialog 确认，软删除语义 D004，重新导入可恢复）
- [ ] 复习页 `/review`（review-flow §6–§8、§10）：数量选择（10 / 20 / 30 / 50 + 自定义，超出词库时提示截断 D009）、active session 恢复（「继续复习 / 放弃本轮」由用户选择，D010；恢复进度用 GET session 逐词结果按 word join 本地快照）、状态机 idle → recalling → revealed → answered、键盘操作（Space 揭示释义，1 / ← 不记得，2 / → 记得）与焦点管理、完成跳转结果页。备注：无 abandon 端点，「放弃本轮」MVP 只清除本地快照，服务端旧 session 在下一轮开始时自动 abandon
- [ ] 复习结果页 `/review/result/:session`（review-flow §9）：汇总（总计 / 记得 / 不记得 / 正确率）、「需要加强」列表、再来一轮 / 回到生词库
- [ ] 页面基础设施：路由 meta → `document.title` 随路由更新、AppShell skip link 与 `main` landmark、各页面 h1 与 Loading / Empty / Error 分支（《前端交互与可访问性规范》§6.1）
- [ ] 纯逻辑单元测试：导入解析聚合、复习状态机迁移、session 快照存取、键盘映射（组件 / 页面集成测试待下方质量设施落地后补）

### 依赖服务端契约扩展（先行登记，前端不自行复算公式）

- [ ] 列表 / 详情 DTO 返回掌握程度与复习优先级（PRD §11 统计、review-flow §4 表格列；权重 / mastery 公式权威在 review/algorithm.md，由服务端动态计算，见 D011）——契约扩展前生词库与详情不展示这两项
- [ ] 导入接口返回逐词结果（review-flow §3 的「ambiguous 新增 / constrain 已存在 +2」反馈）——契约扩展前导入反馈只用聚合统计（created / updated / encounters）

### 质量与品牌

- [ ] 引入剩余质量设施：ESLint + Prettier、MSW、Storybook、Playwright、组件测试环境（jsdom + Testing Library）等（见《前端技术栈》§10/§11；Vitest 与 Vue Router / Pinia / TanStack Query / Tailwind CSS v4 / Reka UI + Radix Colors 已随骨架引入）
- [ ] 品牌设计产出 favicon 后放入 `public/`，并在 `index.html` 补 `<link rel="icon">`（create-vite 模板 favicon 已删除）
