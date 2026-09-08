/// <reference lib="dom" />
import { expect, test } from '@playwright/test'

for (const newline of ['\n', '\\n']) {
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
