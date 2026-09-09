import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { ref } from 'vue'
import { expect, within } from 'storybook/test'
import ReviewSettings from './ReviewSettings.vue'

const meta = {
  title: 'Review/Settings',
  component: ReviewSettings,
} satisfies Meta<typeof ReviewSettings>
export default meta

export const Default: StoryObj = {
  render: () => ({
    components: { ReviewSettings },
    setup: () => ({ enabled: ref(true) }),
    template: '<ReviewSettings v-model="enabled" />',
  }),
  play: async ({ canvas, userEvent }) => {
    const trigger = canvas.getByRole('button', { name: '复习设置' })
    await userEvent.click(trigger)
    const dialog = await within(document.body).findByRole('dialog', {
      name: '复习设置',
    })
    const checkbox = within(dialog).getByRole('checkbox', {
      name: '自动播放单词发音',
    })
    await expect(checkbox).toBeChecked()
    await userEvent.click(checkbox)
    await expect(checkbox).not.toBeChecked()
    await userEvent.keyboard('{Escape}')
    await expect(trigger).toHaveFocus()
  },
}
