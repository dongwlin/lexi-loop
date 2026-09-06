import { describe, expect, it } from 'vitest'

import { formatMeanings } from './formatMeanings'

describe('formatMeanings', () => {
  it('空输入返回空串', () => {
    expect(formatMeanings(null)).toBe('')
    expect(formatMeanings(undefined)).toBe('')
    expect(formatMeanings([])).toBe('')
  })

  it('词性与翻译组合为「pos. 译文」', () => {
    expect(
      formatMeanings([
        { pos: 'adjective', translations: ['模棱两可的', '含糊不清的'] },
      ]),
    ).toBe('adjective. 模棱两可的；含糊不清的')
  })

  it('多个义项用「; 」分隔', () => {
    expect(
      formatMeanings([
        { pos: 'noun', translations: ['歧义'] },
        { pos: 'adjective', translations: ['模棱两可的'] },
      ]),
    ).toBe('noun. 歧义; adjective. 模棱两可的')
  })

  it('缺词性时只展示翻译', () => {
    expect(formatMeanings([{ translations: ['推导', '获得'] }])).toBe('推导；获得')
  })

  it('translations 为 null 或空数组时只保留词性', () => {
    expect(formatMeanings([{ pos: 'verb', translations: null }])).toBe('verb.')
    expect(formatMeanings([{ pos: 'verb', translations: [] }])).toBe('verb.')
  })

  it('词性与翻译都缺失的义项被跳过', () => {
    expect(formatMeanings([{ pos: 'verb', translations: ['运行'] }, {}])).toBe(
      'verb. 运行',
    )
  })

  it('翻译中的空字符串被过滤', () => {
    expect(formatMeanings([{ pos: 'noun', translations: ['', '歧义'] }])).toBe(
      'noun. 歧义',
    )
  })
})
