import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { ref } from 'vue'
import { expect, within } from 'storybook/test'
import Button from '../button/Button.vue'
import Dialog from './Dialog.vue'
import DialogClose from './DialogClose.vue'

const meta = {
  title: 'UI/Dialog',
  component: Dialog,
} satisfies Meta<typeof Dialog>

export default meta
type Story = StoryObj<typeof meta>

// 受控开闭（触发器 → Esc 关闭）在 play 中验证。对话框挂在 body 的 Portal 上，断言需从
// document 查询（不局限于 canvas 容器）。
export const Default: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button, Dialog, DialogClose },
    setup: () => {
      const open = ref(false)
      return { open }
    },
    template: `
      <Dialog
        v-model:open="open"
        title="编辑复习释义"
        description="修改只影响你自己的复习展示，不会改动词典数据。"
      >
        <template #trigger>
          <Button variant="secondary">编辑复习释义</Button>
        </template>
        <label for="story-meaning" class="block text-sm text-muted-foreground">自定义复习释义</label>
        <textarea
          id="story-meaning"
          rows="3"
          class="mt-2 w-full rounded-field border border-field-border bg-field px-3 py-2 text-sm text-foreground shadow-field placeholder:text-muted-foreground"
          placeholder="adj. 模棱两可的；含糊不清的"
        />
        <template #footer>
          <DialogClose as-child>
            <Button variant="outline">取消</Button>
          </DialogClose>
          <Button @click="open = false">保存修改</Button>
        </template>
      </Dialog>
    `,
  }),
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: '编辑复习释义' }))
    const dialog = await within(document.body).findByRole('dialog', {
      name: '编辑复习释义',
    })
    await expect(dialog).toBeVisible()
    await userEvent.click(within(dialog).getByRole('button', { name: '取消' }))
    await expect(
      within(document.body).queryByRole('dialog', { name: '编辑复习释义' }),
    ).toBeNull()
  },
}
