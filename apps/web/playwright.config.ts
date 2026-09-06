/*
 * Playwright E2E（前端技术栈 §10 / 前端测试规范 §16）：从本地预览的生产构建入口
 * 验证关键用户流程；与 Vitest 测试的分工见测试规范 §12。需要后端联调的流程由
 * 环境自行启动 Go 服务（/api 代理与开发环境一致）。
 */
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:4173',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    // E2E 面向生产构建（先 build 再 preview），不面向 dev server。
    // --host 127.0.0.1：vite preview 默认只绑 localhost（本机解析为 IPv6）。
    command:
      'pnpm build && pnpm preview --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
})
