import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from './errors'
import { configureApiClient, customFetch } from './mutator'

interface RecordedCall {
  url: string
  init: RequestInit
}

let calls: RecordedCall[] = []

function stubFetch(handler: (url: string, init: RequestInit) => Response | Promise<Response>): void {
  calls = []
  vi.stubGlobal('fetch', (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    const record: RecordedCall = { url, init: init ?? {} }
    calls.push(record)
    return Promise.resolve(handler(url, record.init))
  })
}

/** 挂起的 fetch：仅在 signal 中止时以 reason 拒绝，用于超时 / 取消分类。 */
function stubHangingFetch(): void {
  calls = []
  vi.stubGlobal('fetch', (input: RequestInfo | URL, init?: RequestInit) => {
    const record: RecordedCall = { url: String(input), init: init ?? {} }
    calls.push(record)
    return new Promise<Response>((_resolve, reject) => {
      init?.signal?.addEventListener('abort', () => reject((init.signal as AbortSignal).reason))
    })
  })
}

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
}

function okEnvelope(data: unknown): unknown {
  return { code: 'OK', message: 'success', data }
}

async function captureRejection(promise: Promise<unknown>): Promise<ApiError> {
  try {
    await promise
  } catch (error) {
    return error as ApiError
  }
  throw new Error('expected the promise to reject')
}

afterEach(() => {
  vi.unstubAllGlobals()
  configureApiClient({})
})

describe('响应解包', () => {
  it('code=OK 时解包返回 data', async () => {
    stubFetch(() => jsonResponse(okEnvelope({ encounters: 4, created: 2, updated: 1 })))

    const data = await customFetch<{ created: number }>('/api/v1/words/import', {
      method: 'POST',
      body: JSON.stringify({ words: [] }),
    })

    expect(data).toEqual({ encounters: 4, created: 2, updated: 1 })
  })

  it('204 无响应体时返回 undefined', async () => {
    stubFetch(() => new Response(null, { status: 204 }))

    const data = await customFetch<unknown>('/api/v1/words')

    expect(data).toBeUndefined()
  })

  it('非 JSON 响应体转为 protocol 错误', async () => {
    stubFetch(() => new Response('<html>bad gateway</html>', { status: 502 }))

    const error = await captureRejection(customFetch('/api/v1/words'))

    expect(error).toBeInstanceOf(ApiError)
    expect(error.kind).toBe('protocol')
    expect(error.httpStatus).toBe(502)
  })

  it('顶层结构不合法转为 protocol 错误', async () => {
    stubFetch(() => jsonResponse({ foo: 'bar' }))

    const error = await captureRejection(customFetch('/api/v1/words'))

    expect(error.kind).toBe('protocol')
  })
})

describe('业务错误', () => {
  it('code≠OK 时抛出携带稳定字段的 ApiError', async () => {
    stubFetch(() =>
      jsonResponse(
        { code: 'BASE.NOT_FOUND.USER', message: 'word not found', data: {} },
        { status: 404, headers: { 'Content-Type': 'application/json', 'X-Request-Id': 'req-42' } },
      ),
    )

    const error = await captureRejection(customFetch('/api/v1/words/nope'))

    expect(error.kind).toBe('http')
    expect(error.code).toBe('BASE.NOT_FOUND.USER')
    expect(error.message).toBe('word not found')
    expect(error.httpStatus).toBe(404)
    expect(error.requestId).toBe('req-42')
  })

  it('data.fieldErrors 聚合为按字段分组的映射', async () => {
    stubFetch(() =>
      jsonResponse({
        code: 'BASE.PARAM.VALIDATION_FAILED',
        message: 'invalid parameter',
        data: {
          fieldErrors: [
            { field: 'words', reason: 'minItems 1' },
            { field: 'words', reason: 'expected array' },
          ],
        },
      }),
    )

    const error = await captureRejection(customFetch('/api/v1/words/import', { method: 'POST', body: '{}' }))

    expect(error.code).toBe('BASE.PARAM.VALIDATION_FAILED')
    expect(error.fieldErrors).toEqual({ words: ['minItems 1', 'expected array'] })
  })

  it('Retry-After 解析为秒数', async () => {
    stubFetch(() =>
      jsonResponse({ code: 'BASE.BIZ.RATE_LIMITED', message: 'rate limited', data: {} }, { status: 429, headers: { 'Retry-After': '30' } }),
    )

    const error = await captureRejection(customFetch('/api/v1/words'))

    expect(error.kind).toBe('http')
    expect(error.retryAfterSeconds).toBe(30)
  })
})

describe('网络错误分类', () => {
  it('fetch 连接失败归为 network', async () => {
    stubFetch(() => {
      throw new TypeError('fetch failed')
    })

    const error = await captureRejection(customFetch('/api/v1/words'))

    expect(error.kind).toBe('network')
    expect(error.cause).toBeInstanceOf(TypeError)
  })

  it('配置 timeoutMs 后超时归为 timeout', async () => {
    configureApiClient({ timeoutMs: 10 })
    stubHangingFetch()

    const error = await captureRejection(customFetch('/api/v1/words'))

    expect(error.kind).toBe('timeout')
  })

  it('调用方中止归为 abort', async () => {
    stubHangingFetch()
    const controller = new AbortController()
    const promise = customFetch('/api/v1/words', { signal: controller.signal })
    controller.abort()

    const error = await captureRejection(promise)

    expect(error.kind).toBe('abort')
  })

  it('发起前已中止时不发送请求', async () => {
    stubHangingFetch()
    const controller = new AbortController()
    controller.abort()

    const error = await captureRejection(customFetch('/api/v1/words', { signal: controller.signal }))

    expect(error.kind).toBe('abort')
    expect(calls).toHaveLength(0)
  })
})

describe('请求构造', () => {
  it('Base URL 前缀由 configureApiClient 注入', async () => {
    configureApiClient({ baseUrl: 'https://api.example.net' })
    stubFetch(() => jsonResponse(okEnvelope({})))

    await customFetch('/api/v1/words')

    expect(calls[0]?.url).toBe('https://api.example.net/api/v1/words')
  })

  it('默认 Accept 与 credentials: omit', async () => {
    stubFetch(() => jsonResponse(okEnvelope({})))

    await customFetch('/api/v1/words', { method: 'POST', body: '{}' })

    const headers = new Headers(calls[0]?.init.headers)
    expect(calls[0]?.init.credentials).toBe('omit')
    expect(headers.get('Accept')).toBe('application/json')
  })

  it('默认不携带 Authorization（MVP 服务端无认证端点）', async () => {
    stubFetch(() => jsonResponse(okEnvelope({})))

    await customFetch('/api/v1/words')

    expect(new Headers(calls[0]?.init.headers).get('Authorization')).toBeNull()
  })

  it('配置 getAccessToken 后附加 Bearer（支持异步）', async () => {
    configureApiClient({ getAccessToken: async () => 'tok-1' })
    stubFetch(() => jsonResponse(okEnvelope({})))

    await customFetch('/api/v1/words')

    expect(new Headers(calls[0]?.init.headers).get('Authorization')).toBe('Bearer tok-1')
  })
})
