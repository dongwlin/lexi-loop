# @lexi-loop/web

LexiLoop（词环）的 Vue 3 + TypeScript + Vite 前端应用。产品与技术文档见 [docs/](../../docs/README.md)，前端工程规范见 [docs/specs/frontend/](../../docs/specs/frontend/)。

## 命令

```bash
pnpm dev        # 开发服务器
pnpm build      # 类型检查 + 生产构建
pnpm preview    # 预览生产构建
pnpm typecheck  # vue-tsc 类型检查
```

## 待办

清理脚手架（2026-09-06）时留下的后续工作，完成后删除对应条目：

- [ ] 设计系统落地时建立全局样式入口 `src/styles/main.css`（reset / 全局字体 / Design Token，见《前端应用架构规范》§4.8；脚手架模板的 `src/style.css` 已删除，当前应用无全局样式）
- [ ] 品牌设计产出 favicon 后放入 `public/`，并在 `index.html` 补 `<link rel="icon">`（create-vite 模板 favicon 已删除）
- [ ] 按需引入规范依赖并落地目录结构：Vue Router、Pinia、TanStack Query、Tailwind CSS v4、Reka UI + Radix Colors、ESLint、Vitest 等（见《前端技术栈》《前端应用架构规范》§3）
- [ ] `src/App.vue` 目前是最小占位根组件，待按《前端应用架构规范》§4.1 建立 `app/` 应用装配（根组件 / router / pinia / 插件安装）后替换
