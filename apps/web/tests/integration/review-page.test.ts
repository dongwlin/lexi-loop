// 复习页集成测试（前端测试规范 §11）：active session 恢复（D010）——
// 「继续复习」把焦点移入复习区域（Space / 1 / 2 快捷键可用）、「放弃本轮」调用
// abandon 端点由服务端标记 abandoned（api/reviews.md §6），瞬时失败保留快照
// 可重试，404 / 422 视为快照失效回数量配置视图；数量选择（review-flow §6）——
// 预设与自定义同组选项、自定义原地切换输入框无需确认、上次选择本地持久化恢复。
// 只在 HTTP 边界 Mock。
import { HttpResponse, http } from 'msw'
import { screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it } from 'vitest'

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

    await user.click(screen.getByRole('button', { name: '自定义' }))

    const input = screen.getByLabelText('自定义数量')
    expect(input).toBeInTheDocument()
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
    expect(JSON.parse(localStorage.getItem('lexi-loop.review-count'))).toEqual({
      version: 1,
      mode: 'preset',
      count: 20,
    })

    // 再点自定义：输入框带回上一次输入的 25 并即时生效，存储同步回自定义 25。
    await user.click(screen.getByRole('button', { name: '自定义' }))
    expect(screen.getByLabelText('自定义数量')).toHaveValue(25)
    expect(JSON.parse(localStorage.getItem('lexi-loop.review-count'))).toEqual({
      version: 1,
      mode: 'custom',
      count: 25,
    })

    await user.click(screen.getByRole('button', { name: '开始复习' }))
    expect(postedBodies).toEqual([{ count: 25 }])
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
    expect(JSON.parse(localStorage.getItem('lexi-loop.review-count'))).toEqual({
      version: 1,
      mode: 'custom',
      count: 2,
    })
    first.unmount()

    await renderAppAtRoute('/review')
    expect(screen.getByLabelText('自定义数量')).toHaveValue(2)
  })
})
