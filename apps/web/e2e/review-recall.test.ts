/// <reference lib="dom" />
import { expect, test } from '@playwright/test'

for (const scenario of [
  { width: 1280, theme: 'light' },
  { width: 375, theme: 'dark' },
]) {
  test(`主动回忆、焦点与提交保护 ${scenario.width} ${scenario.theme}`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: scenario.width, height: 900 })
    await page.addInitScript(
      (theme) =>
        localStorage.setItem(
          'lexi-loop.theme',
          JSON.stringify({ version: 1, preference: theme }),
        ),
      scenario.theme,
    )
    const sessionId = '0f0f3d5a-0000-5000-8000-00000000c001'
    const items = ['ambiguous', 'brisk'].map((word, index) => ({
      itemId: `0f0f3d5a-0000-6000-8000-00000000000${index + 1}`,
      word,
      phonetic: '',
      effectiveReviewMeaning: [
        { pos: 'adj', translations: ['模棱两可的；含糊不清的；有歧义的'] },
      ],
    }))
    const submissions: unknown[] = []
    let release: () => void = () => {}
    const blocked = new Promise<void>((resolve) => {
      release = resolve
    })
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
              total: 2,
              totalPages: 2,
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
          data: { sessionId, requestedCount: 30, totalCount: 2, items },
        },
      }),
    )
    await page.route(
      `**/api/v1/reviews/${sessionId}/items/*`,
      async (route) => {
        submissions.push(route.request().postDataJSON())
        await blocked
        await route.fulfill({ json: { code: 'OK', message: 'ok', data: {} } })
      },
    )
    await page.goto('/review')
    await expect(page.locator('html')).toHaveClass(scenario.theme)
    await page.getByRole('button', { name: '开始复习', exact: true }).click()
    const area = page
      .locator('div[tabindex="-1"]')
      .filter({ has: page.getByRole('heading', { name: 'ambiguous' }) })
    await expect(area).toBeFocused()
    await page.screenshot({
      path: `/tmp/review-recall-${scenario.width}-initial.png`,
    })
    await page.keyboard.press('2')
    await expect(
      page.getByText('adj. 模棱两可的；含糊不清的；有歧义的', { exact: true }),
    ).toBeVisible()
    await expect(area).toBeFocused()
    expect(submissions).toEqual([])
    await page.screenshot({
      path: `/tmp/review-recall-${scenario.width}-revealed.png`,
    })
    // Tab 到修正按钮，用原生 Enter 激活，焦点恢复到区域。
    await page.keyboard.press('Tab')
    await expect(
      page.getByRole('button', { name: '不认识', exact: true }),
    ).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(area).toBeFocused()
    await expect(
      page.getByRole('button', { name: '不认识', exact: true }),
    ).toHaveCount(0)
    await page.keyboard.press('2')
    await expect(
      page.getByRole('button', { name: '认识', exact: true }),
    ).toHaveCount(0)
    // 长按事件不得直接提交。
    await area.dispatchEvent('keydown', {
      key: 'Enter',
      code: 'Enter',
      repeat: true,
    })
    expect(submissions).toEqual([])
    await page.keyboard.press('Tab')
    await expect(page.getByRole('button', { name: '下一词' })).toBeFocused()
    await page.keyboard.press('Space')
    await expect(page.getByRole('button', { name: '下一词' })).toBeDisabled()
    await expect.poll(() => submissions.length).toBe(1)
    await area.focus()
    await page.keyboard.press('Enter')
    await page.keyboard.press('1')
    expect(submissions).toEqual([{ result: 'forgotten' }])
    release()
    await expect(page.getByRole('heading', { name: 'brisk' })).toBeVisible()
    await expect(
      page
        .locator('div[tabindex="-1"]')
        .filter({ has: page.getByRole('heading', { name: 'brisk' }) }),
    ).toBeFocused()
    await expect(
      page.getByRole('button', { name: '认识', exact: true }),
    ).toBeVisible()
    await expect(page.getByRole('button', { name: '下一词' })).toHaveCount(0)
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true)
  })
}
