<script setup lang="ts">
import {
  DropdownMenuContent,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuTrigger,
} from 'reka-ui'

// 下拉菜单容器（设计系统方案 §8.3 Dropdown / Popover 配方 + §9 Reka 状态接入）：固定
// Root + Trigger + Portal + Content 组合，Content 套用弹层配方（不透明 Overlay、Forced
// Colors 下补容器 outline——Reka 有内联 outline: none，必须带 ! 后缀）。触发器经
// as-child 由 #trigger 插槽交给调用方（ui/button 或图标按钮均可），调用方负责其可聚焦
// 与可访问名称；菜单项用同目录 DropdownMenuItem / DropdownMenuRadioItem，分组语义直接
// 组合 reka-ui 的 DropdownMenuRadioGroup。不覆盖 Reka 的键盘、焦点恢复或关闭逻辑。
interface DropdownMenuProps {
  /** Content 对齐方向，取配方与现网用法的 end。 */
  align?: 'start' | 'center' | 'end'
  /** Content 与触发器的间距。 */
  sideOffset?: number
}

withDefaults(defineProps<DropdownMenuProps>(), { align: 'end', sideOffset: 8 })
</script>

<template>
  <DropdownMenuRoot>
    <DropdownMenuTrigger as-child>
      <slot name="trigger" />
    </DropdownMenuTrigger>

    <DropdownMenuPortal>
      <DropdownMenuContent
        :align="align"
        :side-offset="sideOffset"
        class="z-50 max-h-(--reka-dropdown-menu-content-available-height) max-w-[calc(100vw-2rem)] min-w-44 overflow-y-auto rounded-popover bg-overlay p-1 text-foreground shadow-overlay forced-colors:outline-1 forced-colors:outline-[CanvasText]! forced-colors:outline-solid!"
      >
        <slot />
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>
