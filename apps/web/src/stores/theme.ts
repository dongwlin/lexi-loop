import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { localStorageAdapter } from '@/lib/storage/local-storage'

export type ThemePreference = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'lexi-loop.theme'

interface StoredThemeV1 {
  version: 1
  preference: ThemePreference
}

function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

function isStoredThemeV1(value: unknown): value is StoredThemeV1 {
  if (typeof value !== 'object' || value === null) {
    return false
  }
  const record = value as Record<string, unknown>
  return record.version === 1 && isThemePreference(record.preference)
}

// 存储格式 {"version":1,"preference":"..."}；缺失或非法一律回退 'system'。
function readStoredPreference(): ThemePreference {
  const raw = localStorageAdapter.get(STORAGE_KEY)
  if (raw === null) {
    return 'system'
  }
  try {
    const parsed: unknown = JSON.parse(raw)
    if (isStoredThemeV1(parsed)) {
      return parsed.preference
    }
  } catch {
    // 非法 JSON：按无记录处理。
  }
  return 'system'
}

// 主题类挂在 html 上且互斥（设计系统方案 §11），保证挂到 body 的 Portal 继承同一主题；
// color-scheme 由 CSS 侧的 .light / .dark 作用域同步。
function applyResolvedTheme(resolved: ResolvedTheme): void {
  if (typeof document === 'undefined') {
    return
  }
  const root = document.documentElement
  root.classList.toggle('light', resolved === 'light')
  root.classList.toggle('dark', resolved === 'dark')
}

function systemPrefersDark(): ResolvedTheme {
  if (typeof globalThis.matchMedia !== 'function') {
    return 'light'
  }
  return globalThis.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export const useThemeStore = defineStore('theme', () => {
  const preference = ref<ThemePreference>('system')
  const systemTheme = ref<ResolvedTheme>('light')
  const resolvedTheme = computed<ResolvedTheme>(() =>
    preference.value === 'system' ? systemTheme.value : preference.value,
  )

  // 仅 system 偏好跟随系统变化；显式 light / dark 不被系统变化覆盖（设计系统方案 §11）。
  // 监听器随应用生命周期存活，无需摘除。
  function watchSystemTheme(): void {
    if (typeof globalThis.matchMedia !== 'function') {
      return
    }
    const query = globalThis.matchMedia('(prefers-color-scheme: dark)')
    query.addEventListener('change', (event: MediaQueryListEvent) => {
      systemTheme.value = event.matches ? 'dark' : 'light'
      applyResolvedTheme(resolvedTheme.value)
    })
  }

  // 在 pinia 安装后、应用挂载前调用一次：恢复偏好并把主题类应用到 html（设计系统方案 §11）。
  function init(): void {
    preference.value = readStoredPreference()
    systemTheme.value = systemPrefersDark()
    watchSystemTheme()
    applyResolvedTheme(resolvedTheme.value)
  }

  function setPreference(next: ThemePreference): void {
    preference.value = next
    const stored: StoredThemeV1 = { version: 1, preference: next }
    localStorageAdapter.set(STORAGE_KEY, JSON.stringify(stored))
    applyResolvedTheme(resolvedTheme.value)
  }

  return { preference, resolvedTheme, init, setPreference }
})
