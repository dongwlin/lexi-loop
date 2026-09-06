// 应用侧 API 装配（前端应用架构规范 §8.1）：为生成的 @lexi-loop/api-client
// 注入 Base URL 与认证注入点。包本身不读取 import.meta.env；请求函数与 DTO
// 类型由 Feature 层直接从 '@lexi-loop/api-client' 导入使用。
import { configureApiClient } from '@lexi-loop/api-client'
import { env } from './env'

export { ApiError, isAbortError, isApiError } from '@lexi-loop/api-client'
export type { ApiErrorKind } from '@lexi-loop/api-client'

configureApiClient({
  baseUrl: env.apiBaseUrl,
  // 认证落地后在此注入 getAccessToken（前端 API 与认证集成规范 §4.2 / §5.1）。
})
