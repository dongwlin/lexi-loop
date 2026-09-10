/// <reference lib="dom" />
import { expect, test } from '@playwright/test'

for (const width of [375, 1280]) {
  test(`自动发音与本地设置 ${width}`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 })
    await page.addInitScript((dark) => {
      localStorage.setItem(
        'lexi-loop.theme',
        JSON.stringify({ version: 1, preference: dark ? 'dark' : 'light' }),
      )
      Object.defineProperty(window, 'speechSynthesis', {
        value: {
          speak: (utterance: SpeechSynthesisUtterance) => {
            const root = document.documentElement
            root.dataset.spokenWord = utterance.text
            root.dataset.speakCount = String(
              Number(root.dataset.speakCount ?? 0) + 1,
            )
          },
          cancel: () => {
            delete document.documentElement.dataset.spokenWord
          },
        },
      })
    }, width === 375)
    const sessionId = '0f0f3d5a-0000-5000-8000-00000000c001'
    const items = ['ambiguous', 'brisk', 'cite'].map((word, index) => ({
      itemId: `0f0f3d5a-0000-6000-8000-00000000000${index + 1}`,
      word,
      phonetic: '/test/',
      effectiveReviewMeaning: [{ pos: 'adj', translations: ['测试释义'] }],
    }))
    const submissions: unknown[] = []
    await page.route('**/api/v1/words?*', (route) =>
      route.fulfill({
        json: {
          code: 'OK',
          message: 'ok',
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
        },
      }),
    )
    await page.route('**/api/v1/reviews', (route) =>
      route.fulfill({
        json: {
          code: 'OK',
          message: 'ok',
          data: { sessionId, requestedCount: 30, totalCount: 3, items },
        },
      }),
    )
    await page.route(`**/api/v1/reviews/${sessionId}`, (route) =>
      route.fulfill({
        json: {
          code: 'OK',
          message: 'ok',
          data: {
            sessionId,
            status: 'active',
            total: 3,
            remembered: submissions.length,
            forgotten: 0,
            completedAt: null,
            items: items.map((item, index) => ({
              word: item.word,
              result: index < submissions.length ? 'remembered' : 'pending',
            })),
          },
        },
      }),
    )
    await page.route(`**/api/v1/reviews/${sessionId}/items/*`, (route) => {
      submissions.push(route.request().postDataJSON())
      return route.fulfill({ json: { code: 'OK', message: 'ok', data: {} } })
    })
    await page.goto('/review')
    const root = page.locator('html')
    const gear = page.getByRole('button', { name: '复习设置', exact: true })
    const checkbox = page.getByRole('checkbox', { name: '自动播放单词发音' })
    await page.getByRole('button', { name: '开始复习', exact: true }).click()
    await expect(root).toHaveAttribute('data-spoken-word', 'ambiguous')
    await expect(root).toHaveAttribute('data-speak-count', '1')
    for (const target of [
      gear,
      page.getByRole('button', { name: '播放 ambiguous 的发音' }),
    ]) {
      const box = await target.boundingBox()
      expect(box!.width).toBeGreaterThanOrEqual(44)
      expect(box!.height).toBeGreaterThanOrEqual(44)
    }
    await page.getByRole('button', { name: '记得', exact: true }).click()
    await expect(root).toHaveAttribute('data-speak-count', '1')
    await page.getByRole('button', { name: '下一词' }).click()
    await expect(root).toHaveAttribute('data-spoken-word', 'brisk')
    await expect(root).toHaveAttribute('data-speak-count', '2')
    await gear.click()
    await expect(page.getByRole('dialog', { name: '复习设置' })).toBeVisible()
    await expect(checkbox).toBeChecked()
    await checkbox.uncheck()
    await page.screenshot({ path: `/tmp/review-settings-${width}.png` })
    await page.keyboard.press('Escape')
    await expect(gear).toBeFocused()
    await expect(page.getByText('先回忆这个单词的意思')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'brisk' })).toBeVisible()
    expect(submissions).toEqual([{ result: 'remembered' }])
    await expect(root).toHaveAttribute('data-speak-count', '2')
    await page.getByRole('button', { name: '记得', exact: true }).click()
    await page.getByRole('button', { name: '下一词' }).click()
    await expect(page.getByRole('heading', { name: 'cite' })).toBeVisible()
    await expect(root).not.toHaveAttribute('data-spoken-word')
    await expect(root).toHaveAttribute('data-speak-count', '2')
    await page.getByRole('button', { name: '播放 cite 的发音' }).press('Enter')
    await expect(root).toHaveAttribute('data-spoken-word', 'cite')
    await expect(root).toHaveAttribute('data-speak-count', '3')
    expect(submissions).toHaveLength(2)
    await page.reload()
    await gear.click()
    await expect(checkbox).not.toBeChecked()
    await page.keyboard.press('Escape')
    await page.getByRole('button', { name: '继续复习' }).click()
    await expect(page.getByRole('heading', { name: 'cite' })).toBeVisible()
    await expect(root).not.toHaveAttribute('data-speak-count')
    await gear.click()
    await checkbox.check()
    await page.keyboard.press('Escape')
    await expect(root).not.toHaveAttribute('data-speak-count')
    await expect(page.getByText('先回忆这个单词的意思')).toBeVisible()
    expect(submissions).toHaveLength(2)
    await page.reload()
    await page.getByRole('button', { name: '继续复习' }).click()
    await expect(root).toHaveAttribute('data-spoken-word', 'cite')
    await expect(root).toHaveAttribute('data-speak-count', '1')
    await page.screenshot({ path: `/tmp/review-autoplay-${width}.png` })
  })
}
