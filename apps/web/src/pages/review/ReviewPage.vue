<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { CheckCircle, XCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui'
import { useStartReviewSessionMutation, useSubmitReviewResultMutation } from '@/features/review/api/mutations'
import { formatMeanings } from '@/utils/formatMeanings'
import {
  clearReviewSnapshot,
  loadReviewSnapshot,
  saveReviewSnapshot,
  type ReviewSnapshot,
} from '@/utils/review-snapshot'

// review-flow.md §6–§8、§10：复习页。
// 状态机 idle → recalling → revealed → answered → recalling → ... → finished（跳转结果页）。
// 键盘操作 Space 揭示释义；1 / ← 不记得；2 / → 记得（§10）。
// D010：进入时检测 active session，由用户选择继续 / 放弃；无 abandon 端点，MVP 只清本地快照。

const router = useRouter()

// ---- 数量选择 ----

const PRESET_COUNTS = [10, 20, 30, 50] as const
const selectedCount = ref<number>(30)
const customCountInput = ref('')
const customCountError = ref<string | null>(null)

// 词库总数由服务端返回；开始后服务端按 min(count, available) 截断（D009）。
// 页面在 count > totalCount 时给出提示，但不阻止开始。
const totalCount = ref<number | null>(null)

function selectPreset(count: number) {
  selectedCount.value = count
  customCountInput.value = ''
  customCountError.value = null
}

function applyCustomCount() {
  const raw = customCountInput.value.trim()
  const parsed = Number(raw)
  if (!Number.isSafeInteger(parsed) || parsed < 1) {
    customCountError.value = '请输入正整数'
    return
  }
  customCountError.value = null
  selectedCount.value = parsed
}

function handleCountKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    event.preventDefault()
    applyCustomCount()
  }
}

// ---- Active session 恢复（D010） ----

type RecoveryState = 'idle' | 'checking' | 'recoverable' | 'recovering'

const recoveryState = ref<RecoveryState>('idle')
const savedSnapshot = ref<ReviewSnapshot | null>(null)
const savedProgress = ref<{ answered: number; total: number } | null>(null)

const hasTruncationWarning = computed(() => {
  if (totalCount.value === null) return false
  return selectedCount.value > totalCount.value
})

const truncatedCount = computed(() => {
  if (totalCount.value === null) return selectedCount.value
  return Math.min(selectedCount.value, totalCount.value)
})

// 检查是否有可恢复的 active session（进入时 + 路由变化）。
async function checkActiveSession() {
  const snapshot = loadReviewSnapshot()
  if (!snapshot) {
    recoveryState.value = 'idle'
    return
  }

  recoveryState.value = 'checking'
  savedSnapshot.value = snapshot

  try {
    // useReviewSessionQuery 需要手动触发——直接用 queryClient 获取或 fetch
    const response = await fetch(
      `${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/reviews/${snapshot.sessionId}`,
    )
    if (!response.ok) throw new Error('fetch failed')
    const body = (await response.json()) as { code: string; data?: { status: string; items?: Array<{ result: string }> } }

    if (body.code !== 'OK' || !body.data) {
      clearReviewSnapshot()
      recoveryState.value = 'idle'
      return
    }

    const status = body.data.status
    if (status === 'completed') {
      clearReviewSnapshot()
      void router.push({ name: 'review-result', params: { session: snapshot.sessionId } })
      return
    }

    if (status !== 'active') {
      clearReviewSnapshot()
      recoveryState.value = 'idle'
      return
    }

    // 计算已回答数量：服务端 items 中 result !== 'pending' 的数量
    const answered = (body.data.items ?? []).filter(
      (item) => item.result !== 'pending',
    ).length
    savedProgress.value = { answered, total: snapshot.totalCount }
    recoveryState.value = 'recoverable'
  } catch {
    clearReviewSnapshot()
    recoveryState.value = 'idle'
  }
}

function handleResume() {
  if (!savedSnapshot.value || !savedProgress.value) return
  const snapshot = savedSnapshot.value
  // 恢复时 count 等于 snapshot.totalCount（服务端返回的实际抽取数）
  startSessionDirectly(snapshot.totalCount, snapshot)
}

function handleAbandon() {
  clearReviewSnapshot()
  savedSnapshot.value = null
  savedProgress.value = null
  recoveryState.value = 'idle'
}

// ---- 开始复习 ----

const startMutation = useStartReviewSessionMutation()
const isStarting = computed(() => startMutation.isPending.value)
const startError = ref<string | null>(null)

async function handleStart() {
  await startSessionDirectly(selectedCount.value)
}

