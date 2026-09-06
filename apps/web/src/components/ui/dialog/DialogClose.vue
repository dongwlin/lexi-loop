<script setup lang="ts">
import { DialogClose } from 'reka-ui'

// 对话框关闭按钮的语义包装（交互与可访问性规范 §7.2）：经 as-child 由插槽交给调用方的
// ui/button，关闭请求统一走 Root 的 update:open，由调用方裁决是否允许关闭。
</script>

<template>
  <DialogClose as-child>
    <slot />
  </DialogClose>
</template>
