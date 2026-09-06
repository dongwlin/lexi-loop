// HTTP 边界 Mock（前端测试规范 §8.1）：页面集成测试让生产请求链路真实执行到
// 网络边界，由 MSW 返回符合《HTTP API 设计规范》的响应。生命周期在 tests/setup.ts 统一管理。
import { setupServer } from 'msw/node'

import { handlers } from './handlers'

export const server = setupServer(...handlers)
