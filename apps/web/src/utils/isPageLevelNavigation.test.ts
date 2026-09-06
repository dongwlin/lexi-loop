import { describe, expect, it } from 'vitest'

import { isPageLevelNavigation } from './isPageLevelNavigation'

const at = (path: string) => ({ path, matched: [{}] })

describe('isPageLevelNavigation', () => {
  it('初次加载（from 为 START_LOCATION）不移动焦点', () => {
    expect(isPageLevelNavigation(at('/review'), { path: '/', matched: [] })).toBe(false)
  })

  it('同路径仅 query / hash 变化（搜索、分页、筛选）不移动焦点', () => {
    expect(isPageLevelNavigation(at('/words'), at('/words'))).toBe(false)
  })

  it('路径变化是页面级导航', () => {
    expect(isPageLevelNavigation(at('/import'), at('/words'))).toBe(true)
  })

  it('同一路由不同参数（如单词详情换 id）是页面级导航', () => {
    expect(isPageLevelNavigation(at('/words/b'), at('/words/a'))).toBe(true)
  })
})
