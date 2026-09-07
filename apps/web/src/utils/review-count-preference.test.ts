import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  DEFAULT_REVIEW_COUNT,
  loadReviewCountPreference,
  saveReviewCountPreference,
} from './review-count-preference'

function seedStorage(value: unknown): void {
  vi.stubGlobal(
    'localStorage',
    value === undefined
      ? undefined
      : {
          getItem: () =>
            typeof value === 'string' ? value : JSON.stringify(value),
          setItem: () => undefined,
          removeItem: () => undefined,
        },
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('loadReviewCountPreference', () => {
  it('无记录时回退默认预设数量', () => {
    vi.stubGlobal('localStorage', {
      getItem: () => null,
      setItem: () => undefined,
      removeItem: () => undefined,
    })

    expect(loadReviewCountPreference()).toEqual({
      mode: 'preset',
      count: DEFAULT_REVIEW_COUNT,
    })
  })

  it('恢复上次选中的预设数量', () => {
    seedStorage({ version: 1, mode: 'preset', count: 20 })

    expect(loadReviewCountPreference()).toEqual({ mode: 'preset', count: 20 })
  })

  it('恢复自定义模式与上次输入的数量', () => {
    seedStorage({ version: 1, mode: 'custom', count: 25 })

    expect(loadReviewCountPreference()).toEqual({ mode: 'custom', count: 25 })
  })

  it.each([
    ['版本不匹配', { version: 2, mode: 'preset', count: 20 }],
    ['缺少数量字段', { version: 1, mode: 'preset' }],
    ['预设值不在预设列表内', { version: 1, mode: 'preset', count: 25 }],
    ['自定义数量小于 1', { version: 1, mode: 'custom', count: 0 }],
    ['自定义数量非整数', { version: 1, mode: 'custom', count: 2.5 }],
    ['未知模式', { version: 1, mode: 'auto', count: 20 }],
  ])('%s 时回退默认值', (_name, stored) => {
    seedStorage(stored)

    expect(loadReviewCountPreference()).toEqual({
      mode: 'preset',
      count: DEFAULT_REVIEW_COUNT,
    })
  })

  it('非法 JSON 回退默认值', () => {
    seedStorage('{not json')

    expect(loadReviewCountPreference()).toEqual({
      mode: 'preset',
      count: DEFAULT_REVIEW_COUNT,
    })
  })

  it('localStorage 不可用时回退默认值且不抛出', () => {
    seedStorage(undefined)

    expect(loadReviewCountPreference()).toEqual({
      mode: 'preset',
      count: DEFAULT_REVIEW_COUNT,
    })
  })
})

describe('saveReviewCountPreference', () => {
  it('按版本化格式写入预设与自定义选择', () => {
    const written: string[] = []
    vi.stubGlobal('localStorage', {
      getItem: () => null,
      setItem: (_key: string, value: string) => {
        written.push(value)
      },
      removeItem: () => undefined,
    })

    saveReviewCountPreference({ mode: 'preset', count: 50 })
    saveReviewCountPreference({ mode: 'custom', count: 25 })

    expect(written).toEqual([
      JSON.stringify({ version: 1, mode: 'preset', count: 50 }),
      JSON.stringify({ version: 1, mode: 'custom', count: 25 }),
    ])
  })

  it('localStorage 不可用时不抛出（放弃持久化）', () => {
    seedStorage(undefined)

    expect(() =>
      saveReviewCountPreference({ mode: 'custom', count: 25 }),
    ).not.toThrow()
  })
})
