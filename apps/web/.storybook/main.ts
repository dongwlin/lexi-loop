import type { StorybookConfig } from '@storybook/vue3-vite'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath } from 'node:url'

const srcAlias = {
  find: '@',
  replacement: fileURLToPath(new URL('../src', import.meta.url)),
}

const config: StorybookConfig = {
  framework: '@storybook/vue3-vite',
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  // 可访问性检查（addon-a11y）与 Vitest 集成（addon-vitest，Story 即浏览器测试）。
  addons: ['@storybook/addon-a11y', '@storybook/addon-vitest'],

  // 应用样式走 Tailwind CSS v4 的 Vite 插件（与 vite.config.ts 一致），
  // Storybook 的独立 Vite 实例需要显式接入，否则 main.css 编译不出 utilities。
  // @/ 别名与应用侧 vite.config.ts 保持一致，Story 内可直接引用 src 下的模块。
  viteFinal: (config) => ({
    ...config,
    plugins: [...(config.plugins ?? []), tailwindcss()],
    resolve: {
      ...config.resolve,
      alias: Array.isArray(config.resolve?.alias)
        ? [...config.resolve.alias, srcAlias]
        : { ...config.resolve?.alias, '@': srcAlias.replacement },
    },
  }),
}

export default config
