// 复习页集成测试（前端测试规范 §11）：active session 恢复（D010）——
// 「继续复习」把焦点移入复习区域（Space / 1 / 2 快捷键可用）、「放弃本轮」调用
// abandon 端点由服务端标记 abandoned（api/reviews.md §6），瞬时失败保留快照
// 可重试，404 / 422 视为快照失效回数量配置视图；数量选择（review-flow §6）——
// 预设与自定义同组选项、自定义原地切换输入框无需确认、上次选择本地持久化恢复。
// 只在 HTTP 边界 Mock。
import { HttpResponse, http } from 'msw'
import { screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { beforeAll, beforeEach, describe, expect, it } from 'vitest'

import { saveReviewSnapshot } from '@/utils/review-snapshot'

import { renderAppAtRoute } from './helpers'
import { server } from '../mocks/server'

const SESSION_ID = '0f0f3d5a-0000-5000-8000-00000000c001'

const snapshotItems = ['ambient', 'brisk', 'cite'].map((word, index) => ({
  itemId: `0f0f3d5a-0000-6000-8000-00000000000${index + 1}`,
  word,
  phonetic: '',
  effectiveReviewMeaning: [],
}))

// 服务端逐词结果与本地快照按 word 对齐：第一题已答，恢复到第 2 题。
const serverItems = [
  { word: 'ambient', result: 'remembered' },
  { word: 'brisk', result: 'pending' },
  { word: 'cite', result: 'pending' },
]

function mockActiveSession() {
  server.use(
    http.get(`*/api/v1/reviews/${SESSION_ID}`, () =>
      HttpResponse.json({
        code: 'OK',
        message: 'success',
        data: {
          sessionId: SESSION_ID,
          status: 'active',
          total: 3,
          remembered: 1,
          forgotten: 0,
          completedAt: null,
          items: serverItems,
        },
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
            total: 3,
            totalPages: 3,
            hasMore: false,
          },
        },
      }),
    ),
  )
}

function seedSnapshot() {
  saveReviewSnapshot({
    sessionId: SESSION_ID,
    totalCount: 3,
    items: snapshotItems,
  })
}

async function nextFrames(count = 2) {
  for (let i = 0; i < count; i++) {
    await new Promise((resolve) => requestAnimationFrame(() => resolve(null)))
  }
}

// 首次路由导航会懒加载页面；把 Vite 转换与模块加载放在套件准备阶段，
// 避免高负载下占用业务用例的 5s 时限。仍使用真实 RouterView 与 HTTP 链路。
beforeAll(async () => {
  await import('@/pages/review/ReviewPage.vue')
  await import('@/pages/review/ReviewResultPage.vue')
})

beforeEach(() => {
  localStorage.clear()
})