async function startSessionDirectly(
  count: number,
  existingSnapshot?: ReviewSnapshot,
) {
  startError.value = null

  // 如果有 existingSnapshot 且 count 匹配，直接用它
  if (existingSnapshot && existingSnapshot.totalCount === count) {
    const snapshot = existingSnapshot
    items.value = snapshot.items
    currentIndex.value = snapshot.items.length > 0 ? 0 : -1
    currentMode.value = 'recalling'
    totalCount.value = snapshot.totalCount
    await nextTick()
    focusReviewArea()
    return
  }

  try {
    const response = await startMutation.mutateAsync({ count })

    items.value = response.items
    currentIndex.value = response.items.length > 0 ? 0 : -1
    totalCount.value = response.totalCount
    currentMode.value = response.items.length > 0 ? 'recalling' : 'idle'

    // 保存快照
    saveReviewSnapshot({
      sessionId: response.sessionId,
      totalCount: response.totalCount,
      items: response.items,
    })

    await nextTick()
    focusReviewArea()
  } catch (err) {
    startError.value =
      err instanceof Error ? err.message : '开始复习失败，请稍后重试'
  }
}

// ---- 复习状态机 ----

type ReviewMode = 'idle' | 'recalling' | 'revealed'

const currentMode = ref<ReviewMode>('idle')
const items = ref<ReviewSnapshot['items']>([])
const currentIndex = ref(0)
const answeredCount = ref(0)

const currentItem = computed(() => {
  if (currentIndex.value < 0 || currentIndex.value >= items.value.length) return null
  return items.value[currentIndex.value]
})

const currentMeaning = computed(() => {
  if (!currentItem.value) return ''
  return formatMeanings(currentItem.value.effectiveReviewMeaning)
})

function handleReveal() {
  if (currentMode.value !== 'recalling' || !currentItem.value) return
  currentMode.value = 'revealed'
}

const submitMutation = useSubmitReviewResultMutation()
const isSubmitting = computed(() => submitMutation.isPending.value)

async function handleAnswer(result: 'remembered' | 'forgotten') {
  if (currentMode.value !== 'revealed' || !currentItem.value || isSubmitting.value) return

  const snapshot = loadReviewSnapshot()
  if (!snapshot) return

  try {
    await submitMutation.mutateAsync({
      sessionId: snapshot.sessionId,
      itemId: currentItem.value.itemId,
      result,
    })

    answeredCount.value++
    const nextIndex = currentIndex.value + 1

    if (nextIndex < items.value.length) {
      currentIndex.value = nextIndex
      currentMode.value = 'recalling'
      await nextTick()
      focusReviewArea()
    } else {
      // 最后一题，跳转结果页
      clearReviewSnapshot()
      void router.push({
        name: 'review-result',
        params: { session: snapshot.sessionId },
      })
    }
  } catch {
    // 提交失败，保持当前状态，用户可重试
  }
}

// ---- 键盘操作 ----

function handleKeydown(event: KeyboardEvent) {
  // 只在复习阶段响应快捷键，忽略输入框焦点
  if (currentMode.value === 'idle') return
  if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return

  if (event.key === ' ' || event.code === 'Space') {
    event.preventDefault()
    handleReveal()
  } else if (
    currentMode.value === 'revealed' &&
    (event.key === '1' || event.key === 'ArrowLeft')
  ) {
    event.preventDefault()
    void handleAnswer('forgotten')
  } else if (
    currentMode.value === 'revealed' &&
    (event.key === '2' || event.key === 'ArrowRight')
  ) {
    event.preventDefault()
    void handleAnswer('remembered')
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
  void checkActiveSession()
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})

function focusReviewArea() {
  // 将焦点移到复习区域的提示文字上，确保键盘事件被捕获
  reviewAreaRef.value?.focus()
}

const reviewAreaRef = ref<HTMLElement | null>(null)

