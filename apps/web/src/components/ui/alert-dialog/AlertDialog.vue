<script setup lang="ts">
import {
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
  AlertDialogTrigger,
} from 'reka-ui'

// 警示对话框容器（设计系统方案 §8.3 Dialog 配方 + 交互与可访问性规范 §7.2 / §9.4）：
// 用于不可逆或高风险操作的确认。与普通 Dialog 的差别由 Reka 语义保证——AlertDialog 不响应
// Escape 与点击遮罩关闭，用户必须显式选择；初始焦点落在内容首个可聚焦元素（把 Cancel 放在
// Action 之前即为安全操作）。open 为受控属性，开闭统一经 update:open 交给调用方裁决，调用方
// 可在提交中拦截关闭；标题与说明必填，操作按钮经插槽由 AlertDialogCancel / AlertDialogAction
// 组成（视觉交给 ui/button），危险按钮文案用具体动作名称（§9.4）。
interface AlertDialogProps {
  open: boolean
  title: string
  description: string
}

defineProps<AlertDialogProps>()
const emit = defineEmits<{ 'update:open': [open: boolean] }>()
</script>

<template>
  <AlertDialogRoot :open="open" @update:open="emit('update:open', $event)">
    <AlertDialogTrigger as-child>
      <slot name="trigger" />
    </AlertDialogTrigger>

    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-40 bg-black/50" />
      <AlertDialogContent
        class="fixed left-1/2 top-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 flex-col overflow-y-auto rounded-dialog bg-overlay p-6 text-foreground shadow-overlay forced-colors:outline-solid! forced-colors:outline-1 forced-colors:outline-[CanvasText]!"
      >
        <AlertDialogTitle class="text-base font-semibold leading-6">
          {{ title }}
        </AlertDialogTitle>
        <AlertDialogDescription class="mt-1.5 text-sm text-muted-foreground">
          {{ description }}
        </AlertDialogDescription>

        <div v-if="$slots.default" class="mt-4">
          <slot />
        </div>

        <div v-if="$slots.footer" class="mt-6 flex flex-wrap justify-end gap-3">
          <slot name="footer" />
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
