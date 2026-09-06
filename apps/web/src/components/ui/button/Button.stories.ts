import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn } from 'storybook/test'
import Button from './Button.vue'

const meta = {
  title: 'UI/Button',
  component: Button,
} satisfies Meta<typeof Button>

export default meta
type Story = StoryObj<typeof meta>

// click 经 attrs 透传到原生 button（组件未声明 emits，不进 Meta args 类型），用模块级 spy 承接。
const onClick = fn()

export const Default: Story = {
  render: () => ({
    components: { Button },
    setup: () => ({ onClick }),
    template: '<Button @click="onClick">保存修改</Button>',
  }),
  // 核心交互（点击透传）在 play 中验证，供 Interactions 面板调试（前端测试规范 §10.4）。
  // 先清空调用记录，保证 Story 可重复执行。
  play: async ({ canvas, userEvent }) => {
    onClick.mockClear()
    await userEvent.click(canvas.getByRole('button', { name: '保存修改' }))
    await expect(onClick).toHaveBeenCalledOnce()
  },
}

export const Variants: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button },
    template: `
      <div class="flex max-w-xl flex-wrap items-center gap-3">
        <Button>保存修改</Button>
        <Button variant="secondary">保存修改</Button>
        <Button variant="tertiary">保存修改</Button>
        <Button variant="outline">保存修改</Button>
        <Button variant="ghost">保存修改</Button>
        <Button variant="danger">删除项目</Button>
        <Button variant="danger-soft">删除生词</Button>
      </div>
    `,
  }),
}

export const Disabled: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button },
    template: `
      <div class="flex flex-wrap items-center gap-3">
        <Button disabled>保存修改</Button>
        <Button variant="danger-soft" disabled>删除生词</Button>
      </div>
    `,
  }),
}

export const Pending: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button },
    template: '<Button pending>保存修改</Button>',
  }),
}

export const LongContent: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button },
    template: `
      <div class="flex max-w-xs flex-wrap items-center gap-3">
        <Button>长文案按钮：删除这条生词并同步复习计划</Button>
        <Button variant="outline">长文案按钮：删除这条生词并同步复习计划</Button>
      </div>
    `,
  }),
}

// 主题类挂在 html（设计系统方案 §11）；.dark 包裹块会让其子树重新解析为深色语义值，
// 用于不引入主题切换插件时核对深色配方。
export const DarkTheme: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button },
    template: `
      <div class="dark flex max-w-xl flex-wrap items-center gap-3 rounded-card bg-background p-4">
        <Button>保存修改</Button>
        <Button variant="secondary">保存修改</Button>
        <Button variant="outline">保存修改</Button>
        <Button variant="ghost">保存修改</Button>
        <Button variant="danger-soft">删除生词</Button>
        <Button pending>保存修改</Button>
        <Button disabled>保存修改</Button>
      </div>
    `,
  }),
}
