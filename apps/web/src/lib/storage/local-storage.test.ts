import { afterEach, describe, expect, it, vi } from 'vitest'
import { localStorageAdapter } from './local-storage'

function memoryStorage(initial: Record<string, string> = {}): Storage {
  const map = new Map(Object.entries(initial))
  return {
    get length() {
      return map.size
    },
    clear: () => map.clear(),
    getItem: (key) => map.get(key) ?? null,
    key: (index) => [...map.keys()][index] ?? null,
    removeItem: (key) => {
      map.delete(key)
    },
    setItem: (key, value) => {
      map.set(key, value)
    },
  }
}

describe('localStorageAdapter', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('get / set / remove 委托给底层 localStorage', () => {
    const storage = memoryStorage()
    vi.stubGlobal('localStorage', storage)

    expect(localStorageAdapter.get('missing')).toBeNull()
    localStorageAdapter.set('k', 'v')
    expect(storage.getItem('k')).toBe('v')
    localStorageAdapter.remove('k')
    expect(storage.getItem('k')).toBeNull()
  })

  it('localStorage 不可用时 get 返回 null、set / remove 不抛出', () => {
    vi.stubGlobal('localStorage', undefined)

    expect(() => localStorageAdapter.set('k', 'v')).not.toThrow()
    expect(() => localStorageAdapter.remove('k')).not.toThrow()
    expect(localStorageAdapter.get('k')).toBeNull()
  })

  it('底层抛出的异常被吞掉（如配额失败 / 安全限制）', () => {
    const throwing = {
      length: 0,
      clear: () => undefined,
      getItem: () => {
        throw new DOMException('denied')
      },
      key: () => null,
      removeItem: () => {
        throw new DOMException('denied')
      },
      setItem: () => {
        throw new DOMException('quota exceeded')
      },
    } as Storage
    vi.stubGlobal('localStorage', throwing)

    expect(() => localStorageAdapter.set('k', 'v')).not.toThrow()
    expect(() => localStorageAdapter.remove('k')).not.toThrow()
    expect(localStorageAdapter.get('k')).toBeNull()
  })
})
