// 全局测试设施（前端测试规范 §15 / §8.3）：安装 jest-dom DOM Matcher、统一清理
// 挂载的组件、管理 MSW 生命周期——被修改的全局与 Handler 不跨用例泄漏。
import '@testing-library/jest-dom/vitest'

import { cleanup } from '@testing-library/vue'
import { afterAll, afterEach, beforeAll } from 'vitest'

import { server } from './mocks/server'

beforeAll(() =>
  server.listen({
    // 未声明 Handler 的请求直接报错，防止测试触达真实网络。
    onUnhandledRequest: 'error',
  }),
)

afterEach(() => {
  cleanup()
  server.resetHandlers()
})

afterAll(() => server.close())
