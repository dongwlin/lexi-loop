// @lexi-loop/api-client 公共出口。
// 请求函数与 DTO 类型由 Orval 从 docs/openapi/ 生成（`pnpm -F @lexi-loop/api-client generate`，
// 输入经 scripts/unwrap-envelope.mjs 剥掉 Envelope 外壳），不手工修改；
// transport / 错误模型 / mutator（统一解包、Bearer 注入）为手写统一层（前端技术栈 §7.2）。
// 包不读取宿主应用的环境变量；Base URL 与 access token 由应用侧 configureApiClient 注入。

export { ApiError, isApiError, isAbortError } from './errors'
export type { ApiErrorKind, FieldError } from './errors'
export { configureApiClient } from './mutator'
export type { ApiClientConfig } from './mutator'

export {
  abandonReviewSession,
  deleteWord,
  getReviewSession,
  getVersion,
  getWord,
  importWords,
  listWords,
  startReviewSession,
  submitReviewResult,
  updateWordReviewMeaning,
} from './generated/client'
export type * from './generated/model'
