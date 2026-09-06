<script setup lang="ts">
import { DropdownMenuItem } from 'reka-ui'

// 菜单项（设计系统方案 §8.3 Menu Item 配方）：高亮读 Reka 的 data-highlighted，焦点环在
// outline-hidden 之后必须于 focus-visible 恢复 outline-solid，否则整个配方渲染不出焦点环。
// select 事件与 disabled 等属性未在本组件声明，经 attrs 透传给 Reka。
</script>

<template>
  <DropdownMenuItem
    class="flex min-h-11 cursor-default items-center gap-2.5 rounded-item px-3 py-2 text-sm outline-hidden select-none focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring focus-visible:outline-solid data-disabled:pointer-events-none data-disabled:opacity-50 data-highlighted:bg-default-hover"
  >
    <slot />
  </DropdownMenuItem>
</template>
