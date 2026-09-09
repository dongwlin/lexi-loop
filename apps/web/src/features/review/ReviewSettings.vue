<script setup lang="ts">
import { shallowRef, useId } from 'vue'
import { Settings } from 'lucide-vue-next'
import { Button, Dialog } from '@/components/ui'

const autoPlayPronunciation = defineModel<boolean>({ required: true })
const open = shallowRef(false)
const checkboxId = useId()
</script>

<template>
  <Dialog
    v-model:open="open"
    title="复习设置"
    description="切换到新单词时自动播放英语发音"
  >
    <template #trigger>
      <Button variant="ghost" class="min-w-11 shrink-0" aria-label="复习设置">
        <Settings class="size-4" aria-hidden="true" />
      </Button>
    </template>
    <label
      :for="checkboxId"
      class="flex min-h-11 cursor-pointer items-center gap-3 text-sm text-foreground"
    >
      <input
        :id="checkboxId"
        v-model="autoPlayPronunciation"
        type="checkbox"
        data-dialog-initial-focus
        class="size-5 shrink-0 accent-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      />
      自动播放单词发音
    </label>
  </Dialog>
</template>
