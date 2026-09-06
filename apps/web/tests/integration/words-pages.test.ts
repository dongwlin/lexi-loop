// 生词库 / 生词详情页面集成测试（前端测试规范 §11）：验证掌握度与复习优先级
// 两项服务端派生指标（D011）在列表列与详情统计中的展示。
import { HttpResponse, http } from 'msw'
import { screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import WordDetailPage from '@/pages/words/WordDetailPage.vue'
import WordsPage from '@/pages/words/WordsPage.vue'

import { makeWordDetail, makeWordListItem } from '../mocks/handlers'
import { server } from '../mocks/server'
import { renderAtRoute } from './helpers'

function wordsOk(list: ReturnType<typeof makeWordListItem>[]) {
  return HttpResponse.json({
    code: 'OK',
    message: 'success',
    data: {
      list,
      pagination: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1,
        hasMore: false,
      },
    },
  })
}

describe('生词库页（review-flow §4）', () => {
  it('表格展示掌握度与优先级列（服务端派生指标）', async () => {
    server.use(
      http.get('*/api/v1/words', () =>
        wordsOk([
          makeWordListItem(),
          makeWordListItem({
            id: '0f0f3d5a-0000-4000-8000-000000000002',
            word: 'diligent',
            masteryScore: 72,
            reviewWeight: 2.5,
          }),
        ]),
      ),
    )

    await renderAtRoute(WordsPage, '/words')

    expect(
      await screen.findByRole('columnheader', { name: '掌握度' }),
    ).toBeVisible()
    expect(screen.getByRole('columnheader', { name: '优先级' })).toBeVisible()

    const ambiguousRow = screen.getByRole('row', { name: /ambiguous/ })
    expect(ambiguousRow).toHaveTextContent('33%')
    expect(ambiguousRow).toHaveTextContent('4.62')

    const diligentRow = screen.getByRole('row', { name: /diligent/ })
    expect(diligentRow).toHaveTextContent('72%')
    expect(diligentRow).toHaveTextContent('2.5')
  })
})

describe('生词详情页（review-flow §5）', () => {
  it('学习统计展示当前掌握度与当前复习优先级', async () => {
    server.use(
      http.get('*/api/v1/words/:id', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeWordDetail(),
        }),
      ),
    )

    await renderAtRoute(
      WordDetailPage,
      '/words/0f0f3d5a-0000-4000-8000-000000000001',
    )

    expect(await screen.findByText('当前掌握度')).toBeVisible()
    expect(screen.getByText('33%')).toBeVisible()
    expect(screen.getByText('当前复习优先级')).toBeVisible()
    expect(screen.getByText('4.62')).toBeVisible()
  })
})
