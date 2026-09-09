import { shallowRef, watch } from 'vue'
import { localStorageAdapter } from '@/lib/storage/local-storage'

const STORAGE_KEY = 'lexi-loop.review.autoPlayPronunciation'

function loadAutoPlay(): boolean {
  if (typeof window === 'undefined') return true
  try {
    const value: unknown = JSON.parse(
      localStorageAdapter.get(STORAGE_KEY) ?? 'true',
    )
    return typeof value === 'boolean' ? value : true
  } catch {
    return true
  }
}

/** 页面实例持有设置；存储异常由统一适配器降级，不影响本次复习。 */
export function useReviewSettings() {
  const autoPlayPronunciation = shallowRef(loadAutoPlay())
  watch(
    autoPlayPronunciation,
    (value) => {
      if (typeof window !== 'undefined') {
        localStorageAdapter.set(STORAGE_KEY, JSON.stringify(value))
      }
    },
    { flush: 'sync' },
  )
  return { autoPlayPronunciation }
}
