<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useQueryClient } from '@tanstack/vue-query'
import { CheckCircle, XCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui'
import { isApiError } from '@/lib/api'
import { useStartReviewSessionMutation, useSubmitReviewResultMutation } from '@/features/review/api/mutations'
import { reviewSessionQueryOptions } from '@/features/review/api/queries'
import { wordsListQueryOptions } from '@/features/words/api/queries'
import { formatMeanings } from '@/utils/formatMeanings'
import { resolveReviewKeyAction } from '@/utils/review-keyboard'
import { computeResumeProgress } from '@/utils/review-resume'
import {
  clearReviewSnapshot,
  loadReviewSnapshot,
  saveReviewSnapshot,
  type ReviewSnapshot,
} from '@/utils/review-snapshot'
import {
  advanceAfterAnswer,
  enterReview,
  revealCard,
  type ReviewMode,
} from '@/utils/review-state-machine'

// review-flow.md §6–§8、§10：复习页。
// 状态机 idle → recalling → revealed → answered → recalling → ... → finished（跳转结果页），
// 迁移判定与推进收敛在 utils/review-state-machine（本页持有 ref 状态与副作用）。
// 键盘操作绑定在复习区域：Space 揭示释义；1 / ← 不记得；2 / → 记得（§10、§5.4），
// 按键 → 操作的映射在 utils/review-keyboard，本页只负责 DOM 归约（焦点是否在控件上）与 preventDefault。
// D010：进入时检测 active session，由用户选择继续 / 放弃；无 abandon 端点，MVP 只清本地快照。
// 恢复检查复用 Feature 层查询（queryClient.fetchQuery），按 ApiError 区分 404 与瞬时失败。

const router = useRouter()
const queryClient = useQueryClient()

// ---- 数量选择 ----

const PRESET_COUNTS = [10, 20, 30, 50] as const
const selectedCount = ref<number>(30)
const customCountInput = ref('')
const customCountError = ref<string | null>(null)

// 可复习生词量取自 GET /words 分页 total（服务端候选集同为 deleted_at IS NULL 的生词），
// 仅用于开始前的 D009 提示；截断本身由服务端 min(count, available) 保证。
const availableCount = ref<number | null>(null)

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

type RecoveryState = 'idle' | 'checking' | 'recoverable' | 'error'

const recoveryState = ref<RecoveryState>('idle')
const savedSnapshot = ref<ReviewSnapshot | null>(null)
const savedProgress = ref<{ answered: number; total: number; resumeIndex: number } | null>(null)

// 开始前读取可复习生词量（D009 提示）；取不到就放弃提示，截断仍由服务端保证。
async function loadAvailableCount() {
  try {
    const data = await queryClient.fetchQuery(wordsListQueryOptions({ page: 1, pageSize: 1 }))
    availableCount.value = data.pagination.total
  } catch {
    // 网络等瞬时失败不阻塞页面，开始复习时由服务端截断兜底。
  }
}

const hasTruncationWarning = computed(() =>
  availableCount.value !== null && selectedCount.value > availableCount.value,
)

const truncatedCount = computed(() =>
  availableCount.value === null
    ? selectedCount.value
    : Math.min(selectedCount.value, availableCount.value),
)

// 进入页面时检查是否有可恢复的 active session（D010，由用户决定继续 / 放弃）。
async function checkActiveSession() {
  const snapshot = loadReviewSnapshot()
  if (!snapshot) {
    recoveryState.value = 'idle'
    return
  }

  recoveryState.value = 'checking'
  savedSnapshot.value = snapshot

  try {
    // 复用 Feature 层查询并强制取新：恢复检查不能拿到 30s 内的旧状态。
    const session = await queryClient.fetchQuery({
      ...reviewSessionQueryOptions(snapshot.sessionId),
      staleTime: 0,
    })

    if (session.status === 'completed') {
      discardSnapshot()
      void router.push({ name: 'review-result', params: { session: snapshot.sessionId } })
      return
    }

    if (session.status !== 'active') {
      // abandoned：服务端轮次已废弃，本地快照失效。
      discardSnapshot()
      recoveryState.value = 'idle'
      return
    }

    // 服务端逐词结果按 word 对齐本地快照，恢复到首个未答项（review-flow §6「回到原进度」）。
    const progress = computeResumeProgress(
      snapshot.items.map((item) => item.word),
      session.items,
    )
    savedProgress.value = {
      answered: progress
        ? progress.answered
        : session.items.filter((item) => item.result !== 'pending').length,
      total: snapshot.totalCount,
      resumeIndex: progress ? progress.firstPendingIndex : 0,
    }
    recoveryState.value = 'recoverable'
  } catch (err) {
    // 404：session 在服务端已不存在，快照失效；其余（网络 / 超时 / 5xx）为瞬时失败，
    // 保留快照供重试——不能因一次请求失败销毁恢复入口。
    if (isApiError(err) && err.httpStatus === 404) {
      discardSnapshot()
      recoveryState.value = 'idle'
      return
    }
    recoveryState.value = 'error'
  }
}

function discardSnapshot() {
  clearReviewSnapshot()
  savedSnapshot.value = null
  savedProgress.value = null
}

function handleResume() {
  const snapshot = savedSnapshot.value
  if (!snapshot || snapshot.items.length === 0) {
    discardSnapshot()
    recoveryState.value = 'idle'
    return
  }
  // 从首个未答项续答（review-flow §6）；对齐失败时回退第 1 题，已答题的重复提交
  // 经服务端幂等短路不重复计数（api/reviews.md §3），只是多做无效操作。
  const resumeIndex = savedProgress.value?.resumeIndex ?? 0
  items.value = snapshot.items
  sessionId.value = snapshot.sessionId
  const entry = enterReview(snapshot.items.length, resumeIndex)
  currentIndex.value = entry.currentIndex
  currentMode.value = entry.mode
  recoveryState.value = 'idle'
  void nextTick(() => focusReviewArea())
}

function handleAbandon() {
  discardSnapshot()
  recoveryState.value = 'idle'
}

// ---- 开始复习 ----

const startMutation = useStartReviewSessionMutation()
const isStarting = computed(() => startMutation.isPending.value)
const startError = ref<string | null>(null)

async function handleStart() {
  // 自定义输入未点「确认」就直接开始时，先应用输入值，避免按旧预设数量开局。
  if (customCountInput.value.trim()) {
    applyCustomCount()
    if (customCountError.value) return
  }
  await startNewSession(selectedCount.value)
}

async function startNewSession(count: number) {
  startError.value = null

  try {
    const response = await startMutation.mutateAsync({ count })

    items.value = response.items
    sessionId.value = response.sessionId
    const entry = enterReview(response.items.length)
    currentIndex.value = entry.currentIndex
    currentMode.value = entry.mode

    saveReviewSnapshot({
      sessionId: response.sessionId,
      totalCount: response.totalCount,
      items: response.items,
    })

    await nextTick()
    focusReviewArea()
  } catch (err) {
    startError.value = describeStartError(err)
  }
}

// 服务端 message 面向排查（英文原文），界面按错误分类给可读文案。
function describeStartError(err: unknown): string {
  if (isApiError(err)) {
    if (err.code === 'BASE.BIZ.USER_DISABLED') return '当前没有可复习的生词，请先导入生词。'
    if (err.kind === 'network' || err.kind === 'timeout') return '网络异常，请检查连接后重试。'
    if (err.kind === 'http' && (err.httpStatus ?? 0) >= 500) return '服务暂时不可用，请稍后重试。'
  }
  return '开始复习失败，请稍后重试'
}

// ---- 复习状态机 ----

const currentMode = ref<ReviewMode>('idle')
const items = ref<ReviewSnapshot['items']>([])
const currentIndex = ref(0)
const sessionId = ref<string | null>(null)

const currentItem = computed(() => {
  if (currentIndex.value < 0 || currentIndex.value >= items.value.length) return null
  return items.value[currentIndex.value]
})

const currentMeaning = computed(() => {
  if (!currentItem.value) return ''
  return formatMeanings(currentItem.value.effectiveReviewMeaning)
})

async function handleReveal() {
  const revealed = revealCard(currentMode.value, currentItem.value !== null)
  if (!revealed) return
  currentMode.value = revealed
  // 揭示后「查看释义」按钮随分支卸载会把焦点丢到 body，移回复习区域保住键盘操作（§5.3）。
  await nextTick()
  focusReviewArea()
}

const submitMutation = useSubmitReviewResultMutation()
const isSubmitting = computed(() => submitMutation.isPending.value)

async function handleAnswer(result: 'remembered' | 'forgotten') {
  if (currentMode.value !== 'revealed' || !currentItem.value || isSubmitting.value) return
  if (!sessionId.value) return

  try {
    await submitMutation.mutateAsync({
      sessionId: sessionId.value,
      itemId: currentItem.value.itemId,
      result,
    })

    const advance = advanceAfterAnswer(currentIndex.value, items.value.length)

    if (advance.outcome === 'next') {
      currentIndex.value = advance.nextIndex
      currentMode.value = advance.mode
      await nextTick()
      focusReviewArea()
    } else {
      // 最后一题，跳转结果页
      clearReviewSnapshot()
      void router.push({
        name: 'review-result',
        params: { session: sessionId.value },
      })
    }
  } catch {
    // 提交失败，保持当前状态，用户可重试
  }
}

// ---- 键盘操作 ----

// 绑定在复习区域元素上（代替 document 级监听）：快捷键只在复习区域持有焦点时生效，
// 不抢占页头导航 / 主题菜单等处的按键（交互与可访问性规范 §5.4）。
function handleKeydown(event: KeyboardEvent) {
  // 焦点落在按钮 / 链接等控件上时，Space / Enter 保留原生激活行为，不拦截。
  const target = event.target
  const onControl =
    target instanceof HTMLElement &&
    target.closest('button, a, input, textarea, select, [contenteditable]') !== null

  const action = resolveReviewKeyAction(
    { key: event.key, code: event.code },
    currentMode.value,
    onControl,
  )
  if (!action) return

  event.preventDefault()
  if (action === 'reveal') {
    void handleReveal()
  } else if (action === 'answerForgotten') {
    void handleAnswer('forgotten')
  } else {
    void handleAnswer('remembered')
  }
}

onMounted(() => {
  void checkActiveSession()
  void loadAvailableCount()
})

function focusReviewArea() {
  // 将焦点移到复习区域的提示文字上，确保键盘事件被捕获
  reviewAreaRef.value?.focus()
}

const reviewAreaRef = ref<HTMLElement | null>(null)

</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <h1 tabindex="-1" class="text-lg font-semibold text-foreground">开始复习</h1>

    <!-- 加载恢复状态中 -->
    <div v-if="recoveryState === 'checking'" class="mt-4">
      <p role="status" class="text-sm text-muted-foreground">
        正在检查复习进度…
      </p>
    </div>

    <!-- 恢复检查失败（瞬时错误：快照保留，可重试） -->
    <div v-else-if="recoveryState === 'error'" class="mt-4">
      <p role="alert" class="text-sm text-danger-text">
        检查复习进度失败，请重试。
      </p>
      <Button variant="outline" class="mt-3" @click="checkActiveSession">
        重试
      </Button>
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

      <!-- 截断提示（D009）：开始前提示可用量不足，截断由服务端 min(count, available) 保证 -->
      <p v-if="hasTruncationWarning" class="mt-3 text-sm text-muted-foreground">
        当前只有 {{ availableCount }} 个可复习生词，本轮将复习全部 {{ truncatedCount }} 个。
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

      <!-- 单词展示区：键盘快捷键绑定在此区域（§5.4 作用域），焦点经 tabindex=-1 承接 -->
      <div
        ref="reviewAreaRef"
        tabindex="-1"
        class="mt-8 flex flex-col items-center gap-6 text-center outline-hidden"
        @keydown="handleKeydown"
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
            {{ currentMeaning }}
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
