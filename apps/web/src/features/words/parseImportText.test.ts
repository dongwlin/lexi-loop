import { describe, expect, it } from 'vitest'

import { parseImportText } from './parseImportText'

describe('parseImportText', () => {
  it('空输入返回空数组', () => {
    expect(parseImportText('')).toEqual([])
  })

  it('只有空行时返回空数组', () => {
    expect(parseImportText('\n\n\n')).toEqual([])
  })

  it('解析单个单词', () => {
    expect(parseImportText('ambiguous')).toEqual([
      { word: 'ambiguous', count: 1 },
    ])
  })

  it('trim 首尾空白并 lowercase', () => {
    expect(parseImportText('  Constrain  ')).toEqual([
      { word: 'constrain', count: 1 },
    ])
  })

  it('移除空行', () => {
    expect(parseImportText('ambiguous\n\n\nconstrain\n')).toEqual([
      { word: 'ambiguous', count: 1 },
      { word: 'constrain', count: 1 },
    ])
  })

  it('重复单词聚合 count', () => {
    expect(
      parseImportText('ambiguous\nconstrain\nderive\nconstrain\nsubtle'),
    ).toEqual([
      { word: 'ambiguous', count: 1 },
      { word: 'constrain', count: 2 },
      { word: 'derive', count: 1 },
      { word: 'subtle', count: 1 },
    ])
  })

  it('大小写不敏感合并', () => {
    expect(parseImportText('Ambiguous\nAMBIGUOUS\nambiguous')).toEqual([
      { word: 'ambiguous', count: 3 },
    ])
  })

  it('保留首次出现顺序', () => {
    const result = parseImportText('c\na\nb\na\nc')
    expect(result.map((w) => w.word)).toEqual(['c', 'a', 'b'])
  })

  it('末尾换行不影响结果', () => {
    expect(parseImportText('a\nb\n')).toEqual([
      { word: 'a', count: 1 },
      { word: 'b', count: 1 },
    ])
  })
})
