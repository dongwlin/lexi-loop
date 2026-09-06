<script setup lang="ts">
import { computed, type Component } from 'vue'
import { DropdownMenuRadioGroup } from 'reka-ui'
import { Monitor, Moon, Sun } from 'lucide-vue-next'
import { DropdownMenu, DropdownMenuRadioItem } from '@/components/ui'
import { useThemeStore, type ThemePreference } from '@/stores/theme'

// 头部主题切换（设计系统方案 §8.3 Dropdown 配方 + §9 Reka 状态接入）：触发器展示当前
// 偏好图标，菜单用 RadioGroup 表达三选一偏好，选中态由指示图标与语义角色
// （menuitemradio）共同表达，不只靠颜色。
const theme = useThemeStore()

const themeOptions: Array<{ value: ThemePreference; label: string; icon: Component }> = [
  { value: 'light', label: '浅色', icon: Sun },
  { value: 'dark', label: '深色', icon: Moon },
  { value: 'system', label: '跟随系统', icon: Monitor },
]

const triggerIcon = computed(
  () => themeOptions.find((option) => option.value === theme.preference)?.icon ?? Monitor,
)

function setPreference(value: unknown): void {
  const matched = themeOptions.find((option) => option.value === value)
  if (matched) {
    theme.setPreference(matched.value)
  }
}
</script>

<template>
  <DropdownMenu>
    <template #trigger>
      <button
        aria-label="界面主题"
        class="inline-flex size-11 items-center justify-center rounded-item bg-transparent text-foreground transition-colors duration-150 ease-out hover:bg-default data-[state=open]:bg-default outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        <component :is="triggerIcon" class="size-5 shrink-0" aria-hidden="true" />
      </button>
    </template>

    <DropdownMenuRadioGroup
      :model-value="theme.preference"
      @update:model-value="setPreference"
    >
      <DropdownMenuRadioItem
        v-for="option in themeOptions"
        :key="option.value"
        :value="option.value"
      >
        <component
          :is="option.icon"
          class="size-4 shrink-0 text-muted-foreground group-data-[state=checked]/item:text-primary-text"
          aria-hidden="true"
        />
        <span>{{ option.label }}</span>
      </DropdownMenuRadioItem>
    </DropdownMenuRadioGroup>
  </DropdownMenu>
</template>
