<script setup lang="ts">
import { AlertDialogAction } from 'reka-ui'

// 警示对话框的确认按钮：经 as-child 由插槽交给调用方的 ui/button（危险动作用 danger 变体，
// 文案用具体动作名称，交互与可访问性规范 §9.4）。点击默认经 Root 发出关闭请求——调用方需
// 在提交中拦截 update:open，并在异步成功后再关闭；业务侧的确认处理经 attrs 透传。
</script>

<template>
  <AlertDialogAction as-child>
    <slot />
  </AlertDialogAction>
</template>
