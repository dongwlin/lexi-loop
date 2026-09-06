// Orval 自定义 mutator：生成代码的全部请求都经由这里发出
// （前端技术栈 §7.2——生成客户端必须经过统一 Transport/API Client）。
// 认证刷新与 401 重放（前端 API 与认证集成规范 §10）在此接入，不另设拦截器。

import { ApiError, toFieldErrorMap } from './errors'
import type { FieldError } from './errors'
import { sendRequest } from './transport'
import type { TransportResult } from './transport'

export interface ApiClientConfig {
  /** API Base URL（缺省同源相对路径）。由宿主应用从 env 注入，包不读取 import.meta.env。 */
  baseUrl?: string
  /** access token 窄接口，认证落地后由 Session Manager 注入（规范 §4.2）。 */
  getAccessToken?: () => string | null | Promise<string | null>
  /** 包级默认超时（毫秒）。缺省不超时；单个请求可用 signal 自行取消。 */
  timeoutMs?: number
}

let config: ApiClientConfig = {}

/** 应用启动时调用一次（apps/web 在 `src/lib/api.ts` 装配）；重复调用以最后一次为准。 */
export function configureApiClient(next: ApiClientConfig): void {
  config = next
}

/**
 * 生成的请求函数统一调用的 mutator：Base URL 前缀、Bearer 注入、
 * `{code, message, data}` 解包与 ApiError 转换。
 */
export async function customFetch<TData>(url: string, options: RequestInit = {}): Promise<TData> {
  const headers = new Headers(options.headers)
  headers.set('Accept', 'application/json')
  const token = (await config.getAccessToken?.()) ?? null
  if (token !== null) headers.set('Authorization', `Bearer ${token}`)

  const result = await sendRequest({
    method: options.method ?? 'GET',
    url: `${config.baseUrl ?? ''}${url}`,
    headers,
    body: options.body ?? undefined,
    signal: options.signal ?? undefined,
    timeoutMs: config.timeoutMs,
  })

  return unwrap<TData>(result)
}

function unwrap<TData>(result: TransportResult): TData {
  const { status, json, requestId, retryAfterSeconds } = result

  // 204 仅在具体端点契约允许时出现；无响应体直接返回 undefined（规范 §5.2）。
  if (status === 204) return undefined as TData

  const envelope = readEnvelope(json)
  if (envelope === null) {
    throw new ApiError('protocol', '响应不是合法的 API 顶层结构', { httpStatus: status, requestId, retryAfterSeconds, cause: json })
  }

  if (envelope.code === 'OK') {
    return envelope.data as TData
  }

  throw new ApiError('http', envelope.message, {
    httpStatus: status,
    code: envelope.code,
    data: envelope.data,
    fieldErrors: extractFieldErrors(envelope.data),
    requestId,
    retryAfterSeconds,
  })
}

interface EnvelopeFields {
  code: string
  message: string
  data: unknown
}

function readEnvelope(json: unknown): EnvelopeFields | null {
  if (typeof json !== 'object' || json === null) return null
  const record = json as Record<string, unknown>
  if (typeof record.code !== 'string' || typeof record.message !== 'string' || !('data' in record)) return null
  return { code: record.code, message: record.message, data: record.data }
}

function extractFieldErrors(data: unknown): Record<string, string[]> | undefined {
  if (typeof data !== 'object' || data === null) return undefined
  const fieldErrors = (data as Record<string, unknown>).fieldErrors
  if (!Array.isArray(fieldErrors)) return undefined
  return toFieldErrorMap(fieldErrors as FieldError[])
}
