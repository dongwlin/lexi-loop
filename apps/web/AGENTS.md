# AGENTS.md — lexi-loop web

`apps/web` 是 LexiLoop 的 Vue 3 + TypeScript + Vite 前端应用。本文件是 web 子树内 Agent 的补充工作指引；仓库级纪律（改动前读 agent-log、冲突裁决、改动后记录）以根目录 [AGENTS.md](../../AGENTS.md) 为准，前端目录结构与职责权威见 [docs/specs/frontend/前端应用架构规范.md](../../docs/specs/frontend/前端应用架构规范.md)。本文件不维护 schema、公式或决策结论，只引用。

## 当前状态：骨架就绪，导入页与生词库已落地

工程骨架（Vue 3 + vue-router + Pinia + TanStack Query + Tailwind CSS v4 + Reka UI / Radix Colors + Storybook 10）、设计系统主题层（`src/styles/`，Light / Dark / System）、API Client 装配（workspace 包 [`packages/api-client`](../../packages/api-client/)：Orval 生成请求函数与 DTO 类型 + 手写 mutator 统一传输、`{code, message, data}` 解包与 `ApiError` 分类；web 侧装配点为 `src/lib/api.ts`）、Feature 层（`src/features/words/`、`src/features/review/` 的 keys / queries / mutations）均已就绪；四个路由页面中导入页 `/import` 与生词库 `/words` 已完整落地，生词详情 / 复习 / 复习结果页仍为占位卡片。下一步清单（含服务端契约缺口登记）以 [README.md](README.md) 为准——完成里程碑后同步更新两处，避免状态失真。

改动本子树任何代码前：先读 [docs/agent-log/](../../docs/agent-log/) 当月文件的最近记录了解上下文，再核对下表对应文档与当前代码。

## 权威文档速查（改动对象 → 先读）

| 改动对象 | 先读 |
| --- | --- |
| 目录结构 / 职责边界 / 状态归属 | [docs/specs/frontend/前端应用架构规范.md](../../docs/specs/frontend/前端应用架构规范.md)（§2–§8、§11） |
| 依赖与工具链选型 / 测试栈 | [docs/specs/frontend/前端技术栈.md](../../docs/specs/frontend/前端技术栈.md) |
| API 调用 / 错误分类 / Token 与认证 | [docs/specs/frontend/前端 API 与认证集成规范.md](../../docs/specs/frontend/前端%20API%20与认证集成规范.md) |
| 交互行为 / 焦点键盘 / 弹层表单 | [docs/specs/frontend/前端交互与可访问性规范.md](../../docs/specs/frontend/前端交互与可访问性规范.md) |
| 设计令牌 / 主题 / 基础组件配方 | [docs/specs/frontend/Vue 组件设计系统方案.md](../../docs/specs/frontend/Vue%20组件设计系统方案.md) |
| Query 缓存 / 数据流性能 | [docs/specs/frontend/Vue 性能与缓存优化.md](../../docs/specs/frontend/Vue%20性能与缓存优化.md) |
| 测试方法与环境 | [docs/specs/frontend/前端测试规范.md](../../docs/specs/frontend/前端测试规范.md) |
| 页面信息架构与复习状态机 | [docs/frontend/review-flow.md](../../docs/frontend/review-flow.md) |
| HTTP 契约 | [docs/api/words.md](../../docs/api/words.md)、[docs/api/reviews.md](../../docs/api/reviews.md)（Orval 输入源头 `docs/openapi/` 由后端生成，不手改） |
| 字段语义 | [docs/dictionary/data-model.md](../../docs/dictionary/data-model.md)、[docs/review/data-model.md](../../docs/review/data-model.md) |

## 常用命令

```bash
cd apps/web
pnpm dev        # 开发服务器（未设 VITE_API_BASE_URL 时 /api 代理到本地 Go 服务 127.0.0.1:8080）
pnpm build      # vue-tsc -b + vite build
pnpm typecheck  # vue-tsc --noEmit
pnpm test:run   # vitest 单次执行（pnpm test 为 watch）
pnpm storybook  # 组件工作台 localhost:6006（storybook:build 产物 storybook-static/ 不入库）

pnpm -F @lexi-loop/api-client generate  # 后端契约变更（docs/openapi/ 更新）后重新生成 API client，生成物不手改
pnpm -F @lexi-loop/api-client test:run  # api-client mutator / 生成端点单测
```

纯逻辑单元测试已覆盖 `lib/env`、`lib/storage/local-storage`、`stores/theme`、`utils/parseImportText` 与两个 Feature 的 query keys；组件 / 页面级测试待质量设施（jsdom + Testing Library、MSW、Playwright 等，见 README 待办与《前端技术栈》§10/§11）落地后补——设施未引入不构成跳过纯逻辑单测的理由。