describe('复习页恢复（review-flow §6 / D010）', () => {
  it('继续复习后焦点落在复习区域，快捷键立即可用', async () => {
    seedSnapshot()
    mockActiveSession()
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '继续复习' }),
      ).toBeInTheDocument(),
    )
    await user.click(screen.getByRole('button', { name: '继续复习' }))

    // 恢复到首个未答项（第 2 题）。
    await waitFor(() =>
      expect(screen.getByText('先回忆这个单词的意思')).toBeInTheDocument(),
    )
    expect(screen.getByText('brisk')).toBeInTheDocument()

    // 焦点在复习区域（tabindex=-1 的键盘捕获容器），Space / 1 / 2 无需先点一次。
    await nextFrames()
    const active = document.activeElement
    expect(active?.getAttribute('tabindex')).toBe('-1')
    expect(active?.tagName).toBe('DIV')
    expect(active?.textContent).toContain('brisk')
  })

  it('放弃本轮调用 abandon 端点并回到数量配置视图', async () => {
    seedSnapshot()
    mockActiveSession()
    let abandonCalls = 0
    server.use(
      http.post(`*/api/v1/reviews/${SESSION_ID}/abandon`, () => {
        abandonCalls++
        return HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: {},
        })
      }),
    )
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '放弃本轮' }),
      ).toBeInTheDocument(),
    )
    await user.click(screen.getByRole('button', { name: '放弃本轮' }))

    expect(abandonCalls).toBe(1)
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '开始复习' }),
      ).toBeInTheDocument(),
    )
    // 本地快照已清除：刷新语义下不应再出现恢复入口。
    expect(
      screen.queryByRole('button', { name: '放弃本轮' }),
    ).not.toBeInTheDocument()
    expect(localStorage.getItem('lexi-loop.review-snapshot')).toBeNull()
  })

  it('放弃失败（服务端 5xx）保留恢复入口并展示可重试错误', async () => {
    seedSnapshot()
    mockActiveSession()
    server.use(
      http.post(`*/api/v1/reviews/${SESSION_ID}/abandon`, () =>
        HttpResponse.json(
          { code: 'ERROR', message: 'internal error', data: {} },
          { status: 500 },
        ),
      ),
    )
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '放弃本轮' }),
      ).toBeInTheDocument(),
    )
    await user.click(screen.getByRole('button', { name: '放弃本轮' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '服务暂时不可用，请稍后重试。',
    )
    // 快照与恢复入口保留，可重试。
    expect(screen.getByRole('button', { name: '放弃本轮' })).toBeInTheDocument()
    expect(localStorage.getItem('lexi-loop.review-snapshot')).not.toBeNull()
  })

  it('放弃返回 404 时视为快照失效，清除后回数量配置视图', async () => {
    seedSnapshot()
    mockActiveSession()
    server.use(
      http.post(`*/api/v1/reviews/${SESSION_ID}/abandon`, () =>
        HttpResponse.json(
          {
            code: 'BASE.NOT_FOUND.USER',
            message: 'review session not found',
            data: {},
          },
          { status: 404 },
        ),
      ),
    )
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '放弃本轮' }),
      ).toBeInTheDocument(),
    )
    await user.click(screen.getByRole('button', { name: '放弃本轮' }))

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '开始复习' }),
      ).toBeInTheDocument(),
    )
    expect(localStorage.getItem('lexi-loop.review-snapshot')).toBeNull()
  })
})

