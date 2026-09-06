// 统一错误类型与协议错误转换（前端 API 与认证集成规范 §5.3）。
// 只负责分类，不决定页面文案、不弹 Toast。

export type ApiErrorKind = 'http' | 'network' | 'timeout' | 'abort' | 'protocol' | 'storage'

/** 服务端 `data.fieldErrors` 的线格式（HTTP API 设计规范 §3.2）。 */
export interface FieldError {
  field: string
  reason: string
}

export interface ApiErrorOptions {
  httpStatus?: number
  code?: string
  data?: unknown
  fieldErrors?: Record<string, string[]>
  requestId?: string
  retryAfterSeconds?: number
  cause?: unknown
}

export class ApiError extends Error {
  readonly kind: ApiErrorKind
  readonly httpStatus?: number
  /** 稳定业务 code（如 BASE.NOT_FOUND.USER）；程序分支只判断它，不解析 message。 */
  readonly code?: string
  readonly data?: unknown
  readonly fieldErrors?: Record<string, string[]>
  readonly requestId?: string
  readonly retryAfterSeconds?: number

  constructor(kind: ApiErrorKind, message: string, options: ApiErrorOptions = {}) {
    super(message, { cause: options.cause })
    this.name = 'ApiError'
    this.kind = kind
    this.httpStatus = options.httpStatus
    this.code = options.code
    this.data = options.data
    this.fieldErrors = options.fieldErrors
    this.requestId = options.requestId
    this.retryAfterSeconds = options.retryAfterSeconds
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}

/** Abort 是预期控制流：UI 侧用它跳过错误展示与上报（规范 §5.3 / §5.4）。 */
export function isAbortError(error: unknown): boolean {
  return isApiError(error) && error.kind === 'abort'
}

/** 把服务端的 `fieldErrors: [{field, reason}]` 聚合为按字段分组的映射。 */
export function toFieldErrorMap(fieldErrors: readonly FieldError[] | null | undefined): Record<string, string[]> {
  const map: Record<string, string[]> = {}
  for (const item of fieldErrors ?? []) {
    ;(map[item.field] ??= []).push(item.reason)
  }
  return map
}
