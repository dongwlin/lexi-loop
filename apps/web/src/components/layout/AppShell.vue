<script setup lang="ts">
import { useThemeStore, type ThemePreference } from '@/stores/theme'

// 跨页面布局（前端应用架构规范 §4.4 components/layout）：顶部导航 + 主内容容器。
// 通过 slots 承载路由页面，不读取业务 Query 或业务 API。
const theme = useThemeStore()

const themeOptions: Array<{ value: ThemePreference; label: string }> = [
  { value: 'light', label: '浅色' },
  { value: 'dark', label: '深色' },
  { value: 'system', label: '跟随系统' },
]

function onThemeChange(event: Event): void {
  const target = event.target
  if (!(target instanceof HTMLSelectElement)) {
    return
  }
  const matched = themeOptions.find((option) => option.value === target.value)
  if (matched) {
    theme.setPreference(matched.value)
  }
}
</script>

<template>
  <div class="flex min-h-dvh flex-col">
    <header class="border-b border-border bg-surface">
      <div class="mx-auto flex h-14 w-full max-w-2xl items-center px-4">
        <RouterLink
          to="/"
          class="mr-6 rounded-item text-sm font-semibold text-foreground"
        >
          LexiLoop
        </RouterLink>
        <nav aria-label="主导航" class="flex items-center gap-1">
          <RouterLink
            to="/review"
            class="inline-flex min-h-11 items-center rounded-item px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
            active-class="bg-surface-active text-foreground"
          >
            复习
          </RouterLink>
          <RouterLink
            to="/words"
            class="inline-flex min-h-11 items-center rounded-item px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
            active-class="bg-surface-active text-foreground"
          >
            生词库
          </RouterLink>
          <RouterLink
            to="/import"
            class="inline-flex min-h-11 items-center rounded-item px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
            active-class="bg-surface-active text-foreground"
          >
            导入
          </RouterLink>
        </nav>
        <select
          :value="theme.preference"
          aria-label="界面主题"
          class="ml-auto min-h-9 rounded-field border border-field-border bg-field px-2 text-sm text-foreground shadow-field outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          @change="onThemeChange"
        >
          <option v-for="option in themeOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>
    </header>
    <main class="mx-auto w-full max-w-2xl flex-1 px-4 py-6">
      <slot />
    </main>
  </div>
</template>
