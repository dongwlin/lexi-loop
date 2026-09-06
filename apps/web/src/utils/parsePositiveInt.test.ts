import { describe, expect, it } from 'vitest'

import { parsePositiveInt } from './parsePositiveInt'

describe('parsePositiveInt', () => {
  it('合法正整数字符串正常解析', () => {
    expect(parsePositiveInt('1', 9)).toBe(1)
    expect(parsePositiveInt('42', 9)).toBe(42)
  })

  it('数组取首项（vue-router 重复 query 参数形态）', () => {
    expect(parsePositiveInt(['3', '7'], 9)).toBe(3)
  })

  it('数组首项缺失时回退默认值', () => {
    expect(parsePositiveInt(undefined, 9)).toBe(9)
    expect(parsePositiveInt(null, 9)).toBe(9)
    expect(parsePositiveInt([null, '2'], 9)).toBe(9)
    expect(parsePositiveInt(3, 9)).toBe(9)
  })

  it('非数字与空串回退默认值', () => {
    expect(parsePositiveInt('abc', 9)).toBe(9)
    expect(parsePositiveInt('', 9)).toBe(9)
  })

  it('小数、零与负数回退默认值', () => {
    expect(parsePositiveInt('2.5', 9)).toBe(9)
    expect(parsePositiveInt('0', 9)).toBe(9)
    expect(parsePositiveInt('-1', 9)).toBe(9)
  })
})