// 《前端交互与可访问性规范》§6.1：页面标题随路由更新。
watchEffect(() => {
  document.title = '复习 · LexiLoop'
})
</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <h1 class="text-lg font-semibold text-foreground">开始复习</h1>

    <!-- 加载恢复状态中 -->
    <div v-if="recoveryState === 'checking'" class="mt-4">
      <p role="status" class="text-sm text-muted-foreground">
        正在检查复习进度…
      </p>
    </div>

    <!-- 可恢复的 active session -->
    <div v-else-if="recoveryState === 'recoverable' && savedProgress" class="mt-4">
      <p class="text-sm text-foreground">
        上次复习未完成（{{ savedProgress.answered }} / {{ savedProgress.total }}）。
      </p>
      <div class="mt-4 flex gap-3">
        <Button @click="handleResume">继续复习</Button>
        <Button variant="outline" @click="handleAbandon">放弃本轮</Button>
      </div>
    </div>

    <!-- 设置阶段：数量选择 -->
    <template v-else-if="currentMode === 'idle'">
      <p class="mt-4 text-sm text-muted-foreground">
        今天想复习多少个单词？
      </p>

      <!-- 预设数量按钮 -->
      <div class="mt-4 flex flex-wrap gap-2">
        <Button
          v-for="count in PRESET_COUNTS"
          :key="count"
          :variant="selectedCount === count && !customCountInput ? 'secondary' : 'ghost'"
          @click="selectPreset(count)"
        >
          {{ count }}
        </Button>
      </div>

      <!-- 自定义输入 -->
      <div class="mt-3 flex items-end gap-2">
        <div class="flex-1">
          <label for="custom-count" class="block text-sm text-muted-foreground">
            或者自定义数量
          </label>
          <input
            id="custom-count"
            v-model="customCountInput"
            type="number"
            min="1"
            placeholder="例如 25"
            class="mt-2 w-full min-h-11 rounded-field border border-field-border bg-field px-3 py-2 text-sm text-foreground shadow-field placeholder:text-muted-foreground enabled:hover:bg-field-hover outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            @keydown="handleCountKeydown"
            @input="customCountError = null"
          />
          <p v-if="customCountError" role="alert" class="mt-1 text-xs text-danger-text">
            {{ customCountError }}
          </p>
        </div>
        <Button
          variant="secondary"
          class="shrink-0"
          :disabled="!customCountInput.trim()"
          @click="applyCustomCount"
        >
          确认
        </Button>
      </div>

      <!-- 截断提示（D009） -->
      <p v-if="hasTruncationWarning" class="mt-3 text-sm text-muted-foreground">
        当前只有 {{ totalCount }} 个可复习生词，本轮将复习全部 {{ truncatedCount }} 个。
      </p>

      <!-- 错误提示 -->
      <p
        v-if="startError"
        role="alert"
        class="mt-3 text-sm text-danger-text"
      >
        {{ startError }}
      </p>

      <!-- 开始按钮 -->
      <div class="mt-6 flex justify-end">
        <Button :pending="isStarting" @click="handleStart">开始复习</Button>
      </div>
    </template>

    <!-- 复习阶段：闪卡 -->
    <template v-else-if="currentItem">
      <!-- 进度 -->
      <p role="status" class="mt-4 text-center text-sm text-muted-foreground">
        {{ currentIndex + 1 }} / {{ items.length }}
      </p>

      <!-- 单词展示区 -->
      <div
        ref="reviewAreaRef"
        tabindex="-1"
        class="mt-8 flex flex-col items-center gap-6 text-center outline-hidden"
      >
        <h2 class="break-words text-3xl font-semibold text-foreground">
          {{ currentItem.word }}
        </h2>

        <!-- recalling：提示回忆 -->
        <template v-if="currentMode === 'recalling'">
          <p class="text-sm text-muted-foreground">
            先回忆这个单词的意思
          </p>
          <Button
            class="mt-4"
            @click="handleReveal"
          >
            查看释义
          </Button>
        </template>

        <!-- revealed：显示释义 + 作答按钮 -->
        <template v-else-if="currentMode === 'revealed'">
          <p class="text-base text-foreground">
            {{ currentMeaning || '暂无释义' }}
          </p>
          <div class="mt-4 flex gap-4">
            <Button
              variant="outline"
              class="gap-2"
              :disabled="isSubmitting"
              @click="void handleAnswer('forgotten')"
            >
              <XCircle class="size-4 shrink-0" aria-hidden="true" />
              不记得
            </Button>
            <Button
              class="gap-2"
              :disabled="isSubmitting"
              @click="void handleAnswer('remembered')"
            >
              <CheckCircle class="size-4 shrink-0" aria-hidden="true" />
              记得
            </Button>
          </div>
        </template>
      </div>

      <!-- 快捷键提示 -->
      <p class="mt-8 text-center text-xs text-muted-foreground">
        <template v-if="currentMode === 'recalling'">
          按 <kbd class="rounded-field bg-surface-tertiary px-1.5 py-0.5 font-mono text-xs">Space</kbd> 查看释义
        </template>
        <template v-else-if="currentMode === 'revealed'">
          <kbd class="rounded-field bg-surface-tertiary px-1.5 py-0.5 font-mono text-xs">1</kbd> /
          <kbd class="rounded-field bg-surface-tertiary px-1.5 py-0.5 font-mono text-xs">←</kbd>
          不记得
          <span class="mx-2">·</span>
          <kbd class="rounded-field bg-surface-tertiary px-1.5 py-0.5 font-mono text-xs">2</kbd> /
          <kbd class="rounded-field bg-surface-tertiary px-1.5 py-0.5 font-mono text-xs">→</kbd>
          记得
        </template>
      </p>
    </template>
  </section>
</template>
