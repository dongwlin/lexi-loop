/// <reference lib="dom" />
import { expect, test } from '@playwright/test'
import type { Locator } from '@playwright/test'

async function box(locator: Locator) {
  const bounds = await locator.boundingBox()
  expect(bounds).not.toBeNull()
  return bounds!
}

for (const width of [1280, 375, 320]) {
  test(`复习布局稳定且长内容完整 ${width}`, async ({ page }, testInfo) => {
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
      'v. 推导\n源自\n获得',
      'v. 限制\n约束\n限定范围\n抑制发展',
      Array.from(
        { length: 16 },
        (_, i) => `释义 ${i + 1}：${'很长的释义需要完整显示'.repeat(3)}`,
      ).join('\n') +
        '\n' +
        'unbroken'.repeat(30),
    ]
    const items = ['brisk', 'ambiguous', 'derive', 'constrain', longWord].map(
      (word, i) => ({
        word,
        itemId: `item-${i}`,
        phonetic: '',
        effectiveReviewMeaning: [{ pos: '', translations: [meanings[i]] }],
      }),
    )
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
              total: items.length,
              totalPages: items.length,
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
            totalCount: items.length,
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
    const measurements: Array<{
      lines: number
      lineHeight: number
      meaningHeight: number
      recallingCard: Awaited<ReturnType<typeof box>>
      revealedCard: Awaited<ReturnType<typeof box>>
      recallingAction: Awaited<ReturnType<typeof box>>
      revealedAction: Awaited<ReturnType<typeof box>>
    }> = []
    // 显式 1～4 行短释义：排除额外自动折行，并验证移动端 4 行预留高度边界。
    for (let index = 0; index < 4; index++) {
      await test.step(`${index + 1} 行释义`, async () => {
        await expect(
          page.getByRole('heading', { name: items[index].word, exact: true }),
        ).toBeVisible()
        await assertStable(known)
        const recallingCard = await box(section)
        const recallingAction = await box(known)
        await page.screenshot({
          path: `/tmp/review-layout-${width}-${index + 1}-recalling.png`,
        })
        await (
          index % 2 === 0
            ? known
            : page.getByRole('button', { name: '不认识', exact: true })
        ).click()
        const meaning = page.getByText(meanings[index], { exact: true })
        await expect(meaning).toBeVisible()
        await assertStable(next)
        const meaningBox = await box(meaning)
        const lineHeight = await meaning.evaluate((el) =>
          Number.parseFloat(getComputedStyle(el).lineHeight),
        )
        expect(meaningBox.height).toBeCloseTo((index + 1) * lineHeight, 0)
        expect(
          await meaning.evaluate((el) => el.scrollWidth <= el.clientWidth),
        ).toBe(true)
        // 检查释义及直到外层卡片的祖先均未裁切，且文字处于操作区上方。
        expect(
          await meaning.evaluate((el) => {
            let current: Element | null = el
            while (current) {
              const style = getComputedStyle(current)
              if (
                style.overflowX !== 'visible' ||
                style.overflowY !== 'visible'
              )
                return false
              if (current.tagName === 'SECTION') return true
              current = current.parentElement
            }
            return false
          }),
        ).toBe(true)
        const action = await box(next)
        expect(action.y).toBeGreaterThanOrEqual(
          meaningBox.y + meaningBox.height,
        )
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
        ).toBe(true)
        await page.screenshot({
          path: `/tmp/review-layout-${width}-${index + 1}-revealed.png`,
        })
        measurements.push({
          lines: index + 1,
          lineHeight,
          meaningHeight: meaningBox.height,
          recallingCard,
          revealedCard: await box(section),
          recallingAction,
          revealedAction: action,
        })
        if (index % 2 === 0) {
          await page
            .getByRole('button', { name: '不认识', exact: true })
            .click()
          await assertStable(next)
        }
        await next.click()
      })
    }
    await testInfo.attach('ordinary-meaning-layout', {
      body: JSON.stringify({ width, measurements }, null, 2),
      contentType: 'application/json',
    })
    const heading = page.getByRole('heading', { name: longWord, exact: true })
    await expect(heading).toBeVisible()
    await known.click()
    const meaning = page.getByText(meanings[4], { exact: true })
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
