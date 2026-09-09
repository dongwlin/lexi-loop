<script setup lang="ts">
import { onBeforeUnmount, onMounted, shallowRef, useId, watch } from 'vue'
import { Volume2 } from 'lucide-vue-next'
import { Button } from '@/components/ui'

const props = withDefaults(
  defineProps<{
    word: string
    phonetic?: string
    autoPlayPronunciation?: boolean
  }>(),
  { autoPlayPronunciation: true, phonetic: '' },
)
const statusId = useId()
const message = shallowRef('')
let active: SpeechSynthesisUtterance | null = null
let timeout: ReturnType<typeof setTimeout> | undefined

function stop() {
  clearTimeout(timeout)
  if (!active) return
  active.onend = null
  active.onerror = null
  active = null
  // 某些环境暴露 API 却无法使用语音服务；清理不能阻断切词或卸载。
  try {
    window.speechSynthesis.cancel()
  } catch {
    /* 无可清理的语音服务。 */
  }
}

function play() {
  stop()
  message.value = ''
  if (
    typeof window === 'undefined' ||
    !window.speechSynthesis ||
    typeof window.SpeechSynthesisUtterance !== 'function'
  ) {
    message.value = '当前浏览器不支持发音，可继续复习。'
    return
  }
  try {
    const utterance = new SpeechSynthesisUtterance(props.word)
    utterance.lang = 'en-US'
    active = utterance
    utterance.onend = () => {
      if (active !== utterance) return
      clearTimeout(timeout)
      active = null
      message.value = ''
    }
    utterance.onerror = () => {
      if (active !== utterance) return
      stop()
      message.value = '发音播放失败，请重试或继续复习。'
    }
    // 为不发送结束或错误事件的语音引擎提供有界降级。
    timeout = setTimeout(() => {
      stop()
      message.value = '发音播放超时，请重试或继续复习。'
    }, 15000)
    message.value = `正在播放 ${props.word} 的发音`
    window.speechSynthesis.speak(utterance)
  } catch {
    stop()
    message.value = '发音播放失败，请重试或继续复习。'
  }
}

watch(
  () => props.word,
  () => {
    stop()
    message.value = ''
    if (props.autoPlayPronunciation) play()
  },
  // 等待本次 props 批量更新完成，读取同一次更新中的最新设置。
  { flush: 'post' },
)
onMounted(() => {
  if (props.autoPlayPronunciation) play()
})
onBeforeUnmount(stop)
</script>

<template>
  <div class="flex max-w-full flex-col items-center">
    <div class="flex max-w-full items-center gap-1">
      <span
        v-if="phonetic?.trim()"
        class="text-sm wrap-anywhere text-muted-foreground"
        >{{ phonetic }}</span
      >
      <Button
        variant="ghost"
        class="min-w-11 shrink-0"
        :aria-label="`播放 ${word} 的发音`"
        :aria-describedby="statusId"
        @click="play"
      >
        <Volume2 class="size-4" aria-hidden="true" />
      </Button>
    </div>
    <p
      :id="statusId"
      role="status"
      class="min-h-4 max-w-full text-xs wrap-anywhere text-muted-foreground"
    >
      {{ message }}
    </p>
  </div>
</template>
