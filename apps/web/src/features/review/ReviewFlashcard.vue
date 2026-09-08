<script setup lang="ts">
import { CheckCircle, XCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui'
import type { ReviewMode, ReviewResult } from '@/utils/review-state-machine'

// 页面负责会话、提交与焦点；闪卡仅展示当前阶段并发出用户意图。
defineProps<{
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
  <h2 class="text-3xl font-semibold break-words text-foreground">{{ word }}</h2>
  <template v-if="mode === 'recalling'">
    <p class="text-sm text-muted-foreground">先回忆这个单词的意思</p>
    <div class="mt-4 flex gap-4">
      <Button variant="outline" @click="$emit('choose', 'forgotten')">
        <XCircle class="size-4 shrink-0" aria-hidden="true" />不认识
      </Button>
      <Button @click="$emit('choose', 'remembered')">
        <CheckCircle class="size-4 shrink-0" aria-hidden="true" />认识
      </Button>
    </div>
  </template>
  <template v-else-if="mode === 'revealed'">
    <p class="max-w-full text-base whitespace-pre-line text-foreground">
      {{ meaning }}
    </p>
    <p v-if="error" role="alert" class="text-sm text-danger-text">
      {{ error }}
    </p>
    <div class="mt-4 flex gap-4">
      <Button
        v-if="canCorrect"
        variant="outline"
        :disabled="submitting"
        @click="$emit('correct')"
      >
        <XCircle class="size-4 shrink-0" aria-hidden="true" />不认识
      </Button>
      <Button :pending="submitting" @click="$emit('next')">下一词</Button>
    </div>
  </template>
  <p class="mt-2 text-center text-xs text-muted-foreground">
    <template v-if="mode === 'recalling'">1 / ← 不认识 · 2 / → 认识</template>
    <template v-else>
      <span v-if="canCorrect">1 / ← 修正为不认识 · </span>Space / Enter 下一词
    </template>
  </p>
</template>
