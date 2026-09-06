/**
 * 解析用户粘贴的多行文本，标准化后按单词聚合出现次数。
 *
 * 规则（review-flow.md §2）：trim → lowercase → 移除空行 → 重复单词聚合成 count 不丢弃。
 * 保留首次出现顺序，供后续提交 `importWords` 时使用。
 */
export interface ParsedWord {
  /** 标准化后的单词（小写、无首尾空白） */
  word: string
  /** 在用户输入中出现的次数 */
  count: number
}

export function parseImportText(raw: string): ParsedWord[] {
  const lines = raw.split('\n')

  const seen = new Map<string, ParsedWord>()

  for (const line of lines) {
    const word = line.trim().toLowerCase()
    if (word === '') continue

    const existing = seen.get(word)
    if (existing) {
      existing.count++
    } else {
      seen.set(word, { word, count: 1 })
    }
  }

  return [...seen.values()]
}
