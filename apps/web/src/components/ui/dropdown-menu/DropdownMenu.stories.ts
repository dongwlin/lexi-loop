import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { ref } from 'vue'
import { DropdownMenuRadioGroup } from 'reka-ui'
import { Monitor, Moon, Sun } from 'lucide-vue-next'
import { expect, fn, within } from 'storybook/test'
import Button from '../button/Button.vue'
import DropdownMenu from './DropdownMenu.vue'
import DropdownMenuItem from './DropdownMenuItem.vue'
import DropdownMenuRadioItem from './DropdownMenuRadioItem.vue'

const meta = {
  title: 'UI/DropdownMenu',
  component: DropdownMenu,
} satisfies Meta<typeof DropdownMenu>

export default meta
type Story = StoryObj<typeof meta>

// select 经 attrs 透传到 Reka MenuItem（组件未声明 emits，不进 Meta args 类型），用模块级
// spy 承接。菜单挂在 body 的 Portal 上，断言需从 document 查询（不局限于 canvas 容器）。
const onSelect = fn()

export const Default: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: { Button, DropdownMenu, DropdownMenuItem },
    setup: () => ({ onSelect }),
    template: `
      <DropdownMenu>
        <template #trigger>
          <Button variant="outline">更多操作</Button>
        </template>
        <DropdownMenuItem @select="onSelect">重命名</DropdownMenuItem>
        <DropdownMenuItem @select="onSelect">移入其他词库</DropdownMenuItem>
      </DropdownMenu>
    `,
  }),
  // 核心交互（打开 → select 关闭菜单）在 play 中验证，供 Interactions 面板调试。
  // 先清空调用记录，保证 Story 可重复执行。
  play: async ({ canvas, userEvent }) => {
    onSelect.mockClear()
    await userEvent.click(canvas.getByRole('button', { name: '更多操作' }))
    const menu = within(document.body)
    await userEvent.click(await menu.findByRole('menuitem', { name: '重命名' }))
    await expect(onSelect).toHaveBeenCalledOnce()
    await expect(menu.queryByRole('menuitem', { name: '重命名' })).toBeNull()
  },
}

export const RadioGroup: Story = {
  parameters: { controls: { disable: true } },
  render: () => ({
    components: {
      Button,
      DropdownMenu,
      DropdownMenuRadioGroup,
      DropdownMenuRadioItem,
    },
    setup: () => {
      const preference = ref('system')
      const options = [
        { value: 'light', label: '浅色', icon: Sun },
        { value: 'dark', label: '深色', icon: Moon },
        { value: 'system', label: '跟随系统', icon: Monitor },
      ]
      return { preference, options }
    },
    template: `
      <DropdownMenu>
        <template #trigger>
          <Button variant="outline">界面主题</Button>
        </template>
        <DropdownMenuRadioGroup v-model="preference">
          <DropdownMenuRadioItem v-for="option in options" :key="option.value" :value="option.value">
            <component
              :is="option.icon"
              class="size-4 shrink-0 text-muted-foreground group-data-[state=checked]/item:text-primary-text"
              aria-hidden="true"
            />
            <span>{{ option.label }}</span>
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenu>
    `,
  }),
  // 选中态由语义表达：menuitemradio 的 aria-checked 随 RadioGroup 写回而更新。
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: '界面主题' }))
    const menu = within(document.body)
    await userEvent.click(
      await menu.findByRole('menuitemradio', { name: '深色' }),
    )
    await userEvent.click(canvas.getByRole('button', { name: '界面主题' }))
    await expect(
      await menu.findByRole('menuitemradio', { name: '深色' }),
    ).toHaveAttribute('aria-checked', 'true')
    await expect(
      menu.getByRole('menuitemradio', { name: '跟随系统' }),
    ).toHaveAttribute('aria-checked', 'false')
  },
}
