import type { Meta, StoryObj } from '@storybook/vue3-vite'
import ReviewFlashcard from './ReviewFlashcard.vue'

const meta = {
  title: 'Review/Flashcard',
  component: ReviewFlashcard,
  args: {
    autoPlayPronunciation: false,
    word: 'ambiguous',
    phonetic: '/æmˈbɪɡjuəs/',
    meaning: 'adj. 模棱两可的；含糊不清的；有歧义的',
    mode: 'recalling',
    canCorrect: false,
    submitting: false,
    submissionStarted: false,
    error: null,
  },
  decorators: [
    () => ({
      template:
        '<div class="flex max-w-xl flex-col items-center gap-6 text-center"><story /></div>',
    }),
  ],
} satisfies Meta<typeof ReviewFlashcard>
export default meta
type Story = StoryObj<typeof meta>

export const Recalling: Story = {}
// 揭示阶段未提交时内容槽下方常驻「尚未记录 · 点「下一词」提交」：
// Remembered / Forgotten 钉住这条暂定标记，Submitting / Retry 钉住它让位给提交中与错误的状态。
export const Remembered: Story = {
  args: { mode: 'revealed', canCorrect: true },
}
export const Forgotten: Story = { args: { mode: 'revealed' } }
export const Submitting: Story = {
  args: { mode: 'revealed', submitting: true, submissionStarted: true },
}
export const Retry: Story = {
  args: {
    mode: 'revealed',
    submissionStarted: true,
    error: '提交失败，请点击「下一词」重试。',
  },
}
export const Multiline: Story = {
  args: {
    mode: 'revealed',
    canCorrect: true,
    meaning: 'adj. 模棱两可的\n含糊不清的',
  },
}
export const LongContent: Story = {
  args: {
    mode: 'revealed',
    word: 'pneumonoultramicroscopicsilicovolcanoconiosis',
    // 音标必须与当前单词一致：长词场景要连同同样冗长的音标一起压测换行，
    // 沿用 ambiguous 的音标会让这个 Story 声称覆盖的长词排版失真。
    phonetic: '/ˌnjuːmənoʊˌʌltrəˌmaɪkrəˌskɒpɪkˌsɪlɪkoʊvɒlˌkeɪnoʊˌkoʊniˈoʊsɪs/',
    meaning: Array.from(
      { length: 12 },
      (_, i) =>
        `释义 ${i + 1}：很长的释义应完整展示，并允许按容器宽度自动换行。`,
    ).join('\n'),
  },
}

export const WithoutPhonetic: Story = { args: { phonetic: '' } }

// Story 中模拟语音边界，避免工作台或 CI 调用真实系统音源。
export const AutoPlay: Story = {
  args: { autoPlayPronunciation: true },
  beforeEach: () => {
    const descriptor = Object.getOwnPropertyDescriptor(
      window,
      'speechSynthesis',
    )
    Object.defineProperty(window, 'speechSynthesis', {
      configurable: true,
      value: { speak: () => {}, cancel: () => {} },
    })
    return () => {
      if (descriptor)
        Object.defineProperty(window, 'speechSynthesis', descriptor)
      else Reflect.deleteProperty(window, 'speechSynthesis')
    }
  },
}
