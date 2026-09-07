import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, within } from 'storybook/test'
import { Button } from '@/components/ui'
import Popover from './Popover.vue'

const meta = {
  title: 'UI/Popover',
  component: Popover,
} satisfies Meta<typeof Popover>

export default meta
type Story = StoryObj<typeof meta>

// 内容区关闭动作经模块级 spy 承接（弹层挂 body 的 Portal，断言从 document
// 查询，不局限于 canvas 容器）。
const onClose = fn()

export const Default: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button, Popover },
    setup: () => ({ onClose }),
    template: `
      <Popover>
        <template #trigger>
          <Button variant="outline">导入详情</Button>
        </template>
        <div class="w-64 space-y-2">
          <p class="text-sm font-medium">词典导入中</p>
          <p class="text-sm text-muted-foreground">已处理 123,400 / 770,611 行。</p>
          <Button variant="ghost" size="sm" @click="onClose">知道了</Button>
        </div>
      </Popover>
    `,
  }),
  // 核心交互（打开 → 点击关闭）在 play 中验证，供 Interactions 面板调试。
  play: async ({ canvas, userEvent }) => {
    onClose.mockClear()
    await userEvent.click(canvas.getByRole('button', { name: '导入详情' }))
    const layer = within(document.body)
    await userEvent.click(await layer.findByRole('button', { name: '知道了' }))
    await expect(onClose).toHaveBeenCalledOnce()
    // Popover 不因内部点击关闭（与 Menu 语义不同）：Escape 关闭后断言弹层卸载。
    await userEvent.keyboard('{Escape}')
    await expect(layer.queryByRole('button', { name: '知道了' })).toBeNull()
  },
}

export const OpenState: Story = {
  parameters: {
    controls: { disable: true },
    a11y: {
      // Story 以弹层打开收尾：Reka 弹层打开时给应用根节点标 aria-hidden
      // （焦点圈闭在弹层内），axe 静态分析识别不了焦点圈闭，对该 Story 属
      // 误报；按测试规范 §10.5 显式豁免此一条，其余规则仍为门禁。
      config: { rules: [{ id: 'aria-hidden-focus', enabled: false }] },
    },
  },
  render: () => ({
    components: { Button, Popover },
    template: `
      <Popover>
        <template #trigger>
          <Button variant="outline">导入详情</Button>
        </template>
        <div class="w-64 space-y-2">
          <p class="text-sm font-medium">词典导入中</p>
          <p class="text-sm text-muted-foreground">弹层配方：rounded-popover + bg-overlay + shadow-overlay。</p>
        </div>
      </Popover>
    `,
  }),
  // play 只打开弹层并断言内容：以打开态收尾，展示弹层配方的视觉。
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: '导入详情' }))
    const layer = within(document.body)
    await expect(await layer.findByText('词典导入中')).toBeVisible()
  },
}
