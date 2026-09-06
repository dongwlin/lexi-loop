import { describe, expect, it } from 'vitest'

import { buildPageItems } from './buildPageItems'

function pages(current: number, total: number): Array<number | 'ellipsis'> {
  return buildPageItems(current, total).map((item) =>
    item.kind === 'page' ? item.page : 'ellipsis',
  )
}

describe('buildPageItems', () => {
  it('总页数为 0 返回空列表', () => {
    expect(pages(1, 0)).toEqual([])
  })

  it('总页数不超过 7 时全部展开', () => {
    expect(pages(4, 7)).toEqual([1, 2, 3, 4, 5, 6, 7])
  })

  it('第一页折叠右侧缺口', () => {
    expect(pages(1, 22)).toEqual([1, 2, 'ellipsis', 22])
  })

  it('靠前页保留左侧连续区间', () => {
    expect(pages(2, 22)).toEqual([1, 2, 3, 'ellipsis', 22])
    expect(pages(3, 22)).toEqual([1, 2, 3, 4, 'ellipsis', 22])
  })

  it('中间页两侧折叠省略号', () => {
    expect(pages(11, 22)).toEqual([1, 'ellipsis', 10, 11, 12, 'ellipsis', 22])
  })

  it('靠后页与最后页对称', () => {
    expect(pages(21, 22)).toEqual([1, 'ellipsis', 20, 21, 22])
    expect(pages(22, 22)).toEqual([1, 'ellipsis', 21, 22])
  })

  it('current 越界时钳制到有效范围', () => {
    expect(pages(0, 22)).toEqual([1, 2, 'ellipsis', 22])
    expect(pages(99, 22)).toEqual([1, 'ellipsis', 21, 22])
  })
})
