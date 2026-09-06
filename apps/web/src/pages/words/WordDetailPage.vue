<script setup lang="ts">
import { computed, nextTick, ref, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from 'lucide-vue-next'
import {
  AlertDialog,
  AlertDialogCancel,
  Button,
  Dialog,
  DialogClose,
} from '@/components/ui'
import {
  useDeleteWordMutation,
  useUpdateReviewMeaningMutation,
} from '@/features/words/api/mutations'
import { useWordDetailQuery } from '@/features/words/api/queries'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/utils/formatDateTime'
import { formatMeaningText, parseMeaningText } from '@/utils/meaningText'
import type { Meaning } from '@lexi-loop/api-client'

// review-flow.md §5：单词详情。展示学习统计与生效释义（三层取值 custom ?? review ?? raw，
// D007），自定义来源标注「自定义」；编辑复习释义走 PATCH customReviewMeaning（清空保存 =
// null 清除并回退词典层），删除为软删除确认（D004，重新导入可恢复）。掌握度 / 优先级依赖
// 服务端契约扩展（README 待办），落地前不展示（D011）。

const route = useRoute()
const router = useRouter()

const wordId = computed(() => String(route.params.id))

const {
  data: detail,
  isPending,
  isError,
  error,
  isFetching,
  refetch,
} = useWordDetailQuery(wordId)

const isNotFound = computed(
  () =>
    isError.value &&
    error.value instanceof ApiError &&
    error.value.httpStatus === 404,
)
const errorMessage = computed(() =>
  error.value instanceof Error
    ? error.value.message
    : '单词详情加载失败，请稍后重试',
)

// 生效释义逐条渲染；没有可展示译文的义项（如纯词性占位）不出现。
const senseLines = computed(() =>
  (detail.value?.effectiveReviewMeaning ?? []).flatMap((sense) => {
    const translations = (sense.translations ?? []).filter(
      (text) => text.length > 0,
    )
    if (translations.length === 0) return []
    return [{ pos: sense.pos, text: translations.join('；') }]
  }),
)

const stats = computed(() => {
  const word = detail.value
  if (!word) return []
  return [
    { label: '遇到次数', value: String(word.encounterCount) },
    { label: '复习次数', value: String(word.reviewCount) },
    { label: '记得次数', value: String(word.rememberCount) },
    { label: '忘记次数', value: String(word.forgetCount) },
    { label: '上次复习', value: formatDateTime(word.lastReviewedAt) ?? '—' },
  ]
})

// ---- 编辑复习释义 ----

const updateMutation = useUpdateReviewMeaningMutation()
const isUpdating = computed(() => updateMutation.isPending.value)

const editOpen = ref(false)
const meaningInput = ref('')
// 打开时的文本基准：有自定义释义时 effective 即自定义文本；没有时以词典层释义为底稿（D007）。
// 保存按钮在文本未变化时禁用，避免无改动保存把词典释义误存成自定义释义。
const prefillText = ref('')
const editError = ref<string | null>(null)
const editErrorRef = ref<HTMLElement | null>(null)

const hasCustomMeaning = computed(
  () => detail.value?.meaningSource === 'custom',
)
const isMeaningDirty = computed(() => meaningInput.value !== prefillText.value)

function setEditOpen(open: boolean) {
  // 提交中拦截关闭（交互与可访问性规范 §7.2），避免丢失提交状态。
  if (isUpdating.value) return
  editError.value = null
  if (open) {
    prefillText.value = formatMeaningText(detail.value?.effectiveReviewMeaning)
    meaningInput.value = prefillText.value
  }
  editOpen.value = open
}

async function submitMeaning(customReviewMeaning: Meaning[] | null) {
  editError.value = null
  try {
    await updateMutation.mutateAsync({ id: wordId.value, customReviewMeaning })
    // 成功后由失效重取的详情证明更新（§9.2），焦点由 Reka 还原到触发按钮。
    setEditOpen(false)
  } catch (err) {
    editError.value =
      err instanceof Error ? err.message : '保存失败，请重试'
    await nextTick()
    editErrorRef.value?.focus()
  }
}

// 清空即清除自定义释义（PATCH null，D007 回退 review ?? raw）。
async function handleSaveMeaning() {
  if (!detail.value || !isMeaningDirty.value) return
  const parsed = parseMeaningText(meaningInput.value)
  await submitMeaning(parsed.length > 0 ? parsed : null)
}

async function handleClearMeaning() {
  if (!detail.value) return
  await submitMeaning(null)
}

// ---- 删除生词（软删除，D004） ----

const deleteMutation = useDeleteWordMutation()
const isDeleting = computed(() => deleteMutation.isPending.value)

const deleteOpen = ref(false)
const deleteError = ref<string | null>(null)
const deleteErrorRef = ref<HTMLElement | null>(null)

const deleteDescription = computed(
  () =>
    `将删除单词「${detail.value?.word ?? ''}」。删除后它不再出现在生词库与复习中；重新导入同一单词可恢复其学习记录与遇词次数。`,
)

function setDeleteOpen(open: boolean) {
  if (isDeleting.value) return
  deleteError.value = null
  deleteOpen.value = open
}

// 确认按钮不复用 AlertDialogAction 的自动关闭：对话框在提交期间保持打开，
// 成功后再关闭并返回生词库（删除后详情 404，不再停留）。
async function handleDelete() {
  if (!detail.value || isDeleting.value) return
  deleteError.value = null
  try {
    await deleteMutation.mutateAsync(wordId.value)
    deleteOpen.value = false
    void router.push({ name: 'words' })
  } catch (err) {
    deleteError.value =
      err instanceof Error ? err.message : '删除失败，请重试'
    await nextTick()
    deleteErrorRef.value?.focus()
  }
}

// 《前端交互与可访问性规范》§6.1：页面标题随路由更新（路由 meta 基建落地前由页面自行设置）。
watchEffect(() => {
  document.title = detail.value
    ? `${detail.value.word} · LexiLoop`
    : '单词详情 · LexiLoop'
})
</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <RouterLink
      to="/words"
      class="inline-flex min-h-11 items-center gap-1.5 rounded-item text-sm text-muted-foreground transition-colors duration-150 ease-out hover:bg-surface-hover hover:text-foreground outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      <ArrowLeft class="size-4 shrink-0" aria-hidden="true" />
      返回生词库
    </RouterLink>

    <!-- 页面级 Error：区分「不存在」（软删除 / 链接失效）与其他加载失败（§9.3） -->
    <template v-if="isNotFound">
      <h1 tabindex="-1" class="mt-4 text-lg font-semibold text-foreground">没有找到这个单词</h1>
      <p class="mt-2 text-sm text-muted-foreground">
        它可能已被删除，或链接不正确。重新导入同一单词可恢复其学习记录。
      </p>
      <div class="mt-4 flex gap-3">
        <RouterLink
          to="/words"
          class="inline-flex min-h-11 items-center justify-center rounded-control bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-[background-color] duration-150 ease-out hover:bg-primary-hover"
        >
          回到生词库
        </RouterLink>
        <RouterLink
          to="/import"
          class="inline-flex min-h-11 items-center justify-center rounded-control border border-border-strong bg-transparent px-4 py-2 text-sm font-medium text-foreground transition-colors duration-150 ease-out hover:bg-default"
        >
          去导入
        </RouterLink>
      </div>
    </template>

    <template v-else-if="isError">
      <h1 tabindex="-1" class="mt-4 text-lg font-semibold text-foreground">单词详情</h1>
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
      <h1 tabindex="-1" class="mt-4 text-lg font-semibold text-foreground">单词详情</h1>
      <p role="status" class="mt-2 text-sm text-muted-foreground">
        正在加载单词详情…
      </p>
      <div aria-hidden="true" class="mt-4 space-y-4 py-1">
        <div class="h-7 w-40 rounded-field bg-surface-tertiary motion-safe:animate-pulse" />
        <div class="h-5 w-64 rounded-field bg-surface-tertiary motion-safe:animate-pulse" />
        <div class="h-px bg-border" />
        <div v-for="index in 5" :key="index" class="flex items-center justify-between">
          <div class="h-5 w-16 rounded-field bg-surface-tertiary motion-safe:animate-pulse" />
          <div class="h-5 w-10 rounded-field bg-surface-tertiary motion-safe:animate-pulse" />
        </div>
      </div>
    </template>

    <template v-else-if="detail">
      <div class="mt-4 flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h1 tabindex="-1" class="break-words text-2xl font-semibold text-foreground">
          {{ detail.word }}
        </h1>
        <span v-if="detail.phonetic" class="text-sm text-muted-foreground">
          {{ detail.phonetic }}
        </span>
      </div>

      <!-- 生效释义（D007 三层取值），自定义来源标注「自定义」 -->
      <div v-if="senseLines.length > 0" class="mt-2">
        <span
          v-if="hasCustomMeaning"
          class="inline-flex items-center rounded-full bg-primary-subtle px-2.5 py-0.5 text-xs font-medium text-primary-text"
        >
          自定义
        </span>
        <ul class="mt-1 space-y-1 text-base text-foreground">
          <li v-for="(sense, index) in senseLines" :key="index">
            <span v-if="sense.pos" class="text-muted-foreground">
              {{ sense.pos }}.
            </span>
            {{ sense.text }}
          </li>
        </ul>
      </div>
      <p v-else class="mt-2 text-sm text-muted-foreground">
        该单词暂无释义，可通过下方「编辑复习释义」添加自己的复习释义。
      </p>

      <hr class="my-6 border-t border-border">

      <!-- 学习统计（review-flow §5；无掌握度 / 优先级，D011 契约缺口） -->
      <dl class="space-y-2.5 text-sm">
        <div
          v-for="stat in stats"
          :key="stat.label"
          class="flex items-baseline justify-between gap-6"
        >
          <dt class="text-muted-foreground">{{ stat.label }}</dt>
          <dd class="font-medium tabular-nums text-foreground">{{ stat.value }}</dd>
        </div>
      </dl>

      <div class="mt-6 flex flex-wrap gap-3">
        <Dialog
          :open="editOpen"
          title="编辑复习释义"
          description="只修改你自己的复习展示，不会改动共享词典数据。"
          @update:open="setEditOpen"
        >
          <template #trigger>
            <Button variant="secondary">编辑复习释义</Button>
          </template>

          <label
            for="custom-meaning-input"
            class="block text-sm font-medium text-foreground"
          >
            自定义复习释义
          </label>
          <textarea
            id="custom-meaning-input"
            v-model="meaningInput"
            data-dialog-initial-focus
            rows="4"
            aria-describedby="custom-meaning-help"
            placeholder="adj. 模棱两可的；含糊不清的"
            class="mt-2 w-full resize-y rounded-field border border-field-border bg-field px-3 py-2 text-sm text-foreground shadow-field placeholder:text-muted-foreground enabled:hover:bg-field-hover outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          />
          <p id="custom-meaning-help" class="mt-2 text-xs text-muted-foreground">
            每行一条义项，可带词性前缀；清空并保存表示清除自定义释义、回退到词典释义。
          </p>
          <p
            v-if="editError"
            ref="editErrorRef"
            tabindex="-1"
            role="alert"
            class="mt-2 text-sm text-danger-text"
          >
            {{ editError }}
          </p>

          <template #footer>
            <Button
              v-if="hasCustomMeaning"
              variant="danger-soft"
              class="mr-auto"
              :disabled="isUpdating"
              @click="handleClearMeaning"
            >
              清除自定义释义
            </Button>
            <DialogClose as-child>
              <Button variant="outline" :disabled="isUpdating">取消</Button>
            </DialogClose>
            <Button
              :disabled="!isMeaningDirty || isUpdating"
              :pending="isUpdating"
              @click="handleSaveMeaning"
            >
              保存修改
            </Button>
          </template>
        </Dialog>

        <AlertDialog
          :open="deleteOpen"
          title="删除生词"
          :description="deleteDescription"
          @update:open="setDeleteOpen"
        >
          <template #trigger>
            <Button variant="danger-soft">删除生词</Button>
          </template>

          <p
            v-if="deleteError"
            ref="deleteErrorRef"
            tabindex="-1"
            role="alert"
            class="text-sm text-danger-text"
          >
            {{ deleteError }}
          </p>

          <template #footer>
            <AlertDialogCancel as-child>
              <Button variant="outline" :disabled="isDeleting">取消</Button>
            </AlertDialogCancel>
            <Button variant="danger" :pending="isDeleting" @click="handleDelete">
              删除生词
            </Button>
          </template>
        </AlertDialog>
      </div>
    </template>
  </section>
</template>
