// 复习数量选择的本地持久化（review-flow.md §6）：预设数量与「自定义」是同一组
// 选项，上次选择——包括自定义模式与输入的数量——在下次进入复习页时恢复。
// 无记录或数据非法一律回退默认值；localStorage 不可用时（隐私模式 / 配额）放弃
// 持久化，功能不受影响。

import { localStorageAdapter } from '@/lib/storage/local-storage'

export const PRESET_COUNTS = [10, 20, 30, 50] as const

export const DEFAULT_REVIEW_COUNT = 30

const STORAGE_KEY = 'lexi-loop.review-count'

export type ReviewCountPreference =
  { mode: 'preset'; count: number } | { mode: 'custom'; count: number }

const schemaVersion = 1

const DEFAULT_PREFERENCE: ReviewCountPreference = {
  mode: 'preset',
  count: DEFAULT_REVIEW_COUNT,
}

function isPresetCount(value: number): value is (typeof PRESET_COUNTS)[number] {
  return (PRESET_COUNTS as readonly number[]).includes(value)
}

export function saveReviewCountPreference(
  preference: ReviewCountPreference,
): void {
  const payload = JSON.stringify({ version: schemaVersion, ...preference })
  localStorageAdapter.set(STORAGE_KEY, payload)
}

export function loadReviewCountPreference(): ReviewCountPreference {
  try {
    const raw = localStorageAdapter.get(STORAGE_KEY)
    if (!raw) return DEFAULT_PREFERENCE
    const parsed = JSON.parse(raw) as Record<string, unknown>
    if (parsed.version !== schemaVersion || typeof parsed.count !== 'number') {
      return DEFAULT_PREFERENCE
    }
    if (parsed.mode === 'preset' && isPresetCount(parsed.count)) {
      return { mode: 'preset', count: parsed.count }
    }
    if (
      parsed.mode === 'custom' &&
      Number.isSafeInteger(parsed.count) &&
      parsed.count >= 1
    ) {
      return { mode: 'custom', count: parsed.count }
    }
    return DEFAULT_PREFERENCE
  } catch {
    return DEFAULT_PREFERENCE
  }
}
