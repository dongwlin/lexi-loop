/*
 * 测试配置独立于 vite.config.ts 并与其合并，复用框架插件、@/ 别名与转换配置
 * （前端测试规范 §15）。Vitest Projects：`web`（jsdom 单元 / 组件 / 集成）与
 * `storybook`（Story 即真实浏览器测试，配置形态来自 @storybook/addon-vitest
 * 的 vitest.config.4 模板）。两个 project 都 `extends: true` 继承根级 Vite
 * 插件与别名；jsdom 环境与全局设施只挂在 web project，避免泄漏进浏览器测试。
 */
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { playwright } from '@vitest/browser-playwright'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, mergeConfig } from 'vitest/config'

import viteConfig from './vite.config.ts'

const dirname = path.dirname(fileURLToPath(import.meta.url))

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
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
      projects: [
        {
          extends: true,
          test: {
            name: 'web',
            // 单元与组件测试默认 jsdom 环境；真实浏览器行为走 Storybook Vitest
            // addon 与 Playwright（前端测试规范 §2.2 / §12）。
            environment: 'jsdom',
            setupFiles: ['./tests/setup.ts'],
            include: ['src/**/*.test.{ts,tsx}', 'tests/**/*.test.{ts,tsx}'],
            clearMocks: true,
            restoreMocks: true,
            unstubEnvs: true,
            unstubGlobals: true,
            // 不用 retry 掩盖不稳定测试（前端测试规范 §8.3）。
            retry: 0,
            env: {
              // 测试环境显式指向保留假主机：node/jsdom 的 fetch 不接受相对路径，
              // MSW 在网络边界按绝对 URL 拦截，不会发出真实请求。
              VITE_API_BASE_URL: 'http://lexi-loop.test',
            },
          },
        },
        {
          extends: true,
          plugins: [
            // 插件会把 .storybook 配置下的 Stories 转换为浏览器测试
            // （内部自动注入 setup 文件，无需手写 vitest.setup.ts）。
            storybookTest({ configDir: path.join(dirname, '.storybook') }),
          ],
          test: {
            name: 'storybook',
            browser: {
              enabled: true,
              headless: true,
              provider: playwright({}),
              instances: [{ browser: 'chromium' }],
            },
          },
        },
      ],
    },
  }),
)
