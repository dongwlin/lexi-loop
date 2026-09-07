// 复习页集成测试（前端测试规范 §11）：active session 恢复（D010）——
// 「继续复习」把焦点移入复习区域（Space / 1 / 2 快捷键可用）、「放弃本轮」调用
// abandon 端点由服务端标记 abandoned（api/reviews.md §6），瞬时失败保留快照
// 可重试，404 / 422 视为快照失效回数量配置视图。只在 HTTP 边界 Mock。
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
