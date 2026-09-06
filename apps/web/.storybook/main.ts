import type { StorybookConfig } from '@storybook/vue3-vite'
import tailwindcss from '@tailwindcss/vite'

const config: StorybookConfig = {
  framework: '@storybook/vue3-vite',
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  // 应用样式走 Tailwind CSS v4 的 Vite 插件（与 vite.config.ts 一致），
  // Storybook 的独立 Vite 实例需要显式接入，否则 main.css 编译不出 utilities。
  viteFinal: (config) => ({
    ...config,
    plugins: [...(config.plugins ?? []), tailwindcss()],
  }),
}

export default config
