import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  abandonReviewSession,
  deleteWord,
  getReviewSession,
  getWord,
  importWords,
  listWords,
  startReviewSession,
  submitReviewResult,
  updateWordReviewMeaning,
} from './generated/client'

interface RecordedCall {
  url: string
  init: RequestInit
}

let calls: RecordedCall[] = []

function stubFetch(): void {
  calls = []
  vi.stubGlobal('fetch', (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    calls.push({ url, init: init ?? {} })
    return Promise.resolve(new Response(JSON.stringify({ code: 'OK', message: 'success', data: {} })))
  })
}

function lastCall(): RecordedCall {
  const call = calls.at(-1)
  if (call === undefined) throw new Error('expected a recorded fetch call')
  return call
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('words 端点', () => {
  it('importWords：POST /api/v1/words/import，JSON body', async () => {
    stubFetch()

    await importWords({ words: [{ word: 'ambiguous', count: 2 }] })

    const call = lastCall()
    expect(call.url).toBe('/api/v1/words/import')
    expect(call.init.method).toBe('POST')
    expect(new Headers(call.init.headers).get('Content-Type')).toBe('application/json')
    expect(JSON.parse(String(call.init.body))).toEqual({ words: [{ word: 'ambiguous', count: 2 }] })
  })

  it('listWords：GET /api/v1/words 携带分页与搜索', async () => {
    stubFetch()

    await listWords({ page: 2, pageSize: 50, search: 'amb' })

    expect(lastCall().url).toBe('/api/v1/words?page=2&pageSize=50&search=amb')
  })

  it('listWords：无参数时不带 query', async () => {
    stubFetch()

    await listWords()

    expect(lastCall().url).toBe('/api/v1/words')
  })

  it('getWord：GET /api/v1/words/:id', async () => {
    stubFetch()

    await getWord('01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001')

    const call = lastCall()
    expect(call.url).toBe('/api/v1/words/01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001')
    expect(call.init.method).toBe('GET')
  })

  it('updateWordReviewMeaning：PATCH，传 null 表示清除自定义', async () => {
    stubFetch()

    await updateWordReviewMeaning('01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001', { customReviewMeaning: null })

    const call = lastCall()
    expect(call.url).toBe('/api/v1/words/01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001')
    expect(call.init.method).toBe('PATCH')
    expect(JSON.parse(String(call.init.body))).toEqual({ customReviewMeaning: null })
  })

  it('deleteWord：DELETE /api/v1/words/:id', async () => {
    stubFetch()

    await deleteWord('01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001')

    const call = lastCall()
    expect(call.url).toBe('/api/v1/words/01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001')
    expect(call.init.method).toBe('DELETE')
  })
})

describe('reviews 端点', () => {
  it('startReviewSession：POST /api/v1/reviews', async () => {
    stubFetch()

    await startReviewSession({ count: 30 })

    const call = lastCall()
    expect(call.url).toBe('/api/v1/reviews')
    expect(call.init.method).toBe('POST')
    expect(JSON.parse(String(call.init.body))).toEqual({ count: 30 })
  })

  it('submitReviewResult：POST /api/v1/reviews/:sessionId/items/:itemId', async () => {
    stubFetch()

    await submitReviewResult(
      '01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001',
      '01991f3e-7b4c-7a21-8e3f-2c5d7a9b2002',
      { result: 'remembered' },
    )

    const call = lastCall()
    expect(call.url).toBe('/api/v1/reviews/01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001/items/01991f3e-7b4c-7a21-8e3f-2c5d7a9b2002')
    expect(JSON.parse(String(call.init.body))).toEqual({ result: 'remembered' })
  })

  it('abandonReviewSession：POST /api/v1/reviews/:sessionId/abandon，无请求体', async () => {
    stubFetch()

    await abandonReviewSession('01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001')

    const call = lastCall()
    expect(call.url).toBe('/api/v1/reviews/01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001/abandon')
    expect(call.init.method).toBe('POST')
    expect(call.init.body).toBeUndefined()
  })

  it('getReviewSession：GET /api/v1/reviews/:id', async () => {
    stubFetch()

    await getReviewSession('01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001')

    const call = lastCall()
    expect(call.url).toBe('/api/v1/reviews/01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001')
    expect(call.init.method).toBe('GET')
  })
})
