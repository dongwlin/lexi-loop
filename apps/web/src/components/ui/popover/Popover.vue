<script setup lang="ts">
import {
  PopoverContent,
  PopoverPortal,
  PopoverRoot,
  PopoverTrigger,
} from 'reka-ui'

// 气泡卡容器（设计系统方案 §8.3 Dropdown / Popover 配方 + §9 Reka 状态接入）：
// 固定 Root + Trigger + Portal + Content 组合，Content 套用弹层配方（不透明
// Overlay、Forced Colors 下补容器 outline——Reka 有内联 outline: none，必须带
// ! 后缀）。触发器经 as-child 由 #trigger 插槽交给调用方，调用方负责其可聚焦
// 与可访问名称；Reka 的 Escape / 点击外部关闭与焦点恢复逻辑不在此覆盖。
interface PopoverProps {
  /** Content 对齐方向。 */
  align?: 'start' | 'center' | 'end'
  /** Content 与触发器的间距。 */
  sideOffset?: number
}

withDefaults(defineProps<PopoverProps>(), { align: 'end', sideOffset: 8 })
</script>

<template>
  <PopoverRoot>
    <PopoverTrigger as-child>
      <slot name="trigger" />
    </PopoverTrigger>

    <PopoverPortal>
      <PopoverContent
        :align="align"
        :side-offset="sideOffset"
        class="z-50 w-max max-w-[calc(100vw-2rem)] rounded-popover bg-overlay p-4 text-foreground shadow-overlay forced-colors:outline-1 forced-colors:outline-[CanvasText]! forced-colors:outline-solid!"
      >
        <slot />
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>
