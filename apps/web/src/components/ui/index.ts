// components/ui 公共出口：每个组件独立文件夹，新增组件后在此追加导出。
// 这是跨 Feature 复用的稳定出口（前端应用架构规范 §5.2），组件内部文件路径不对外承诺。
export { default as Button } from './button/Button.vue'
