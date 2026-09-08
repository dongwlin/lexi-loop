/// <reference lib="dom" />
import { expect, test } from '@playwright/test'
import type { Locator } from '@playwright/test'

async function box(locator: Locator) {
  const bounds = await locator.boundingBox()
  expect(bounds).not.toBeNull()
  return bounds!
}

for (const width of [1280, 375, 320]) {
  test(`复习布局稳定且长内容完整 ${width}`, async ({ page }) => {
    await page.setViewportSize({ width, height: 800 })
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.addInitScript(() =>
      localStorage.setItem(
        'lexi-loop.theme',
        JSON.stringify({ version: 1, preference: 'light' }),
      ),
    )
    const longWord = 'pneumonoultramicroscopicsilicovolcanoconiosis'
    const meanings = [
      'adj. 敏捷的',
      'adj. 模棱两可的\n含糊不清的',
      Array.from(
        { length: 16 },
        (_, i) => `释义 ${i + 1}：${'很长的释义需要完整显示'.repeat(3)}`,
      ).join('\n') +
        '\n' +
        'unbroken'.repeat(30),
    ]
    const items = ['brisk', 'ambiguous', longWord].map((word, i) => ({
      word,
      itemId: `item-${i}`,
      phonetic: '',
      effectiveReviewMeaning: [{ pos: '', translations: [meanings[i]] }],
    }))
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
          data: {
            sessionId: 'layout-session',
            requestedCount: 30,
            totalCount: 3,
            items,
          },
        },
      }),
    )
    await page.route('**/api/v1/reviews/layout-session/items/*', (route) =>
      route.fulfill({ json: { code: 'OK', message: 'ok', data: {} } }),
    )
    await page.goto('/review')
    await page.getByRole('button', { name: '开始复习', exact: true }).click()
    const section = page
      .locator('section')
      .filter({ has: page.getByRole('heading', { name: '开始复习' }) })
    const known = page.getByRole('button', { name: '认识', exact: true })
    const next = page.getByRole('button', { name: '下一词', exact: true })
    await expect(known).toBeVisible()
    const initialCard = await box(section)
    const initialAction = await box(known)
    const assertStable = async (action: Locator) => {
      expect((await box(section)).height).toBeCloseTo(initialCard.height, 0)
      const bounds = await box(action)
      expect(bounds.y).toBeCloseTo(initialAction.y, 0)
      expect(bounds.x).toBeCloseTo(initialAction.x, 0)
      expect(bounds.height).toBeGreaterThanOrEqual(44)
    }
    await page.screenshot({ path: `/tmp/review-layout-${width}-recalling.png` })
    await known.click()
    await expect(page.getByText(meanings[0], { exact: true })).toBeVisible()
    await assertStable(next)
    await page.getByRole('button', { name: '不认识', exact: true }).click()
    await assertStable(next)
    await next.click()
    await expect(
      page.getByRole('heading', { name: 'ambiguous', exact: true }),
    ).toBeVisible()
    await assertStable(known)
    await page.getByRole('button', { name: '不认识', exact: true }).click()
    await expect(page.getByText(meanings[1], { exact: true })).toBeVisible()
    await assertStable(next)
    await page.screenshot({ path: `/tmp/review-layout-${width}-revealed.png` })
    await next.click()
    const heading = page.getByRole('heading', { name: longWord, exact: true })
    await expect(heading).toBeVisible()
    await known.click()
    const meaning = page.getByText(meanings[2], { exact: true })
    await expect(meaning).toBeVisible()
    expect((await box(section)).height).toBeGreaterThan(initialCard.height)
    for (const text of [heading, meaning]) {
      const size = await text.evaluate((el) => ({
        width: el.clientWidth,
        scrollWidth: el.scrollWidth,
        height: el.clientHeight,
        scrollHeight: el.scrollHeight,
        overflow: getComputedStyle(el).overflow,
      }))
      expect(size.scrollWidth, JSON.stringify(size)).toBeLessThanOrEqual(
        size.width + 1,
      )
      expect(size.overflow).toBe('visible')
    }
    const meaningBox = await box(meaning)
    expect((await box(next)).y).toBeGreaterThanOrEqual(
      meaningBox.y + meaningBox.height,
    )
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true)
    await next.scrollIntoViewIfNeeded()
    await expect(next).toBeInViewport()
    await page.screenshot({
      path: `/tmp/review-layout-${width}-long.png`,
      fullPage: true,
    })
  })
}
