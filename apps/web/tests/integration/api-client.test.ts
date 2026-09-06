// 设施冒烟：真实请求链路（api-client 生成端点 → mutator → transport）执行到
// HTTP 边界，由 MSW 拦截（前端测试规范 §8.1）。验证 Envelope 解包与 ApiError
// 转换在测试环境下同样成立。
import { listWords, ApiError } from '@lexi-loop/api-client'
import { HttpResponse, http } from 'msw'
import { describe, expect, it } from 'vitest'

// 先引入应用侧装配：configureApiClient 注入 vitest.config.ts 里的测试 Base URL。
import '@/lib/api'
import { makeWordListItem } from '../mocks/handlers'
import { server } from '../mocks/server'

describe('API Client 经 MSW 的 HTTP 边界', () => {
  it('listWords 发出正确的路径与 query，并解包 OK envelope 返回 data', async () => {
    const captured: URL[] = []
    server.use(
      http.get('*/api/v1/words', ({ request }) => {
        captured.push(new URL(request.url))
        return HttpResponse.json({
          code: 'OK',
          message: '',
          data: {
            list: [makeWordListItem()],
            pagination: { page: 2, pageSize: 20, total: 21 },
          },
        })
      }),
    )

    const page = await listWords({ page: 2, pageSize: 20, search: 'ambiguous' })

    expect(captured).toHaveLength(1)
    expect(captured[0]?.pathname).toBe('/api/v1/words')
    expect(captured[0]?.searchParams.get('page')).toBe('2')
    expect(captured[0]?.searchParams.get('pageSize')).toBe('20')
    expect(captured[0]?.searchParams.get('search')).toBe('ambiguous')
    expect(page.list).toHaveLength(1)
    expect(page.list[0]?.word).toBe('ambiguous')
    expect(page.pagination.total).toBe(21)
  })

  it('非 OK envelope 转换为携带 code 的 ApiError', async () => {
    server.use(
      http.get('*/api/v1/words', () =>
        HttpResponse.json(
          {
            code: 'BASE.NOT_FOUND.USER',
            message: 'word not found',
            data: null,
          },
          { status: 404 },
        ),
      ),
    )

    const error: unknown = await listWords({ page: 1, pageSize: 20 }).then(
      () => null,
      (e) => e,
    )

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).kind).toBe('http')
    expect((error as ApiError).code).toBe('BASE.NOT_FOUND.USER')
    expect((error as ApiError).httpStatus).toBe(404)
  })
})
