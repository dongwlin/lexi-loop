<script setup lang="ts">
import { AlertDialogCancel } from 'reka-ui'

// 警示对话框的取消按钮（交互与可访问性规范 §9.4：取消操作应容易找到且不导致数据变化）：
// 经 as-child 由插槽交给调用方的 ui/button；点击经 Root 的 update:open 关闭，由调用方裁决。
</script>

<template>
  <AlertDialogCancel as-child>
    <slot />
  </AlertDialogCancel>
</template>
