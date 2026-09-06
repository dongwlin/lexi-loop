// 全局测试设施（前端测试规范 §15）：安装 jest-dom DOM Matcher、统一清理挂载的组件。
import '@testing-library/jest-dom/vitest'

import { cleanup } from '@testing-library/vue'

import { afterEach } from 'vitest'

afterEach(() => {
  cleanup()
})
