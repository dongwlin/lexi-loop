/**
 * 分页页码窗口（生词库分页导航用）：始终保留第 1 页与最后一页，当前页两侧各留 1 页，
 * 与端点之间的连续缺口折叠为省略号；总页数不超过 7 时全部展开。
 * current 越界时钳制到有效范围，调用方无需预处理。
 */
export type PageItem = { kind: 'page'; page: number } | { kind: 'ellipsis' }

const MAX_EXPANDED_PAGES = 7

export function buildPageItems(current: number, total: number): PageItem[] {
  if (total <= 0) return []

  const page = Math.min(Math.max(1, Math.trunc(current)), total)
  if (total <= MAX_EXPANDED_PAGES) {
    return Array.from({ length: total }, (_, index) => ({
      kind: 'page',
      page: index + 1,
    }))
  }

  const start = Math.max(2, page - 1)
  const end = Math.min(total - 1, page + 1)

  const items: PageItem[] = [{ kind: 'page', page: 1 }]
  if (start > 2) items.push({ kind: 'ellipsis' })
  for (let p = start; p <= end; p++) {
    items.push({ kind: 'page', page: p })
  }
  if (end < total - 1) items.push({ kind: 'ellipsis' })
  items.push({ kind: 'page', page: total })
  return items
}
