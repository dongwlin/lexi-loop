/*
 * 测试配置独立于 vite.config.ts 并与其合并，复用框架插件、@/ 别名与转换配置
 * （前端测试规范 §15）。
 */
import { defineConfig, mergeConfig } from 'vitest/config'

import viteConfig from './vite.config.ts'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      // 单元与组件测试默认 jsdom 环境；真实浏览器行为走 Storybook Vitest addon
      // 与 Playwright（前端测试规范 §2.2 / §12）。
      environment: 'jsdom',
      setupFiles: ['./tests/setup.ts'],
      include: ['src/**/*.test.{ts,tsx}', 'tests/**/*.test.{ts,tsx}'],
      clearMocks: true,
      restoreMocks: true,
      unstubEnvs: true,
      unstubGlobals: true,
      // 不用 retry 掩盖不稳定测试（前端测试规范 §8.3）。
      retry: 0,
      coverage: {
        provider: 'v8',
        reporter: ['text', 'html'],
        include: ['src/**/*.{ts,tsx,vue}'],
        exclude: [
          'src/**/*.d.ts',
          'src/**/*.test.{ts,tsx}',
          'src/**/*.stories.{ts,tsx}',
          'src/**/generated/**',
        ],
      },
    },
  }),
)
