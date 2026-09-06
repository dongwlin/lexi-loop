<script setup lang="ts">
import { DropdownMenuItemIndicator, DropdownMenuRadioItem, type AcceptableValue } from 'reka-ui'
import { Check } from 'lucide-vue-next'

// 单选菜单项（Reka RadioGroup 三选一等偏好场景）：选中态由指示图标与语义角色
// （menuitemradio）共同表达，不只靠颜色。本组件提供 group/item 命名组，插槽内图标可用
// group-data-[state=checked]/item: 响应选中态。
interface DropdownMenuRadioItemProps {
  /** 该项对应的选中值，透传给 Reka RadioItem。 */
  value: AcceptableValue
}

defineProps<DropdownMenuRadioItemProps>()
</script>

<template>
  <DropdownMenuRadioItem
    :value="value"
    class="group/item flex min-h-11 cursor-default select-none items-center gap-2.5 rounded-item px-3 py-2 text-sm outline-hidden data-highlighted:bg-default-hover focus-visible:outline-solid focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring data-disabled:pointer-events-none data-disabled:opacity-50"
  >
    <slot />
    <DropdownMenuItemIndicator class="ml-auto inline-flex">
      <Check class="size-4 text-primary-text" aria-hidden="true" />
    </DropdownMenuItemIndicator>
  </DropdownMenuRadioItem>
</template>
