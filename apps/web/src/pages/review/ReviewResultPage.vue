<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { CheckCircle, RotateCcw } from 'lucide-vue-next'
import { Button } from '@/components/ui'
import { isApiError } from '@/lib/api'
import { useReviewSessionQuery } from '@/features/review/api/queries'

// review-flow.md §9：本轮复习结果（汇总 / 需要加强列表 / 再来一轮 / 回到生词库）。

const route = useRoute()
const router = useRouter()

const sessionId = computed(() => String(route.params.session))

const {
  data: session,
  isPending,
  isError,
  error,
  refetch,
  isFetching,
} = useReviewSessionQuery(sessionId)

const accuracy = computed(() => {
  if (!session.value || session.value.total === 0) return 0
  return Math.round((session.value.remembered / session.value.total) * 100)
})

const forgottenWords = computed(() =>
  (session.value?.items ?? []).filter((item) => item.result === 'forgotten'),
)

const isNotFound = computed(
  () => isApiError(error.value) && error.value.httpStatus === 404,
)

// 服务端 message 面向排查（英文原文），界面按错误分类给可读文案。
const errorMessage = computed(() => {
  const err = error.value
  if (isApiError(err)) {
    if (err.kind === 'network' || err.kind === 'timeout')
      return '网络异常，请检查连接后重试。'
    if (err.kind === 'http' && (err.httpStatus ?? 0) >= 500)
      return '服务暂时不可用，请稍后重试。'
  }
  return '复习结果加载失败，请稍后重试'
})

const notCompletedText = computed(() =>
  session.value?.status === 'abandoned'
    ? '本轮复习已放弃。'
    : '该复习轮次尚未完成。',
)

function goToReview() {
  void router.push({ name: 'review' })
}
</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <h1 tabindex="-1" class="text-lg font-semibold text-foreground">
      复习结果
    </h1>

    <!-- Session 不存在（含手工输入不存在的地址）：404 需先于通用 Error 判断 -->
    <template v-if="isPending">
      <p role="status" class="mt-4 text-sm text-muted-foreground">
        正在加载复习结果…
      </p>
      <div aria-hidden="true" class="mt-4 space-y-4 py-1">
        <div
          class="h-7 w-48 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
        />
        <div
          class="h-5 w-32 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
        />
        <div
          class="h-5 w-32 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
        />
        <div
          class="h-5 w-32 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
        />
      </div>
    </template>

    <template v-else-if="isNotFound">
      <p class="mt-4 text-sm text-muted-foreground">没有找到这轮复习。</p>
      <div class="mt-4 flex gap-3">
        <Button @click="goToReview">去复习</Button>
        <RouterLink
          to="/words"
          class="inline-flex min-h-11 items-center justify-center rounded-control border border-border-strong bg-transparent px-4 py-2 text-sm font-medium text-foreground transition-colors duration-150 ease-out hover:bg-default"
        >
          回到生词库
        </RouterLink>
      </div>
    </template>

    <!-- Error -->
    <template v-else-if="isError">
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

    <!-- Session 未完成 / 已放弃 -->
    <template v-else-if="session && session.status !== 'completed'">
      <p class="mt-4 text-sm text-muted-foreground">
        {{ notCompletedText }}
      </p>
      <div class="mt-4 flex gap-3">
        <Button @click="goToReview">去复习</Button>
        <RouterLink
          to="/words"
          class="inline-flex min-h-11 items-center justify-center rounded-control border border-border-strong bg-transparent px-4 py-2 text-sm font-medium text-foreground transition-colors duration-150 ease-out hover:bg-default"
        >
          回到生词库
        </RouterLink>
      </div>
    </template>

    <!-- 结果展示 -->
    <template v-else-if="session">
      <!-- 完成标记 -->
      <div class="mt-4 flex items-center gap-2">
        <CheckCircle
          class="size-5 shrink-0 text-success-text"
          aria-hidden="true"
        />
        <span class="text-base font-medium text-foreground">本轮完成</span>
      </div>

      <!-- 汇总统计 -->
      <dl class="mt-4 space-y-2 text-sm">
        <div class="flex items-baseline justify-between gap-6">
          <dt class="text-muted-foreground">总计</dt>
          <dd class="font-medium text-foreground tabular-nums">
            {{ session.total }} 个单词
          </dd>
        </div>
        <div class="flex items-baseline justify-between gap-6">
          <dt class="text-muted-foreground">记得</dt>
          <dd class="font-medium text-foreground tabular-nums">
            {{ session.remembered }}
          </dd>
        </div>
        <div class="flex items-baseline justify-between gap-6">
          <dt class="text-muted-foreground">不记得</dt>
          <dd class="font-medium text-foreground tabular-nums">
            {{ session.forgotten }}
          </dd>
        </div>
        <div class="flex items-baseline justify-between gap-6">
          <dt class="text-muted-foreground">正确率</dt>
          <dd class="font-medium text-foreground tabular-nums">
            {{ accuracy }}%
          </dd>
        </div>
      </dl>

      <!-- 需要加强的单词列表 -->
      <template v-if="forgottenWords.length > 0">
        <hr class="my-6 border-t border-border" />
        <h2 class="text-sm font-medium text-foreground">需要加强</h2>
        <ul class="mt-2 space-y-1">
          <li
            v-for="item in forgottenWords"
            :key="item.word"
            class="text-sm text-muted-foreground"
          >
            {{ item.word }}
          </li>
        </ul>
      </template>

      <!-- 操作按钮 -->
      <div class="mt-6 flex gap-3">
        <Button class="gap-2" @click="goToReview">
          <RotateCcw class="size-4 shrink-0" aria-hidden="true" />
          再来一轮
        </Button>
        <RouterLink
          to="/words"
          class="inline-flex min-h-11 items-center justify-center rounded-control border border-border-strong bg-transparent px-4 py-2 text-sm font-medium text-foreground transition-colors duration-150 ease-out hover:bg-default"
        >
          回到生词库
        </RouterLink>
      </div>
    </template>
  </section>
</template>
