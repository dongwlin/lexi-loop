import { effectScope } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useReviewSettings } from './useReviewSettings'

const key = 'lexi-loop.review.autoPlayPronunciation'
const scopes: ReturnType<typeof effectScope>[] = []
function settings() {
  const scope = effectScope()
  scopes.push(scope)
  return scope.run(useReviewSettings)!
}
beforeEach(() => localStorage.clear())
afterEach(() => {
  scopes.forEach((scope) => scope.stop())
  scopes.length = 0
  vi.unstubAllGlobals()
})

describe('复习本地设置', () => {
  it('没有记录默认开启，修改后立即持久化并可重新初始化恢复', () => {
    const first = settings()
    expect(first.autoPlayPronunciation.value).toBe(true)
    first.autoPlayPronunciation.value = false
    expect(localStorage.getItem(key)).toBe('false')
    const second = settings()
    expect(second.autoPlayPronunciation.value).toBe(false)
    second.autoPlayPronunciation.value = true
    expect(localStorage.getItem(key)).toBe('true')
    expect(settings().autoPlayPronunciation.value).toBe(true)
  })
  it.each(['oops', 'null', '0', '"false"', '{}', '[]'])(
    '非法值 %s 默认开启',
    (value) => {
      localStorage.setItem(key, value)
      expect(settings().autoPlayPronunciation.value).toBe(true)
    },
  )
  it('存储读写抛异常时仍可修改内存设置', () => {
    vi.stubGlobal('localStorage', {
      getItem: () => {
        throw new Error('SecurityError')
      },
      setItem: () => {
        throw new Error('QuotaExceededError')
      },
    })
    const state = settings()
    expect(state.autoPlayPronunciation.value).toBe(true)
    expect(() => {
      state.autoPlayPronunciation.value = false
    }).not.toThrow()
    expect(state.autoPlayPronunciation.value).toBe(false)
    expect(settings().autoPlayPronunciation.value).toBe(true)
  })
  it('无 window 或无 localStorage 时安全默认开启', () => {
    vi.stubGlobal('window', undefined)
    vi.stubGlobal('localStorage', undefined)
    const state = settings()
    expect(state.autoPlayPronunciation.value).toBe(true)
    expect(() => {
      state.autoPlayPronunciation.value = false
    }).not.toThrow()
  })
})
