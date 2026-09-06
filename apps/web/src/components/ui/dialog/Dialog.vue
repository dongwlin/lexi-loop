<script setup lang="ts">
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  DialogTrigger,
} from 'reka-ui'
import { X } from 'lucide-vue-next'

// 对话框容器（设计系统方案 §8.3 Dialog 配方 + §9 Reka 状态接入）：固定 Root + Trigger +
// Portal + Overlay + Content 组合，Content 套用弹层配方（不透明 Overlay、Forced Colors 下补
// 容器 outline——Reka 有内联 outline: none，必须带 ! 后缀）。open 为受控属性，打开与关闭
// （触发器 / Escape / 点击遮罩 / 右上角关闭）统一经 update:open 交给调用方裁决，调用方可在
// 提交中拦截关闭（交互与可访问性规范 §7.2）；焦点圈闭与关闭后焦点恢复由 Reka 承担。
// Title / Description 语义必备，故都设为必填；正文与底部操作经插槽承载。
interface DialogProps {
  open: boolean
  title: string
  description: string
}

defineProps<DialogProps>()
const emit = defineEmits<{ 'update:open': [open: boolean] }>()

// Reka 打开时默认聚焦首个可聚焦元素（右上角关闭按钮）；内容里有 data-dialog-initial-focus
// 标记时优先聚焦它（交互与可访问性规范 §5.3：打开 Dialog 移入合适的首个元素）。未标记则
// 交给 Reka 默认行为。
function handleOpenAutoFocus(event: Event) {
  const initial = (event.currentTarget as HTMLElement | null)?.querySelector<HTMLElement>(
    '[data-dialog-initial-focus]',
  )
  if (initial) {
    event.preventDefault()
    initial.focus()
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogTrigger as-child>
      <slot name="trigger" />
    </DialogTrigger>

    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-40 bg-black/50" />
      <DialogContent
        @open-auto-focus="handleOpenAutoFocus"
        class="fixed left-1/2 top-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 flex-col overflow-y-auto rounded-dialog bg-overlay p-6 text-foreground shadow-overlay forced-colors:outline-solid! forced-colors:outline-1 forced-colors:outline-[CanvasText]!"
      >
        <div class="flex items-start justify-between gap-4">
          <DialogTitle class="text-base font-semibold leading-6">
            {{ title }}
          </DialogTitle>
          <DialogClose
            aria-label="关闭"
            class="-mr-2 -mt-1 flex size-9 shrink-0 items-center justify-center rounded-item text-muted-foreground transition-colors duration-150 ease-out hover:bg-default hover:text-foreground outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            <X class="size-4" aria-hidden="true" />
          </DialogClose>
        </div>
        <DialogDescription class="mt-1.5 text-sm text-muted-foreground">
          {{ description }}
        </DialogDescription>

        <div class="mt-4">
          <slot />
        </div>

        <div v-if="$slots.footer" class="mt-6 flex flex-wrap justify-end gap-3">
          <slot name="footer" />
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