describe('复习页数量选择（review-flow §6）', () => {
  const mockAvailableCount = () => {
    server.use(
      http.get('*/api/v1/words', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: {
            list: [],
            pagination: {
              page: 1,
              pageSize: 1,
              total: 427,
              totalPages: 427,
              hasMore: false,
            },
          },
        }),
      ),
    )
  }

  function mockStartSession(words: string[]) {
    const postedBodies: unknown[] = []
    server.use(
      http.post('*/api/v1/reviews', async ({ request }) => {
        postedBodies.push(await request.json())
        return HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: {
            sessionId: SESSION_ID,
            requestedCount: words.length,
            totalCount: words.length,
            items: words.map((word, index) => ({
              itemId: `0f0f3d5a-0000-6000-8000-000000000a0${index}`,
              word,
              phonetic: '',
              effectiveReviewMeaning: [],
            })),
          },
        })
      }),
    )
    return postedBodies
  }

  it('预设数量与自定义位于同一组选项，页面无独立自定义输入区与确认按钮', async () => {
    mockAvailableCount()
    await renderAppAtRoute('/review')

    for (const count of ['10', '20', '30', '50']) {
      expect(screen.getByRole('button', { name: count })).toBeInTheDocument()
    }
    expect(screen.getByRole('button', { name: '自定义' })).toBeInTheDocument()
    expect(screen.queryByLabelText('自定义数量')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: '确认' }),
    ).not.toBeInTheDocument()
    // 无记录时默认选中 30。
    expect(screen.getByRole('button', { name: '30' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  it('点击自定义后原地切换为输入框并聚焦，预设选项保持原位', async () => {
    mockAvailableCount()
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    // 按钮与输入框同用 w-20 槽位：替换前后占位一致，flex-wrap 临界宽度下不换行。
    expect(screen.getByRole('button', { name: '自定义' })).toHaveClass('w-20')
    await user.click(screen.getByRole('button', { name: '自定义' }))

    const input = screen.getByLabelText('自定义数量')
    expect(input).toBeInTheDocument()
    expect(input).toHaveClass('w-20')
    expect(
      screen.queryByRole('button', { name: '自定义' }),
    ).not.toBeInTheDocument()
    for (const count of ['10', '20', '30', '50']) {
      expect(screen.getByRole('button', { name: count })).toBeInTheDocument()
    }
    await waitFor(() => expect(input).toHaveFocus())
  })

  it('输入有效自定义数量后无需确认，开始复习直接使用该数量', async () => {
    mockAvailableCount()
    const postedBodies = mockStartSession(['ambient', 'brisk'])
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await user.click(screen.getByRole('button', { name: '自定义' }))
    await user.type(screen.getByLabelText('自定义数量'), '2')
    await user.click(screen.getByRole('button', { name: '开始复习' }))

    expect(postedBodies).toEqual([{ count: 2 }])
    // 开始成功进入第一题。
    await waitFor(() => expect(screen.getByText('ambient')).toBeInTheDocument())
  })

  it('自定义输入无效时开始复习展示错误且不发起请求', async () => {
    mockAvailableCount()
    const postedBodies = mockStartSession(['ambient'])
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await user.click(screen.getByRole('button', { name: '自定义' }))
    await user.click(screen.getByRole('button', { name: '开始复习' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('请输入正整数')
    // §8.4：错误经稳定 ID 与输入框关联（aria-invalid + aria-describedby）。
    expect(screen.getByLabelText('自定义数量')).toHaveAttribute(
      'aria-invalid',
      'true',
    )
    expect(screen.getByLabelText('自定义数量')).toHaveAttribute(
      'aria-describedby',
      'custom-count-error',
    )
    expect(postedBodies).toEqual([])
  })

  it('再次输入有效数量后错误提示清除且取消无效标记', async () => {
    mockAvailableCount()
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await user.click(screen.getByRole('button', { name: '自定义' }))
    await user.click(screen.getByRole('button', { name: '开始复习' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('请输入正整数')

    await user.type(screen.getByLabelText('自定义数量'), '3')

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByLabelText('自定义数量')).not.toHaveAttribute(
      'aria-invalid',
    )
    expect(screen.getByLabelText('自定义数量')).not.toHaveAttribute(
      'aria-describedby',
    )
  })

  it('预设数量的上一次选择被记忆并在下次进入时恢复', async () => {
    mockAvailableCount()
    const user = userEvent.setup()
    const first = await renderAppAtRoute('/review')

    await user.click(screen.getByRole('button', { name: '20' }))
    first.unmount()

    await renderAppAtRoute('/review')
    expect(screen.getByRole('button', { name: '20' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(screen.getByRole('button', { name: '30' })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
  })

  it('自定义模式与上一次输入值被记忆并在下次进入时恢复', async () => {
    mockAvailableCount()
    const user = userEvent.setup()
    const first = await renderAppAtRoute('/review')

    await user.click(screen.getByRole('button', { name: '自定义' }))
    await user.type(screen.getByLabelText('自定义数量'), '25')
    first.unmount()

    await renderAppAtRoute('/review')
    // 直接恢复为选中的自定义输入框并带回上次输入的数量。
    expect(screen.getByLabelText('自定义数量')).toHaveValue(25)
    for (const count of ['10', '20', '30', '50']) {
      expect(screen.getByRole('button', { name: count })).toHaveAttribute(
        'aria-pressed',
        'false',
      )
    }

    // 开始复习直接使用恢复的数量，无需重新输入。
    const postedBodies = mockStartSession(['ambient', 'brisk', 'cite'])
    await user.click(screen.getByRole('button', { name: '开始复习' }))
    expect(postedBodies).toEqual([{ count: 25 }])
  })

  it('自定义切预设再切回时带回残留输入，选中态与持久化保持一致', async () => {
    mockAvailableCount()
    const postedBodies = mockStartSession(['ambient', 'brisk'])
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    await user.click(screen.getByRole('button', { name: '自定义' }))
    await user.type(screen.getByLabelText('自定义数量'), '25')
    await user.click(screen.getByRole('button', { name: '20' }))
    expect(JSON.parse(localStorage.getItem('lexi-loop.review-count')!)).toEqual(
      {
        version: 1,
        mode: 'preset',
        count: 20,
      },
    )

    // 再点自定义：输入框带回上一次输入的 25 并即时生效，存储同步回自定义 25。
    await user.click(screen.getByRole('button', { name: '自定义' }))
    expect(screen.getByLabelText('自定义数量')).toHaveValue(25)
    expect(JSON.parse(localStorage.getItem('lexi-loop.review-count')!)).toEqual(
      {
        version: 1,
        mode: 'custom',
        count: 25,
      },
    )

    await user.click(screen.getByRole('button', { name: '开始复习' }))
    expect(postedBodies).toEqual([{ count: 25 }])
  })

  it('自定义输入无效时隐藏截断提示，输入有效后恢复', async () => {
    // 可复习量 3 < 预设 50：预设选中时出现 D009 截断提示。
    server.use(
      http.get('*/api/v1/words', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: {
            list: [],
            pagination: {
              page: 1,
              pageSize: 1,
              total: 3,
              totalPages: 3,
              hasMore: false,
            },
          },
        }),
      ),
    )
    const user = userEvent.setup()
    await renderAppAtRoute('/review')

    const hint = /当前只有 3 个可复习生词/
    await user.click(screen.getByRole('button', { name: '50' }))
    expect(screen.getByText(hint)).toBeInTheDocument()

    // 切到自定义（空输入）：selectedCount 是残留值，提示会失真，一并隐藏。
    await user.click(screen.getByRole('button', { name: '自定义' }))
    expect(screen.queryByText(hint)).not.toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()

    // 输入有效数量后提示基于新选中值恢复。
    await user.type(screen.getByLabelText('自定义数量'), '4')
    expect(screen.getByText(hint)).toBeInTheDocument()
  })

  it('逐键持久化：输入回退到空时停留在最后有效值并恢复', async () => {
    mockAvailableCount()
    const user = userEvent.setup()
    const first = await renderAppAtRoute('/review')

    await user.click(screen.getByRole('button', { name: '自定义' }))
    const input = screen.getByLabelText('自定义数量')
    await user.type(input, '25')
    await user.type(input, '{Backspace}{Backspace}')
    // 空输入不覆盖持久化：停留在最后有效值 2，刷新语义下恢复 2。
    expect(input).toHaveValue(null)
    expect(JSON.parse(localStorage.getItem('lexi-loop.review-count')!)).toEqual(
      {
        version: 1,
        mode: 'custom',
        count: 2,
      },
    )
    first.unmount()

    await renderAppAtRoute('/review')
    expect(screen.getByLabelText('自定义数量')).toHaveValue(2)
  })
})

// 主动回忆：真实页面与 HTTP 边界共同验证暂定结果不会提前提交。
describe('先判断再揭示释义', () => {
  async function startCards() {
    const submissions: { itemId: string; result: string }[] = []
    let failNext = false
    const cards = snapshotItems.map((item) => ({
      ...item,
      effectiveReviewMeaning: [
        { pos: 'noun', translations: [`${item.word}释义`] },
      ],
    }))
    mockActiveSession()
    server.use(
      http.post('*/api/v1/reviews', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'ok',
          data: {
            sessionId: SESSION_ID,
            requestedCount: 30,
            totalCount: 3,
            items: cards,
          },
        }),
      ),
      http.post(
        `*/api/v1/reviews/${SESSION_ID}/items/:itemId`,
        async ({ params, request }) => {
          const body = (await request.json()) as { result: string }
          submissions.push({
            itemId: String(params.itemId),
            result: body.result,
          })
          if (failNext) {
            failNext = false
            return HttpResponse.json(
              { code: 'ERROR', message: 'error', data: {} },
              { status: 500 },
            )
          }
          return HttpResponse.json({ code: 'OK', message: 'ok', data: {} })
        },
      ),
    )
    const user = userEvent.setup()
    const view = await renderAppAtRoute('/review')
    await user.click(await screen.findByRole('button', { name: '开始复习' }))
    await screen.findByRole('heading', { name: 'ambient' })
    return {
      user,
      submissions,
      view,
      fail: () => {
        failNext = true
      },
    }
  }

  it.each([
    ['认识', false, 'remembered'],
    ['不认识', false, 'forgotten'],
    ['认识', true, 'forgotten'],
  ] as const)(
    '初选 %s、修正 %s：下一词才提交 %s',
    async (label, correct, result) => {
      const { user, submissions } = await startCards()
      expect(screen.queryByText('noun. ambient释义')).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: '查看释义' }),
      ).not.toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: label }))
      expect(screen.getByText('noun. ambient释义')).toBeInTheDocument()
      expect(submissions).toEqual([])
      expect(
        screen.queryByRole('button', { name: '认识' }),
      ).not.toBeInTheDocument()
      if (correct)
        await user.click(screen.getByRole('button', { name: '不认识' }))
      if (result === 'forgotten')
        expect(
          screen.queryByRole('button', { name: '不认识' }),
        ).not.toBeInTheDocument()
      expect(
        screen.getByRole('heading', { name: 'ambient' }),
      ).toBeInTheDocument()
      expect(submissions).toEqual([])
      await user.click(screen.getByRole('button', { name: '下一词' }))
      await screen.findByRole('heading', { name: 'brisk' })
      expect(submissions).toEqual([{ itemId: snapshotItems[0].itemId, result }])
      expect(screen.queryByText('noun. brisk释义')).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: '下一词' }),
      ).not.toBeInTheDocument()
      await nextFrames()
      expect(document.activeElement?.textContent).toContain('brisk')
    },
  )

  it('提交失败保留释义与最终结果，重试不会改变本卡结果', async () => {
    const { user, submissions, fail } = await startCards()
    await user.click(screen.getByRole('button', { name: '认识' }))
    fail()
    await user.click(screen.getByRole('button', { name: '下一词' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('提交失败')
    expect(screen.getByText('noun. ambient释义')).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: '不认识' }),
    ).not.toBeInTheDocument()
    await nextFrames()
    await user.keyboard('1')
    await user.keyboard('{Enter}')
    await screen.findByRole('heading', { name: 'brisk' })
    expect(submissions.map((item) => item.result)).toEqual([
      'remembered',
      'remembered',
    ])
  })

  it('键盘初选、单向修正与最后一题提交后进入结果页', async () => {
    const { user, submissions } = await startCards()
    await nextFrames()
    await user.keyboard('2')
    expect(submissions).toHaveLength(0)
    await nextFrames()
    await user.keyboard('1')
    await nextFrames()
    await user.keyboard('2') // 看过答案后不可向上修正。
    await user.keyboard(' ')
    await screen.findByRole('heading', { name: 'brisk' })
    await nextFrames()
    await user.keyboard('{ArrowLeft}')
    await nextFrames()
    await user.keyboard('{Enter}')
    await screen.findByRole('heading', { name: 'cite' })
    await nextFrames()
    await user.keyboard('{ArrowRight}')
    expect(submissions).toHaveLength(2)
    server.use(
      http.get(`*/api/v1/reviews/${SESSION_ID}`, () =>
        HttpResponse.json({
          code: 'OK',
          message: 'ok',
          data: {
            sessionId: SESSION_ID,
            status: 'completed',
            total: 3,
            remembered: 1,
            forgotten: 2,
            completedAt: '2026-09-08T08:00:00Z',
            items: serverItems.map((item, index) => ({
              ...item,
              result: index === 2 ? 'remembered' : 'forgotten',
            })),
          },
        }),
      ),
    )
    await nextFrames()
    await user.keyboard('{Enter}')
    await screen.findByText('本轮完成')
    expect(submissions.map((item) => item.result)).toEqual([
      'forgotten',
      'forgotten',
      'remembered',
    ])
    expect(localStorage.getItem('lexi-loop.review-snapshot')).toBeNull()
  })

  it('未提交的暂定结果不持久化，恢复从服务端 pending 卡重新判断', async () => {
    const { user, submissions, view } = await startCards()
    await user.click(screen.getByRole('button', { name: '认识' }))
    expect(submissions).toHaveLength(0)
    view.unmount()
    await renderAppAtRoute('/review')
    await user.click(await screen.findByRole('button', { name: '继续复习' }))
    await screen.findByRole('heading', { name: 'brisk' })
    expect(screen.queryByText('noun. brisk释义')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '认识' })).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: '下一词' }),
    ).not.toBeInTheDocument()
  })
})
