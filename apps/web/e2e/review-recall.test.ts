/// <reference lib="dom" />
import { expect, test } from '@playwright/test'

for (const scenario of [
  { width: 1280, theme: 'light', hasTouch: false },
  { width: 375, theme: 'dark', hasTouch: true },
  { width: 1280, theme: 'light', hasTouch: true },
  { width: 375, theme: 'dark', hasTouch: false },
]) {
  test.describe(`输入能力 ${scenario.hasTouch ? 'touch' : 'pointer'} ${scenario.width}`, () => {
    test.use({ hasTouch: scenario.hasTouch })
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
      const starts: unknown[] = []
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
      await page.route('**/api/v1/reviews', (route) => {
        starts.push(route.request().postDataJSON())
        return route.fulfill({
          json: {
            code: 'OK',
            message: 'ok',
            data: { sessionId, requestedCount: 30, totalCount: 2, items },
          },
        })
      })
      await page.route(`**/api/v1/reviews/${sessionId}`, (route) =>
        route.fulfill({
          json: {
            code: 'OK',
            message: 'ok',
            data: {
              sessionId,
              status: starts.length > 1 ? 'active' : 'completed',
              total: 2,
              remembered: 1,
              forgotten: 1,
              createdAt: '2026-09-09T00:00:00Z',
              completedAt: starts.length > 1 ? null : '2026-09-09T00:01:00Z',
              items: items.map((item, index) => ({
                word: item.word,
                result:
                  starts.length > 1
                    ? 'pending'
                    : index === 0
                      ? 'forgotten'
                      : 'remembered',
              })),
            },
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
      // 验证真实媒体能力与计算样式，避免只缩小视口却仍模拟桌面输入。
      expect(
        await page.evaluate(
          () => matchMedia('(pointer: fine) and (hover: hover)').matches,
        ),
      ).toBe(!scenario.hasTouch)
      const recallHint = page.getByText('1 / ← 不认识 · 2 / → 认识', {
        exact: true,
      })
      const nextHint = page.locator('p').filter({ hasText: 'Space / Enter' })
      await expect(recallHint).toHaveCSS(
        'display',
        scenario.hasTouch ? 'none' : 'block',
      )
      await expect(recallHint).toBeVisible({ visible: !scenario.hasTouch })
      await page.screenshot({
        path: `/tmp/review-${scenario.hasTouch ? 'touch' : 'pointer'}-recall-${scenario.width}-initial.png`,
      })
      await page.keyboard.press('2')
      await expect(
        page.getByText('adj. 模棱两可的；含糊不清的；有歧义的', {
          exact: true,
        }),
      ).toBeVisible()
      await expect(area).toBeFocused()
      expect(submissions).toEqual([])
      await expect(nextHint).toContainText('1 / ← 修正为不认识')
      await expect(nextHint).toBeVisible({ visible: !scenario.hasTouch })
      await page.screenshot({
        path: `/tmp/review-${scenario.hasTouch ? 'touch' : 'pointer'}-recall-${scenario.width}-revealed.png`,
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
      await expect(nextHint).not.toContainText('修正为不认识')
      await expect(nextHint).toBeVisible({ visible: !scenario.hasTouch })
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
      // 完成最后一题：通过按钮原生 Enter 提交，进入结果页后直接续练。
      await page.keyboard.press('2')
      await page.keyboard.press('Tab')
      await page.keyboard.press('Tab')
      await expect(page.getByRole('button', { name: '下一词' })).toBeFocused()
      await page.keyboard.press('Enter')
      const replay = page.getByRole('button', { name: '再来一轮' })
      await expect(replay).toBeFocused()
      await expect(
        page.getByRole('heading', { name: '复习结果' }),
      ).not.toBeFocused()
      await page.screenshot({
        path: `/tmp/review-${scenario.hasTouch ? 'touch' : 'pointer'}-result-${scenario.width}.png`,
      })
      await page.keyboard.press('Tab')
      await expect(
        page.getByRole('link', { name: '回到生词库', exact: true }),
      ).toBeFocused()
      await page.keyboard.press('Shift+Tab')
      await expect(replay).toBeFocused()
      await page.keyboard.press(scenario.width === 1280 ? 'Enter' : 'Space')
      await expect(area).toBeFocused()
      await expect(
        page.getByRole('heading', { name: 'ambiguous' }),
      ).toBeVisible()
      expect(starts).toEqual([{ count: 30 }, { count: 2 }])
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true)
    })
  })
}
