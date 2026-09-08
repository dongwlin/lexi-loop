// 复习结果页集成测试（前端测试规范 §11）：「再来一轮」按上一轮实际词数直接
// 开始新一轮（review-flow §9 定稿），跨页断言复习页跳过恢复询问直接进入第一题；
// 开始失败时留在结果页展示错误与重试。
import { HttpResponse, http } from 'msw'
import { QueryClient } from '@tanstack/vue-query'
import { reviewKeys } from '@/features/review/api/keys'
import { screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { beforeAll, beforeEach, describe, expect, it } from 'vitest'

import { router } from '@/app/router'

import { makeSessionResponse } from '../mocks/handlers'
import { server } from '../mocks/server'
import { renderAppAtRoute } from './helpers'

const OLD_SESSION_ID = '0f0f3d5a-0000-5000-8000-00000000c001'
const NEW_SESSION_ID = '0f0f3d5a-0000-5000-8000-00000000c002'

const replayItems = ['ambient', 'brisk', 'cite'].map((word, index) => ({
  itemId: `0f0f3d5a-0000-6000-8000-00000000000${index + 1}`,
  word,
  phonetic: '',
  effectiveReviewMeaning: [],
}))

// 首次路由导航会懒加载页面；把 Vite 转换与模块加载放在套件准备阶段，
// 避免高负载下占用业务用例的 5s 时限。仍使用真实 RouterView 与 HTTP 链路。
beforeAll(async () => {
  await Promise.all([
    import('@/pages/review/ReviewPage.vue'),
    import('@/pages/review/ReviewResultPage.vue'),
  ])
})

beforeEach(() => {
  localStorage.clear()
  server.use(
    http.get('*/api/v1/words', () =>
      HttpResponse.json({
        code: 'OK',
        message: 'success',
        data: {
          list: [],
          pagination: {
            page: 1,
            pageSize: 20,
            total: 0,
            totalPages: 0,
            hasMore: false,
          },
        },
      }),
    ),
  )
})

describe('复习结果页「再来一轮」（review-flow §9）', () => {
  it.each(['{Enter}', ' '])(
    '使用 %s 按上一轮实际词数开始新一轮并直接进入第一题',
    async (key) => {
      const user = userEvent.setup()
      const postedBodies: unknown[] = []
      server.use(
        http.get(`*/api/v1/reviews/${OLD_SESSION_ID}`, () =>
          HttpResponse.json({
            code: 'OK',
            message: 'success',
            data: makeSessionResponse({ sessionId: OLD_SESSION_ID }),
          }),
        ),
        http.post('*/api/v1/reviews', async ({ request }) => {
          postedBodies.push(await request.json())
          return HttpResponse.json({
            code: 'OK',
            message: 'success',
            data: {
              sessionId: NEW_SESSION_ID,
              requestedCount: 3,
              totalCount: 3,
              items: replayItems,
            },
          })
        }),
        http.get(`*/api/v1/reviews/${NEW_SESSION_ID}`, () =>
          HttpResponse.json({
            code: 'OK',
            message: 'success',
            data: makeSessionResponse({
              sessionId: NEW_SESSION_ID,
              status: 'active',
              remembered: 0,
              forgotten: 0,
              completedAt: null,
              items: replayItems.map((item) => ({
                word: item.word,
                result: 'pending',
              })),
            }),
          }),
        ),
        http.get('*/api/v1/words', () =>
          HttpResponse.json({
            code: 'OK',
            message: 'success',
            data: {
              list: [],
              pagination: {
                page: 1,
                pageSize: 1,
                total: 5,
                totalPages: 5,
                hasMore: true,
              },
            },
          }),
        ),
      )

      await renderAppAtRoute(`/review/result/${OLD_SESSION_ID}`)
      expect(await screen.findByText('本轮完成')).toBeVisible()

      await waitFor(() =>
        expect(screen.getByRole('button', { name: /再来一轮/ })).toHaveFocus(),
      )
      await user.keyboard(key)

      // 以 session 汇总的 total（上一轮实际词数 3）作为新一轮数量。
      expect(postedBodies).toEqual([{ count: 3 }])

      // 跳转复习页：不走「继续 / 放弃」恢复询问，直接进入第一题。
      await waitFor(() => expect(router.currentRoute.value.name).toBe('review'))
      expect(await screen.findByText('ambient')).toBeVisible()
      expect(screen.queryByText(/上次复习未完成/)).not.toBeInTheDocument()

      // autoResume 是一次性标记：进入即消费，之后的刷新恢复仍由用户选择（D010）。
      const snapshot = JSON.parse(
        localStorage.getItem('lexi-loop.review-snapshot') ?? '{}',
      )
      expect(snapshot.sessionId).toBe(NEW_SESSION_ID)
      expect(snapshot.autoResume).toBe(false)
    },
  )

  it('延迟加载后聚焦主操作，Tab 顺序和后台刷新保留用户焦点', async () => {
    const user = userEvent.setup()
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    let release!: () => void
    const blocked = new Promise<void>((resolve) => {
      release = resolve
    })
    server.use(
      http.get(`*/api/v1/reviews/${OLD_SESSION_ID}`, async () => {
        await blocked
        return HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeSessionResponse({ sessionId: OLD_SESSION_ID }),
        })
      }),
    )
    await renderAppAtRoute(`/review/result/${OLD_SESSION_ID}`, client)
    expect(await screen.findByText('正在加载复习结果…')).toBeVisible()
    expect(
      screen.queryByRole('button', { name: /再来一轮/ }),
    ).not.toBeInTheDocument()
    release()
    const replay = await screen.findByRole('button', { name: /再来一轮/ })
    await waitFor(() => expect(replay).toHaveFocus())
    await user.tab()
    const back = screen.getByRole('link', { name: '回到生词库' })
    expect(back).toHaveFocus()
    await client.refetchQueries({
      queryKey: reviewKeys.session(OLD_SESSION_ID),
    })
    expect(back).toHaveFocus()
    await user.keyboard('{Enter}')
    await waitFor(() => expect(router.currentRoute.value.name).toBe('words'))
  })

  it('页面级导航命中缓存时，主操作焦点不会被全局标题焦点覆盖', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { staleTime: Infinity, retry: false } },
    })
    client.setQueryData(
      reviewKeys.session(OLD_SESSION_ID),
      makeSessionResponse({ sessionId: OLD_SESSION_ID }),
    )
    await renderAppAtRoute('/review', client)
    await router.push(`/review/result/${OLD_SESSION_ID}`)
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /再来一轮/ })).toHaveFocus(),
    )
    expect(screen.getByRole('heading', { name: '复习结果' })).not.toHaveFocus()
  })

  it.each(['active', 'abandoned'] as const)(
    '轮次 %s 时不聚焦续练按钮',
    async (status) => {
      server.use(
        http.get(`*/api/v1/reviews/${OLD_SESSION_ID}`, () =>
          HttpResponse.json({
            code: 'OK',
            message: 'success',
            data: makeSessionResponse({ sessionId: OLD_SESSION_ID, status }),
          }),
        ),
      )
      await renderAppAtRoute('/review')
      await router.push(`/review/result/${OLD_SESSION_ID}`)
      expect(
        await screen.findByRole('button', { name: '去复习' }),
      ).toBeVisible()
      expect(
        screen.queryByRole('button', { name: /再来一轮/ }),
      ).not.toBeInTheDocument()
      expect(screen.getByRole('heading', { name: '复习结果' })).toHaveFocus()
    },
  )

  it('总词数为零时不聚焦禁用按钮', async () => {
    server.use(
      http.get(`*/api/v1/reviews/${OLD_SESSION_ID}`, () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeSessionResponse({ sessionId: OLD_SESSION_ID, total: 0 }),
        }),
      ),
    )
    await renderAppAtRoute(`/review/result/${OLD_SESSION_ID}`)
    const replay = await screen.findByRole('button', { name: /再来一轮/ })
    expect(replay).toBeDisabled()
    expect(replay).not.toHaveFocus()
  })

  it('开始失败时留在结果页展示错误', async () => {
    const user = userEvent.setup()
    server.use(
      http.get(`*/api/v1/reviews/${OLD_SESSION_ID}`, () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeSessionResponse({ sessionId: OLD_SESSION_ID }),
        }),
      ),
      http.post('*/api/v1/reviews', () =>
        HttpResponse.json(
          {
            code: 'BASE.BIZ.USER_DISABLED',
            message: 'no reviewable words available',
            data: {},
          },
          { status: 422 },
        ),
      ),
    )

    await renderAppAtRoute(`/review/result/${OLD_SESSION_ID}`)
    expect(await screen.findByText('本轮完成')).toBeVisible()

    await user.click(screen.getByRole('button', { name: /再来一轮/ }))

    expect(
      await screen.findByText('当前没有可复习的生词，请先导入生词。'),
    ).toBeVisible()
    expect(router.currentRoute.value.name).toBe('review-result')
    expect(screen.getByRole('alert')).toHaveFocus()
  })
})
