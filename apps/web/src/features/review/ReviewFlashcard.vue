<script setup lang="ts">
import { CheckCircle, XCircle } from 'lucide-vue-next'
import ReviewPronunciation from './ReviewPronunciation.vue'
import { Button } from '@/components/ui'
import type { ReviewMode, ReviewResult } from '@/utils/review-state-machine'

// 页面负责会话、提交与焦点；闪卡仅展示当前阶段并发出用户意图。
defineProps<{
  phonetic?: string
  word: string
  meaning: string
  mode: ReviewMode
  canCorrect: boolean
  submitting: boolean
  error: string | null
}>()
defineEmits<{
  choose: [result: ReviewResult]
  correct: []
  next: []
}>()
</script>

<template>
  <div class="flex w-full min-w-0 flex-col items-center gap-6">
    <div class="flex max-w-full flex-col items-center gap-1">
      <h2
        class="max-w-full text-3xl font-semibold wrap-anywhere text-foreground"
      >
        {{ word }}
      </h2>
      <ReviewPronunciation :word="word" :phonetic="phonetic" />
    </div>
    <!-- 共用内容槽吸收常规释义行数差异；超长内容自然撑开，不隐藏待回忆的释义。 -->
    <div
      class="flex min-h-24 w-full flex-col items-center justify-center gap-3 sm:min-h-32"
    >
      <p v-if="mode === 'recalling'" class="text-sm text-muted-foreground">
        先回忆这个单词的意思
      </p>
      <template v-else-if="mode === 'revealed'">
        <p
          class="max-w-full text-base wrap-anywhere whitespace-pre-line text-foreground"
        >
          {{ meaning }}
        </p>
        <p
          v-if="error"
          role="alert"
          class="text-sm wrap-anywhere text-danger-text"
        >
          {{ error }}
        </p>
      </template>
    </div>
    <div class="flex w-full flex-col items-center gap-3">
      <!-- 下一词保留右列位置，修正入口消失时也不移动触摸目标。 -->
      <div class="grid w-full max-w-72 grid-cols-2 gap-4">
        <template v-if="mode === 'recalling'">
          <Button variant="outline" @click="$emit('choose', 'forgotten')">
            <XCircle class="size-4 shrink-0" aria-hidden="true" />不认识
          </Button>
          <Button @click="$emit('choose', 'remembered')">
            <CheckCircle class="size-4 shrink-0" aria-hidden="true" />认识
          </Button>
        </template>
        <template v-else-if="mode === 'revealed'">
          <Button
            v-if="canCorrect"
            variant="outline"
            :disabled="submitting"
            @click="$emit('correct')"
          >
            <XCircle class="size-4 shrink-0" aria-hidden="true" />不认识
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
          >1 / ← 不认识 · 2 / → 认识</template
        >
        <template v-else>
          <span v-if="canCorrect">1 / ← 修正为不认识 · </span>Space / Enter
          下一词
        </template>
      </p>
    </div>
  </div>
</template>
