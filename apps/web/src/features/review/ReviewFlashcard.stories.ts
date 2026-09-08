import type { Meta, StoryObj } from '@storybook/vue3-vite'
import ReviewFlashcard from './ReviewFlashcard.vue'

const meta = {
  title: 'Review/Flashcard',
  component: ReviewFlashcard,
  args: {
    word: 'ambiguous',
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
