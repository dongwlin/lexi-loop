// fetch 薄封装：发送请求、解析 Header 与响应体、支持 Abort（前端 API 与认证集成规范 §3）。
// 不负责认证注入、响应解包与业务错误展示。

import { ApiError } from './errors'

export interface TransportOptions {
  method: string
  url: string
  headers?: HeadersInit
  body?: BodyInit
  signal?: AbortSignal
  timeoutMs?: number
}

export interface TransportResult {
  status: number
  headers: Headers
  /** 响应体 JSON 解析结果；空响应体（如 204）为 undefined。 */
  json: unknown
  requestId?: string
  retryAfterSeconds?: number
}

export async function sendRequest(options: TransportOptions): Promise<TransportResult> {
  const { method, url, headers, body, signal, timeoutMs } = options

  if (signal?.aborted) {
    throw new ApiError('abort', '请求已取消', { cause: signal.reason })
  }

  const composed = composeSignal(signal, timeoutMs)

  let response: Response
  try {
    response = await fetch(url, {
      method,
      headers,
      body,
      signal: composed,
      credentials: 'omit',
    })
  } catch (error) {
    throw classifyFetchError(error, signal, composed)
  }

  const requestId = response.headers.get('X-Request-Id') ?? undefined
  const retryAfterSeconds = parseRetryAfter(response.headers.get('Retry-After'))
  const json = await parseJsonBody(response, requestId, retryAfterSeconds)

  return { status: response.status, headers: response.headers, json, requestId, retryAfterSeconds }
}

/** 组合调用方 AbortSignal 与超时信号；两者都缺省时不设信号。 */
function composeSignal(userSignal: AbortSignal | undefined, timeoutMs: number | undefined): AbortSignal | undefined {
  const signals: AbortSignal[] = []
  if (userSignal !== undefined) signals.push(userSignal)
  if (timeoutMs !== undefined) signals.push(AbortSignal.timeout(timeoutMs))
  const [first] = signals
  if (first === undefined) return undefined
  return signals.length === 1 ? first : AbortSignal.any(signals)
}

function classifyFetchError(error: unknown, userSignal: AbortSignal | undefined, composed: AbortSignal | undefined): ApiError {
  if (userSignal?.aborted) {
    return new ApiError('abort', '请求已取消', { cause: error })
  }
  // AbortSignal.timeout 中止时 reason 为 TimeoutError，与调用方取消（AbortError）区分（规范 §5.3）。
  if ((composed?.reason as { name?: string } | undefined)?.name === 'TimeoutError') {
    return new ApiError('timeout', '请求超时', { cause: error })
  }
  if (error instanceof DOMException && error.name === 'AbortError') {
    return new ApiError('abort', '请求已取消', { cause: error })
  }
  return new ApiError('network', '网络连接失败', { cause: error })
}

async function parseJsonBody(response: Response, requestId: string | undefined, retryAfterSeconds: number | undefined): Promise<unknown> {
  const text = await response.text()
  if (text === '') return undefined
  try {
    return JSON.parse(text) as unknown
  } catch (error) {
    // 非 JSON 响应转为 ProtocolError，不伪装成业务错误（规范 §5.2）。
    throw new ApiError('protocol', '响应不是合法 JSON', { httpStatus: response.status, requestId, retryAfterSeconds, cause: error })
  }
}

function parseRetryAfter(value: string | null): number | undefined {
  if (value === null) return undefined
  const seconds = Number.parseInt(value, 10)
  return Number.isNaN(seconds) || seconds < 0 ? undefined : seconds
}
