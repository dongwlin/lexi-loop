import { describe, expect, it } from 'vitest'
import { formatMeaningText, parseMeaningText } from './meaningText'

describe('formatMeaningText', () => {
  it('空入参返回空串', () => {
    expect(formatMeaningText(null)).toBe('')
    expect(formatMeaningText(undefined)).toBe('')
    expect(formatMeaningText([])).toBe('')
  })

  it('带词性的单义项输出「pos. 译文」', () => {
    expect(
      formatMeaningText([
        { pos: 'adjective', translations: ['模棱两可的', '含糊不清的'] },
      ]),
    ).toBe('adjective. 模棱两可的；含糊不清的')
  })

  it('多个义项逐行输出', () => {
    expect(
      formatMeaningText([
        { pos: 'adjective', translations: ['模棱两可的'] },
        { pos: 'noun', translations: ['歧义'] },
      ]),
    ).toBe('adjective. 模棱两可的\nnoun. 歧义')
  })

  it('无词性的义项省略前缀', () => {
    expect(formatMeaningText([{ translations: ['推导', '获得'] }])).toBe(
      '推导；获得',
    )
  })

  it('过滤空串译文，无译文的义项整条跳过', () => {
    expect(
      formatMeaningText([
        { pos: 'noun' },
        { translations: ['', '限制'] },
      ]),
    ).toBe('限制')
  })
})

describe('parseMeaningText', () => {
  it('空文本与纯空白返回空数组', () => {
    expect(parseMeaningText('')).toEqual([])
    expect(parseMeaningText('  \n \r\n\t')).toEqual([])
  })

  it('无词性行整体作为译文', () => {
    expect(parseMeaningText('模棱两可的')).toEqual([
      { translations: ['模棱两可的'] },
    ])
  })

  it('「pos. 译文」拆出词性，全角分号拆译文', () => {
    expect(parseMeaningText('adj. 模棱两可的；含糊不清的')).toEqual([
      { pos: 'adj', translations: ['模棱两可的', '含糊不清的'] },
    ])
  })

  it('半角分号同样拆分译文并修剪空白', () => {
    expect(parseMeaningText('推导; 获得')).toEqual([
      { translations: ['推导', '获得'] },
    ])
  })

  it('多行拆为多个义项，空行与 CRLF 兜底', () => {
    expect(parseMeaningText('adj. 模棱两可的\r\n\n  \n推导；获得\n')).toEqual([
      { pos: 'adj', translations: ['模棱两可的'] },
      { translations: ['推导', '获得'] },
    ])
  })

  it('数字与含点前缀不误判为词性', () => {
    expect(parseMeaningText('3.14 圆周率')).toEqual([
      { translations: ['3.14 圆周率'] },
    ])
    expect(parseMeaningText('U.S. 经济')).toEqual([
      { translations: ['U.S. 经济'] },
    ])
  })

  it('「.」后无空格不视为词性前缀', () => {
    expect(parseMeaningText('v.跑')).toEqual([{ translations: ['v.跑'] }])
  })

  it('「n.」整行按译文兜底，纯分隔行丢弃', () => {
    expect(parseMeaningText('n.')).toEqual([{ translations: ['n.'] }])
    expect(parseMeaningText('adj. ；')).toEqual([])
  })

  it('与 formatMeaningText 往返一致', () => {
    const samples = [
      [
        { pos: 'adjective', translations: ['模棱两可的', '含糊不清的'] },
        { pos: 'noun', translations: ['歧义'] },
      ],
      [{ translations: ['推导', '获得'] }],
      [],
    ]
    for (const sample of samples) {
      expect(parseMeaningText(formatMeaningText(sample))).toEqual(sample)
    }
  })
})
