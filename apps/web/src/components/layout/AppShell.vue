<script setup lang="ts">
import ThemeSwitcher from './ThemeSwitcher.vue'

// 跨页面布局（前端应用架构规范 §4.4 components/layout）：顶部导航 + 主内容容器。
// 通过 slots 承载路由页面，不读取业务 Query 或业务 API。
</script>

<template>
  <div class="flex min-h-dvh flex-col">
    <!-- 《前端交互与可访问性规范》§6.1：Skip Link——视觉隐藏，Tab 聚焦时显示，直达主内容。 -->
    <a
      href="#main-content"
      class="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:rounded-item focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-primary-foreground focus:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      跳到主内容
    </a>
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
        <div class="ml-auto">
          <ThemeSwitcher />
        </div>
      </div>
    </header>
    <!-- §6.1：唯一主内容 Landmark，skip link 目标。tabindex=-1 使跳转后焦点真正落入主内容。 -->
    <main
      id="main-content"
      tabindex="-1"
      class="mx-auto w-full max-w-2xl flex-1 px-4 py-6"
    >
      <slot />
    </main>
  </div>
</template>
