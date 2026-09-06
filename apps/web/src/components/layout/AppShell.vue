<script setup lang="ts">
// 跨页面布局（前端应用架构规范 §4.4 components/layout）：顶部导航 + 主内容容器。
// 通过 slots 承载路由页面，不读取业务 Store 或业务 API。
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
            class="rounded-item px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
            active-class="bg-surface-active text-foreground"
          >
            复习
          </RouterLink>
          <RouterLink
            to="/words"
            class="rounded-item px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
            active-class="bg-surface-active text-foreground"
          >
            生词库
          </RouterLink>
          <RouterLink
            to="/import"
            class="rounded-item px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
            active-class="bg-surface-active text-foreground"
          >
            导入
          </RouterLink>
        </nav>
      </div>
    </header>
    <main class="mx-auto w-full max-w-2xl flex-1 px-4 py-6">
      <slot />
    </main>
  </div>
</template>
