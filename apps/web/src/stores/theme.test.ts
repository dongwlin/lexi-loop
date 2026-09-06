import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useThemeStore } from './theme'

const STORAGE_KEY = 'lexi-loop.theme'

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

/** node 环境下 document / matchMedia 均不存在，用最小桩替换。 */
function stubDocument(): Set<string> {
  const classes = new Set<string>()
  vi.stubGlobal('document', {
    documentElement: {
      classList: {
        contains: (name: string) => classes.has(name),
        toggle: (name: string, force?: boolean) => {
          const next = force ?? !classes.has(name)
          if (next) {
            classes.add(name)
          } else {
            classes.delete(name)
          }
          return next
        },
      },
    },
  })
  return classes
}

function stubMatchMedia(initialMatches: boolean) {
  let matches = initialMatches
  const listeners = new Set<(event: { matches: boolean }) => void>()
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({
      get matches() {
        return matches
      },
      addEventListener: (_type: string, listener: (event: { matches: boolean }) => void) => {
        listeners.add(listener)
      },
      removeEventListener: (_type: string, listener: (event: { matches: boolean }) => void) => {
        listeners.delete(listener)
      },
    })),
  )
  return {
    emit(next: boolean) {
      matches = next
      for (const listener of listeners) {
        listener({ matches: next })
      }
    },
  }
}

describe('theme store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('无存储记录时回退 system 并应用当前系统主题', () => {
    const classes = stubDocument()
    stubMatchMedia(false)
    vi.stubGlobal('localStorage', memoryStorage())

    const store = useThemeStore()
    store.init()

    expect(store.preference).toBe('system')
    expect(store.resolvedTheme).toBe('light')
    expect(classes.has('light')).toBe(true)
    expect(classes.has('dark')).toBe(false)
  })

  it('恢复持久化的显式偏好，不被系统偏好覆盖', () => {
    const classes = stubDocument()
    stubMatchMedia(true)
    vi.stubGlobal(
      'localStorage',
      memoryStorage({ [STORAGE_KEY]: '{"version":1,"preference":"light"}' }),
    )

    const store = useThemeStore()
    store.init()

    expect(store.preference).toBe('light')
    expect(store.resolvedTheme).toBe('light')
    expect(classes.has('light')).toBe(true)
    expect(classes.has('dark')).toBe(false)
  })

  it.each([
    ['非法 JSON', 'not-json'],
    ['版本不识别', '{"version":2,"preference":"dark"}'],
    ['偏好值未知', '{"version":1,"preference":"blue"}'],
    ['结构不是对象', '"dark"'],
  ])('存储内容非法（%s）时回退 system', (_name, raw) => {
    stubDocument()
    stubMatchMedia(false)
    vi.stubGlobal('localStorage', memoryStorage({ [STORAGE_KEY]: raw }))

    const store = useThemeStore()
    store.init()

    expect(store.preference).toBe('system')
    expect(store.resolvedTheme).toBe('light')
  })

  it('setPreference 持久化偏好并应用互斥的主题类', () => {
    const classes = stubDocument()
    stubMatchMedia(false)
    const storage = memoryStorage()
    vi.stubGlobal('localStorage', storage)

    const store = useThemeStore()
    store.init()
    store.setPreference('dark')

    expect(store.resolvedTheme).toBe('dark')
    expect(classes.has('dark')).toBe(true)
    expect(classes.has('light')).toBe(false)
    expect(storage.getItem(STORAGE_KEY)).toBe('{"version":1,"preference":"dark"}')
  })

  it('system 偏好下跟随系统主题变化并重新应用', () => {
    const classes = stubDocument()
    const media = stubMatchMedia(false)
    vi.stubGlobal('localStorage', memoryStorage())

    const store = useThemeStore()
    store.init()
    media.emit(true)

    expect(store.resolvedTheme).toBe('dark')
    expect(classes.has('dark')).toBe(true)
    expect(classes.has('light')).toBe(false)
  })

  it('显式偏好下系统主题变化不影响已解析主题', () => {
    const classes = stubDocument()
    const media = stubMatchMedia(false)
    vi.stubGlobal('localStorage', memoryStorage())

    const store = useThemeStore()
    store.init()
    store.setPreference('light')
    media.emit(true)

    expect(store.resolvedTheme).toBe('light')
    expect(classes.has('light')).toBe(true)
    expect(classes.has('dark')).toBe(false)
  })
})
