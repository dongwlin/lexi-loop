// components/ui 公共出口：每个组件独立文件夹，新增组件后在此追加导出。
// 这是跨 Feature 复用的稳定出口（前端应用架构规范 §5.2），组件内部文件路径不对外承诺。
export { default as AlertDialog } from './alert-dialog/AlertDialog.vue'
export { default as AlertDialogAction } from './alert-dialog/AlertDialogAction.vue'
export { default as AlertDialogCancel } from './alert-dialog/AlertDialogCancel.vue'
export { default as Button } from './button/Button.vue'
export { default as Dialog } from './dialog/Dialog.vue'
export { default as DialogClose } from './dialog/DialogClose.vue'
export { default as DropdownMenu } from './dropdown-menu/DropdownMenu.vue'
export { default as DropdownMenuItem } from './dropdown-menu/DropdownMenuItem.vue'
export { default as DropdownMenuRadioItem } from './dropdown-menu/DropdownMenuRadioItem.vue'
export { default as Popover } from './popover/Popover.vue'
