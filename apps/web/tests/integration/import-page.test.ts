// 导入页集成测试（前端测试规范 §11）：导入成功后的聚合统计与逐词反馈
// （review-flow §3「ambiguous 新增 / constrain 已存在 +2」）。
import { HttpResponse, http } from 'msw'
import { screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import ImportPage from '@/pages/import/ImportPage.vue'

import { makeImportWordsResponse } from '../mocks/handlers'
import { server } from '../mocks/server'
import { renderAtRoute } from './helpers'

describe('导入页（review-flow §2–§3）', () => {
  it('导入成功后展示聚合统计与逐词结果', async () => {
    const user = userEvent.setup()
    server.use(
      http.post('*/api/v1/words/import', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeImportWordsResponse(),
        }),
      ),
    )

    await renderAtRoute(ImportPage, '/import')

    await user.type(
      screen.getByLabelText('每行输入一个单词'),
      'ambiguous\nconstrain\nderive',
    )
    await user.click(screen.getByRole('button', { name: '导入' }))

    expect(await screen.findByText('导入完成')).toBeVisible()

    // 聚合统计（created / updated / encounters）。
    expect(screen.getByText('新增单词').closest('div')).toHaveTextContent('3')
    expect(screen.getByText('已有单词').closest('div')).toHaveTextContent('1')
    expect(screen.getByText('本次遇到次数').closest('div')).toHaveTextContent(
      '4',
    )

    // 逐词结果：新增与已存在（含本次累计次数）分别标注。
    expect(screen.getByText('ambiguous')).toBeVisible()
    expect(screen.getAllByText('新增')).toHaveLength(2)
    expect(screen.getByText('constrain')).toBeVisible()
    expect(screen.getByText('已存在 +2')).toBeInTheDocument()
    expect(screen.getByText('derive')).toBeVisible()
  })
})
