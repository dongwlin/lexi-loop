import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { createPinia, setActivePinia } from 'pinia'
import { expect, within } from 'storybook/test'
import { useThemeStore, type ThemePreference } from '@/stores/theme'
import ThemeSwitcher from './ThemeSwitcher.vue'

const meta = {
  title: 'Layout/ThemeSwitcher',
  component: ThemeSwitcher,
} satisfies Meta<typeof ThemeSwitcher>

export default meta
type Story = StoryObj<typeof meta>

// 组件读取全局 theme store：每个 Story 建独立 Pinia 并显式设定偏好，
// 保证 Story 之间互不影响，也与浏览器实际偏好无关。
function renderWithPreference(preference: ThemePreference) {
  return () => ({
    components: { ThemeSwitcher },
    setup: () => {
      setActivePinia(createPinia())
      useThemeStore().setPreference(preference)
    },
    template: '<div class="flex min-h-40 items-start justify-end p-2"><ThemeSwitcher /></div>',
  })
}

export const Light: Story = {
  render: renderWithPreference('light'),
}

export const System: Story = {
  parameters: { controls: { disable: true } },
  render: renderWithPreference('system'),
}

// 打开菜单核对弹层配方（rounded-popover / bg-overlay / shadow-overlay）与选中指示。
export const DarkMenuOpen: Story = {
  parameters: { controls: { disable: true } },
  render: renderWithPreference('dark'),
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: '界面主题' }))
    // 菜单挂在 body 的 Portal 上，需从 document 查询（不局限于 canvas 容器）。
    const menu = within(document.body)
    await expect(menu.getByRole('menuitemradio', { name: '深色' })).toHaveAttribute(
      'aria-checked',
      'true',
    )
    await expect(menu.getByRole('menuitemradio', { name: '跟随系统' })).toHaveAttribute(
      'aria-checked',
      'false',
    )
  },
}

// 选择动作经 RadioGroup 写回 store：切换后菜单关闭、html 主题类同步更新。
export const SwitchToDark: Story = {
  parameters: { controls: { disable: true } },
  render: renderWithPreference('light'),
  play: async ({ canvas, userEvent }) => {
    await expect(document.documentElement.classList.contains('dark')).toBe(false)
    await userEvent.click(canvas.getByRole('button', { name: '界面主题' }))
    await userEvent.click(within(document.body).getByRole('menuitemradio', { name: '深色' }))
    await expect(document.documentElement.classList.contains('dark')).toBe(true)
    await expect(document.documentElement.classList.contains('light')).toBe(false)
  },
}
