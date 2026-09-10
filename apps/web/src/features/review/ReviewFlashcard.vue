<script setup lang="ts">
import { CheckCircle, XCircle } from 'lucide-vue-next'
import { computed } from 'vue'
import ReviewPronunciation from './ReviewPronunciation.vue'
import { Button } from '@/components/ui'
import type { ReviewMode, ReviewResult } from '@/utils/review-state-machine'

// 页面负责会话、提交与焦点；闪卡仅展示当前阶段并发出用户意图。
// 自评用词统一为「记得 / 没记住」（product/prd.md §8：判断依据是是否回忆成功，
// 不能只凭眼熟作答）；「认识 / 不认识」问的是眼熟，与任务、结果页措辞都冲突。
const props = withDefaults(
  defineProps<{
    autoPlayPronunciation?: boolean
    phonetic?: string
    word: string
    meaning: string
    mode: ReviewMode
    canCorrect: boolean
    submitting: boolean
    /** 本卡是否已发起过提交；false 表示结果仍是暂定值，点「下一词」才落库（review-flow §7）。 */
    submissionStarted: boolean
    error: string | null
  }>(),
  { autoPlayPronunciation: true, phonetic: '' },
)
defineEmits<{
  choose: [result: ReviewResult]
  correct: []
  next: []
}>()

const RECALLING_HINT = '先回忆这个单词的意思'

const isRevealed = computed(() => props.mode === 'revealed')
// 提示与释义共用同一个常驻元素，见模板内 live region 说明。
const slotText = computed(() =>
  isRevealed.value ? props.meaning : RECALLING_HINT,
)
</script>

<template>
  <div class="flex w-full min-w-0 flex-col items-center gap-6">
    <div class="flex max-w-full flex-col items-center gap-1">
      <h2
        class="max-w-full text-3xl font-semibold wrap-anywhere text-foreground"
      >
        {{ word }}
      </h2>
      <ReviewPronunciation
        :word="word"
        :phonetic="phonetic"
        :auto-play-pronunciation="autoPlayPronunciation"
      />
    </div>
    <!-- 共用内容槽吸收常规释义行数差异；超长内容自然撑开，不隐藏待回忆的释义。
         预留高度按「4 行释义 + 暂定标记」取（4 × 24 + gap 12 + 标记 16 = 124 ≤ min-h-32），
         e2e 覆盖的 1～4 行释义在揭示前后不改变卡片高度与按钮位置；移动端因此与桌面端
         取同一预留值，不再区分断点。
         提示与释义是同一个常驻的 role="status" 元素：容器先于内容存在，文本变化才
         触发播报（复用发音状态已有的 live region 模式），揭示这个最重要的状态变化
         因此对屏幕阅读器可见；容器只播报，不抢焦点也不移动焦点。 -->
    <div
      class="flex min-h-32 w-full flex-col items-center justify-center gap-3"
    >
      <p
        role="status"
        class="max-w-full wrap-anywhere whitespace-pre-line"
        :class="
          isRevealed
            ? 'text-base text-foreground'
            : 'text-sm text-muted-foreground'
        "
      >
        {{ slotText }}
      </p>
      <!-- 暂定结果的可见标记：延迟提交（点「下一词」才落库）是一条不可见规则，
           标记让它变成可感知的规则。行高常驻，提交中只清空文字不塌陷，
           避免释义在按下「下一词」的瞬间跳动。 -->
      <p
        v-if="isRevealed"
        class="min-h-4 text-center text-xs text-muted-foreground"
      >
        {{ submissionStarted ? '' : '尚未记录 · 点「下一词」提交' }}
      </p>
      <p
        v-if="error"
        role="alert"
        class="text-sm wrap-anywhere text-danger-text"
      >
        {{ error }}
      </p>
    </div>
    <div class="flex w-full flex-col items-center gap-3">
      <!-- 下一词保留右列位置，修正入口消失时也不移动触摸目标。 -->
      <div class="grid w-full max-w-72 grid-cols-2 gap-4">
        <template v-if="mode === 'recalling'">
          <Button variant="outline" @click="$emit('choose', 'forgotten')">
            <XCircle class="size-4 shrink-0" aria-hidden="true" />没记住
          </Button>
          <Button @click="$emit('choose', 'remembered')">
            <CheckCircle class="size-4 shrink-0" aria-hidden="true" />记得
          </Button>
        </template>
        <template v-else-if="mode === 'revealed'">
          <Button
            v-if="canCorrect"
            variant="outline"
            :disabled="submitting"
            @click="$emit('correct')"
          >
            <XCircle class="size-4 shrink-0" aria-hidden="true" />没记住
          </Button>
          <Button
            class="col-start-2"
            :pending="submitting"
            @click="$emit('next')"
            >下一词</Button
          >
        </template>
      </div>
      <!-- 仅按主输入能力展示提示；触摸设备外接键盘仍可使用页面快捷键。 -->
      <p
        class="hidden min-h-8 text-center text-xs text-muted-foreground pointer-fine:[@media(hover:hover)]:block"
      >
        <template v-if="mode === 'recalling'"
          >1 / ← 没记住 · 2 / → 记得</template
        >
        <template v-else>
          <span v-if="canCorrect">1 / ← 修正为没记住 · </span>Space / Enter
          下一词
        </template>
      </p>
    </div>
  </div>
</template>
