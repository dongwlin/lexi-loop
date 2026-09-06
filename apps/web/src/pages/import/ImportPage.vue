<script setup lang="ts">
import { computed, nextTick, ref, watchEffect } from 'vue'
import { RouterLink } from 'vue-router'
import { Button } from '@/components/ui'
import { useImportWordsMutation } from '@/features/words/api/mutations'
import { parseImportText } from '@/utils/parseImportText'
import type { ParsedWord } from '@/utils/parseImportText'

// review-flow.md §2：多行粘贴 → 前端解析 → 聚合提交 → 结果反馈（§3）。

const inputText = ref('')
const parsed = computed<ParsedWord[]>(() => parseImportText(inputText.value))
const totalEncounters = computed(() =>
  parsed.value.reduce((sum, w) => sum + w.count, 0),
)

const importMutation = useImportWordsMutation()

const isInputMode = ref(true)
const errorText = ref<string | null>(null)
// 程序化焦点目标（非原生可聚焦，临时 tabindex="-1"，见《前端交互与可访问性规范》§5.3）。
const errorSummaryRef = ref<HTMLElement | null>(null)
const resultHeadingRef = ref<HTMLElement | null>(null)

async function handleImport() {
  if (parsed.value.length === 0) return

  errorText.value = null
  try {
    await importMutation.mutateAsync({ words: parsed.value })
    isInputMode.value = false
    inputText.value = ''
    // 触发按钮随视图切换卸载，焦点移到结果标题作合理后继，避免丢到 body。
    await nextTick()
    resultHeadingRef.value?.focus()
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '导入失败，请重试'
    await nextTick()
    errorSummaryRef.value?.focus()
  }
}

function handleContinue() {
  isInputMode.value = true
  errorText.value = null
}

// 《前端交互与可访问性规范》§6.1：页面标题随路由更新（路由 meta 基建落地前由页面自行设置）。
watchEffect(() => {
  document.title = '导入生词 · LexiLoop'
})
</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <h1 class="text-lg font-semibold text-foreground">导入生词</h1>

    <!-- 输入模式：review-flow.md §2 -->
    <template v-if="isInputMode">
      <label
        for="import-textarea"
        class="mt-4 block text-sm text-muted-foreground"
      >
        每行输入一个单词
      </label>
      <textarea
        id="import-textarea"
        v-model="inputText"
        rows="12"
        placeholder="ambiguous&#10;constrain&#10;derive&#10;subtle&#10;constrain"
        class="mt-2 w-full resize-y rounded-field border border-field-border bg-field px-3 py-2.5 text-sm text-foreground shadow-field placeholder:text-muted-foreground focus:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      />

      <p class="mt-2 text-sm text-muted-foreground">
        共识别 <span class="font-medium text-foreground">{{ totalEncounters }}</span> 次
        <template v-if="parsed.length > 0">
          （{{ parsed.length }} 个不同单词）
        </template>
      </p>

      <p
        v-if="errorText"
        ref="errorSummaryRef"
        tabindex="-1"
        role="alert"
        class="mt-2 text-sm text-danger-text"
      >
        {{ errorText }}
      </p>

      <div class="mt-4 flex justify-end">
        <Button
          :disabled="parsed.length === 0"
          :pending="importMutation.isPending.value"
          @click="handleImport"
        >
          导入
        </Button>
      </div>
    </template>

    <!-- 结果模式：review-flow.md §3 聚合反馈 -->
    <template v-else>
      <div role="status" class="mt-4">
        <h2
          ref="resultHeadingRef"
          tabindex="-1"
          class="text-base font-medium text-foreground"
        >
          导入完成
        </h2>
        <dl class="mt-3 space-y-1 text-sm">
          <div class="flex gap-2">
            <dt class="w-28 text-muted-foreground">新增单词</dt>
            <dd class="font-medium text-foreground">
              {{ importMutation.data.value?.created ?? 0 }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-28 text-muted-foreground">已有单词</dt>
            <dd class="font-medium text-foreground">
              {{ importMutation.data.value?.updated ?? 0 }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-28 text-muted-foreground">本次遇到次数</dt>
            <dd class="font-medium text-foreground">
              {{ importMutation.data.value?.encounters ?? 0 }}
            </dd>
          </div>
        </dl>
      </div>

      <div class="mt-6 flex gap-3">
        <Button variant="secondary" @click="handleContinue">
          继续导入
        </Button>
        <RouterLink
          to="/words"
          class="inline-flex min-h-11 items-center rounded-control px-4 py-2 text-sm font-medium text-primary-text transition-colors hover:bg-primary-subtle"
        >
          回到生词库
        </RouterLink>
      </div>
    </template>
  </section>
</template>
