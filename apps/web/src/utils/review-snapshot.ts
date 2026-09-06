// 复习进行中的本地快照存取（review-flow.md §6 / §9 / §10；D010 active session 恢复）。
// 服务端无 abandon 端点，「放弃本轮」只清除本地快照；下一轮开始时服务端自动 abandon 旧 session。
// autoResume 是一次性标记：结果页「再来一轮」开始新一轮后置 true，复习页据此跳过
// 「继续 / 放弃」询问直接进入（用户刚显式开始本轮）；进入时即消费置回 false，
// 之后的刷新 / 重进恢复仍由用户选择（D010）。
// localStorage 不可用时（隐私模式 / 配额）放弃持久化，功能不受影响。

import { localStorageAdapter } from '@/lib/storage/local-storage'
import type { SessionWordItem } from '@lexi-loop/api-client'

const STORAGE_KEY = 'lexi-loop.review-snapshot'

export interface ReviewSnapshot {
  sessionId: string
  totalCount: number
  items: SessionWordItem[]
  autoResume?: boolean
}

const schemaVersion = 1

export function saveReviewSnapshot(snapshot: ReviewSnapshot): void {
  const payload = JSON.stringify({
    version: schemaVersion,
    ...snapshot,
  })
  localStorageAdapter.set(STORAGE_KEY, payload)
}

export function loadReviewSnapshot(): ReviewSnapshot | null {
  try {
    const raw = localStorageAdapter.get(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Record<string, unknown>
    if (
      parsed.version !== schemaVersion ||
      typeof parsed.sessionId !== 'string' ||
      typeof parsed.totalCount !== 'number' ||
      !Array.isArray(parsed.items) ||
      !parsed.items.every(isValidSnapshotItem)
    ) {
      return null
    }
    return {
      sessionId: parsed.sessionId,
      totalCount: parsed.totalCount,
      items: parsed.items as SessionWordItem[],
      ...(typeof parsed.autoResume === 'boolean' && parsed.autoResume
        ? { autoResume: true }
        : {}),
    }
  } catch {
    return null
  }
}

// 快照 items 至少校验渲染与提交依赖的字段（itemId / word），畸形数据整体回退 null。
function isValidSnapshotItem(item: unknown): boolean {
  return (
    typeof item === 'object' &&
    item !== null &&
    typeof (item as Record<string, unknown>).itemId === 'string' &&
    typeof (item as Record<string, unknown>).word === 'string'
  )
}

export function clearReviewSnapshot(): void {
  localStorageAdapter.remove(STORAGE_KEY)
}
