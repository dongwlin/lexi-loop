import type { Meta, StoryObj } from '@storybook/vue3-vite'
import ReviewFlashcard from './ReviewFlashcard.vue'

const meta = {
  title: 'Review/Flashcard',
  component: ReviewFlashcard,
  args: {
    word: 'ambiguous',
    phonetic: '/æmˈbɪɡjuəs/',
    meaning: 'adj. 模棱两可的；含糊不清的；有歧义的',
    mode: 'recalling',
    canCorrect: false,
    submitting: false,
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
export const Remembered: Story = {
  args: { mode: 'revealed', canCorrect: true },
}
export const Forgotten: Story = { args: { mode: 'revealed' } }
export const Submitting: Story = {
  args: { mode: 'revealed', submitting: true },
}
export const Retry: Story = {
  args: { mode: 'revealed', error: '提交失败，请点击「下一词」重试。' },
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
    meaning: Array.from(
      { length: 12 },
      (_, i) =>
        `释义 ${i + 1}：很长的释义应完整展示，并允许按容器宽度自动换行。`,
    ).join('\n'),
  },
}

export const WithoutPhonetic: Story = { args: { phonetic: '' } }
