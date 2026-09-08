/// <reference lib="dom" />
import { expect, test } from '@playwright/test'

for (const newline of ['\n', '\\n']) {
  test(`复习与恢复后保留 ${JSON.stringify(newline)} 和词性分行`, async ({
    page,
  }) => {
    const first = '（英）把…理想化（等于idealize）'
    const second = 'vi. 形成理想；理想化地表现'
    const long = '这是一段用于验证复习长释义自动折行的文本。'.repeat(12)
    const sessionId = '0f0f3d5a-0000-5000-8000-00000000c001'
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
              total: 1,
              totalPages: 1,
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
            sessionId,
            requestedCount: 30,
            totalCount: 1,
            items: [
              {
                itemId: '0f0f3d5a-0000-6000-8000-000000000001',
                word: 'idealised',
                phonetic: '',
                effectiveReviewMeaning: [
                  {
                    pos: 'transitive verb',
                    translations: [`${first}${newline}${second}`],
                  },
                  { pos: 'noun', translations: [long] },
                ],
              },
            ],
          },
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
            total: 1,
            remembered: 0,
            forgotten: 0,
            completedAt: null,
            items: [{ word: 'idealised', result: 'pending' }],
          },
        },
      }),
    )
    await page.goto('/review')
    await page.getByRole('button', { name: '开始复习', exact: true }).click()

    for (const width of [1280, 375]) {
      await page.setViewportSize({ width, height: 900 })
      await expect(page.getByText(first, { exact: false })).not.toBeVisible()
      await page.getByRole('button', { name: '认识', exact: true }).click()
      const meaning = page.getByText(first, { exact: false })
      await expect(meaning).not.toContainText('\\n')
      const layout = await meaning.evaluate(
        (element, texts) => {
          const node = element.firstChild!
          const content = node.textContent!
          const tops = texts.map((text) => {
            const start = content.indexOf(text)
            if (start < 0) throw new Error(`Missing text: ${text}`)
            const range = document.createRange()
            range.setStart(node, start)
            range.setEnd(node, start + 1)
            return range.getBoundingClientRect().top
          })
          const last = document.createRange()
          last.setStart(node, content.trimEnd().length - 1)
          last.setEnd(node, content.trimEnd().length)
          return {
            tops,
            lastTop: last.getBoundingClientRect().top,
            width: element.clientWidth,
            scrollWidth: element.scrollWidth,
          }
        },
        [first, second, 'noun.'],
      )
      expect(layout.tops[1]).toBeGreaterThan(layout.tops[0])
      expect(layout.tops[2]).toBeGreaterThan(layout.tops[1])
      expect(layout.lastTop).toBeGreaterThan(layout.tops[2])
      expect(layout.scrollWidth).toBeLessThanOrEqual(layout.width)

      if (width === 1280) {
        await page.reload()
        await page.getByRole('button', { name: '继续复习' }).click()
      }
    }
  })

  test(`详情释义保留 ${JSON.stringify(newline)} 换行且长文本自动折行`, async ({
    page,
  }) => {
    const first = '（英）把…理想化（等于idealize）'
    const second = 'vi. 形成理想；理想化地表现'
    const long = '这是一段用于验证正常长释义自动换行的文本。'.repeat(12)
    const word = {
      id: '0f0f3d5a-0000-4000-8000-000000000001',
      word: 'idealised',
      phonetic: '',
      definition: '',
      encounterCount: 1,
      reviewCount: 0,
      rememberCount: 0,
      forgetCount: 0,
      currentStreak: 0,
      lastReviewedAt: null,
      masteryScore: 0,
      reviewWeight: 1,
      meaningSource: 'raw',
      effectiveReviewMeaning: [
        {
          pos: 'transitive verb',
          translations: [`${first}${newline}${second}`],
        },
        { pos: 'noun', translations: [long] },
      ],
    }
    await page.route(`**/api/v1/words/${word.id}`, (route) =>
      route.fulfill({ json: { code: 'OK', message: 'ok', data: word } }),
    )
    await page.goto(`/words/${word.id}`)
    const meaning = page.getByRole('listitem').filter({ hasText: first })
    await expect(meaning).toBeVisible()
    await expect(meaning).not.toContainText('\\n')

    for (const width of [1280, 375]) {
      await page.setViewportSize({ width, height: 900 })
      // Range measures rendered text positions, so this catches CSS whitespace
      // collapse as well as escaped-newline decoding regressions.
      const positions = await meaning.evaluate(
        (element, texts) => {
          return texts.map((text) => {
            const walker = document.createTreeWalker(
              element,
              NodeFilter.SHOW_TEXT,
            )
            while (walker.nextNode()) {
              const node = walker.currentNode
              const start = node.textContent?.indexOf(text) ?? -1
              if (start < 0) continue
              const range = document.createRange()
              range.setStart(node, start)
              range.setEnd(node, start + 1)
              return range.getBoundingClientRect().top
            }
            throw new Error(`Missing text: ${text}`)
          })
        },
        [first, second],
      )
      expect(positions[1]).toBeGreaterThan(positions[0])

      const longMeaning = page.getByRole('listitem').filter({ hasText: long })
      const layout = await longMeaning.evaluate((element) => ({
        height: element.getBoundingClientRect().height,
        lineHeight: Number.parseFloat(getComputedStyle(element).lineHeight),
        width: element.clientWidth,
        scrollWidth: element.scrollWidth,
      }))
      expect(layout.height).toBeGreaterThan(layout.lineHeight)
      expect(layout.scrollWidth).toBeLessThanOrEqual(layout.width)
    }
  })
}