## 架构与实现要点

以前端应用架构规范为准，以下只记常见踩坑的速记指针：

- 跨目录 import 统一 `@/`（→ `src/`，`vite.config.ts` alias；架构规范 §5.2）；barrel file 只用于确实稳定的公共出口（§5.2）。
- 应用装配一次性收在 `src/app/`（显式路由表 `router.ts`、providers / pinia / query-client）；路由页面在 `src/pages/`，页面结构与状态机以 review-flow.md 为准（架构规范 §2.1、§3、§4.1）。
- 网络访问只经 `@lexi-loop/api-client`：包内 `src/generated/` 与派生 spec 不手改，后端契约变更后用 `pnpm -F @lexi-loop/api-client generate` 重建；web 侧装配点只有 `src/lib/api.ts`（`VITE_API_BASE_URL` 经 `lib/env.ts` 校验后注入），页面不得自行 fetch 或复制 DTO 类型（API 集成规范 §3、§5）。
- Query Key / Query / Mutation 放各 Feature 的 `src/features/<域>/api/`（架构规范 §8.1）；DTO 直接用生成类型，queryFn 透传 `signal`；服务端数据只经 TanStack Query 管理，不复制进 Pinia（API 集成规范 §2.5、架构规范 §7）。
- 无状态纯函数放 `src/utils/`（架构规范 §4.5 / §4.8），跨 Feature 客户端状态放 `src/stores/`（Pinia，§4.6），跨 Feature 带状态的 composable 放 `src/composables/`（§4.5）；状态归属决策顺序见 §7.1。
- 基础组件 `src/components/ui/` 每组件独立文件夹（`Button.vue` + 同目录 Story），经 `components/ui/index.ts` 统一导出；保持业务无关——不 import stores / api-client / 路由（架构规范 §4.4）；视觉配方以设计系统方案 §8 为准，不在组件里另起一套。
- 主题：偏好 Light / Dark / System 持久化在 `localStorage`（`lexi-loop.theme`），`main.ts` 在应用挂载前初始化，解析后的主题类互斥挂 `<html>`（设计系统方案 §11）。
- 可访问性基线随页面落地，不后补：页面 h1 与 landmark、Loading / Empty / Error 分支（架构规范 §11、交互与可访问性规范 §6.1）、焦点移动与快捷键（同规范 §5.3–§5.4）；复习页键盘映射 Space / 1 / 2 / ← / → 以 review-flow.md 为准。
- 可分享页面状态（搜索、分页、恢复进度）写 URL（架构规范 §6.2、交互与可访问性规范 §6.2）。

## MVP 边界与冻结决策（web 侧）

冻结决策索引见根 [AGENTS.md](../../AGENTS.md) 与 [docs/decisions/README.md](../../docs/decisions/README.md)。前端落地时的硬约束（细节以对应权威文档为准，这里不展开）：

- D011：权重 / mastery 由服务端动态计算，前端不复算公式；列表 / 详情 DTO 未返回掌握度前，生词库 / 详情不展示这两项（README 已登记契约缺口）。
- D009：复习数量截断 `min(count, available)` 由服务端保证；前端数量选择（10 / 20 / 30 / 50 + 自定义）超出词库时只做提示，不复算可用量。
- D010：全库至多一个 active session，恢复由用户选择（继续 / 放弃）；服务端无 abandon 端点，「放弃本轮」只清本地快照，旧 session 由下一轮开始时自动 abandon（api/reviews.md）。
- D007：释义展示取值 `custom ?? review ?? raw` 语义以 dictionary/data-model.md 为准；清除自定义释义（PATCH null）后回退三层取值。
- D004：删除生词为软删除语义（可重新导入恢复），界面不出现「物理删除」类表述。
- D012：MVP 只做随机复习，不做到期调度 UI。
- 认证未落地：服务端暂无认证端点；`getAccessToken` 注入点已预留于 `src/lib/api.ts`，刷新与 401 重放按《前端 API 与认证集成规范》§10 在 mutator 内实现，对调用方透明；此前不建认证占位页面或路由守卫。
- 不属于 MVP（V2 / V3）：认证、在线 Provider / Enrich、缓存细节、多用户——不为此创建占位文件。

## 变更时的文档同步

- 先改权威文档正文，再同步引用它的下游（本文件、README、decisions 索引、agent-log），避免同一决策长期存在两套说法（根 AGENTS.md 整理约定）。
- 完成产生仓库文件变化的工作单元后，由主 Agent 按 [docs/agent-log/README.md](../../docs/agent-log/README.md) 在当月文件记录一条，写明完成内容、涉及文件与验证方式。
