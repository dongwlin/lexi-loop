<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui'
import { useVersionQuery } from '@/features/meta/api/queries'
import { formatDateTime } from '@/utils/formatDateTime'

// 「关于」页：展示后端 /api/v1/version 返回的版本信息（docs/api/meta.md）。
// 版本信息单一来源在后端构建期注入（server 的 infra/buildinfo），前端不重复
// 注入；本地开发构建版本为 dev 占位、构建时间为空（复用 formatDateTime，
// 空值 / 无效值兜底展示「—」）。

const {
  data,
  isPending,
  isError,
  error,
  isFetching,
  refetch,
} = useVersionQuery()

const errorMessage = computed(() =>
  error.value instanceof Error
    ? error.value.message
    : '版本信息加载失败，请稍后重试',
)

const rows = computed(() => {
  const version = data.value
  if (!version) return []
  return [
    { label: '应用版本', value: version.version },
    { label: '构建时间', value: formatDateTime(version.buildTime) ?? '—' },
    { label: 'Go 版本', value: version.goVersion },
  ]
})
</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <template v-if="isError">
      <h1 tabindex="-1" class="text-lg font-semibold text-foreground">关于</h1>
      <p role="alert" class="mt-4 text-sm text-danger-text">
        {{ errorMessage }}
      </p>
      <Button
        variant="outline"
        class="mt-3"
        :pending="isFetching"
        @click="refetch()"
      >
        重试
      </Button>
    </template>

    <!-- Loading：骨架形状接近最终结构（§9.1），仅首次加载展示 -->
    <template v-else-if="isPending">
      <h1 tabindex="-1" class="text-lg font-semibold text-foreground">关于</h1>
      <p role="status" class="mt-2 text-sm text-muted-foreground">
        正在加载版本信息…
      </p>
      <div aria-hidden="true" class="mt-6 space-y-2.5 py-1">
        <div
          v-for="index in 3"
          :key="index"
          class="flex items-center justify-between"
        >
          <div
            class="h-5 w-16 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
          />
          <div
            class="h-5 w-24 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
          />
        </div>
      </div>
    </template>

    <template v-else-if="data">
      <h1 tabindex="-1" class="text-lg font-semibold text-foreground">关于</h1>
      <p class="mt-2 text-sm text-muted-foreground">
        LexiLoop（词环）：英语生词复习系统。
      </p>
      <dl class="mt-6 space-y-2.5 text-sm">
        <div
          v-for="row in rows"
          :key="row.label"
          class="flex items-baseline justify-between gap-6"
        >
          <dt class="text-muted-foreground">{{ row.label }}</dt>
          <dd class="font-medium text-foreground tabular-nums">
            {{ row.value }}
          </dd>
        </div>
      </dl>
    </template>
  </section>
</template>
