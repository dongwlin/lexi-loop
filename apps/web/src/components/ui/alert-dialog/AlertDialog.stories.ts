import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { ref } from 'vue'
import { expect, within } from 'storybook/test'
import Button from '../button/Button.vue'
import AlertDialog from './AlertDialog.vue'
import AlertDialogAction from './AlertDialogAction.vue'
import AlertDialogCancel from './AlertDialogCancel.vue'

const meta = {
  title: 'UI/AlertDialog',
  component: AlertDialog,
} satisfies Meta<typeof AlertDialog>

export default meta
type Story = StoryObj<typeof meta>

// 不可逆确认（交互与可访问性规范 §9.4）：危险按钮用具体动作名称，Cancel 在前让初始焦点落
// 在安全操作上。确认与取消的语义角色（alertdialog + 两个 button）在 play 中验证。
export const Default: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button, AlertDialog, AlertDialogAction, AlertDialogCancel },
    setup: () => {
      const open = ref(false)
      return { open }
    },
    template: `
      <AlertDialog
        v-model:open="open"
        title="删除生词"
        description="将删除单词「ambiguous」。删除后它不再出现在生词库与复习中；重新导入同一单词可恢复记录与遇词次数。"
      >
        <template #trigger>
          <Button variant="danger-soft">删除生词</Button>
        </template>
        <template #footer>
          <AlertDialogCancel as-child>
            <Button variant="outline">取消</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button variant="danger">删除生词</Button>
          </AlertDialogAction>
        </template>
      </AlertDialog>
    `,
  }),
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: '删除生词' }))
    const dialog = await within(document.body).findByRole('alertdialog', {
      name: '删除生词',
    })
    await expect(dialog).toBeVisible()
    await userEvent.click(within(dialog).getByRole('button', { name: '取消' }))
    await expect(
      within(document.body).queryByRole('alertdialog', { name: '删除生词' }),
    ).toBeNull()
  },
}
