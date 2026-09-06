<script setup lang="ts">
// 兜底 404 页：处理未匹配路由（前端应用架构规范 §11 的 Not Found 分支）。
</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <h1 tabindex="-1" class="text-lg font-semibold text-foreground">页面不存在</h1>
    <p class="mt-2 text-sm text-muted-foreground">访问的地址不存在或已被移动。</p>
    <RouterLink
      to="/"
      class="mt-4 inline-flex min-h-11 items-center rounded-control px-4 py-2 text-sm font-medium text-primary-text transition-colors hover:bg-primary-subtle"
    >
      回到首页
    </RouterLink>
  </section>
</template>
