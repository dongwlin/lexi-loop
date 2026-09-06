import { describe, expect, it } from 'vitest'

import { wordsKeys } from './keys'

describe('wordsKeys 层级', () => {
  it('list 键挂在 lists 之下，lists 挂在 all 之下', () => {
    const list = wordsKeys.list({ page: 2, pageSize: 20, search: 'amb' })

    expect(list.slice(0, 2)).toEqual(wordsKeys.lists())
    expect(wordsKeys.lists().slice(0, 1)).toEqual(wordsKeys.all)
  })

  it('detail 键挂在 details 之下，details 挂在 all 之下', () => {
    const detail = wordsKeys.detail('w-1')

    expect(detail.slice(0, 2)).toEqual(wordsKeys.details())
    expect(wordsKeys.details().slice(0, 1)).toEqual(wordsKeys.all)
  })

  it('list 与 detail 互不重叠，all 失效同时覆盖两者', () => {
    const list = wordsKeys.list({ page: 1 })
    const detail = wordsKeys.detail('w-1')

    expect(list).not.toEqual(detail)
    expect(list.slice(0, 1)).toEqual(wordsKeys.all)
    expect(detail.slice(0, 1)).toEqual(wordsKeys.all)
  })

  it('不同参数得到不同 list 键', () => {
    expect(wordsKeys.list({ page: 1 })).not.toEqual(wordsKeys.list({ page: 2 }))
  })
})
