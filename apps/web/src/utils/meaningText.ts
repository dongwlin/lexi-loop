/**
 * 释义编辑器的文本 ⇄ 结构化释义编解码（review-flow §5 编辑复习释义）。
 * 文本格式：每行一条义项，「pos. 译文一；译文二」——词性前缀可省略，义项内译文用
 * 全角「；」或半角「;」分隔，义项间用换行分隔。与展示层的单行 formatMeanings 区分：
 * 展示用「; 」连接义项、编辑器用换行，两者不混用。入参为结构化局部类型，utils 层
 * 不依赖 api-client（与 formatMeanings 同口径）。
 */
import type { MeaningLike } from './formatMeanings'

export function formatMeaningText(
  meanings: readonly MeaningLike[] | null | undefined,
): string {
  if (!meanings) return ''

  const lines: string[] = []
  for (const meaning of meanings) {
    const translations = (meaning.translations ?? [])
      .filter((text) => text.length > 0)
      .join('；')
    // 没有译文的义项（如纯词性占位）不落文本：parse 无法还原它，跳过保证往返一致。
    if (!translations) continue
    lines.push(meaning.pos ? `${meaning.pos}. ${translations}` : translations)
  }
  return lines.join('\n')
}

// 词性前缀：字母开头、可含 & / -（如 vt&vi），后接「. 」且译文非空才视作词性。
// 不允许前缀内出现「.」，且「.」后必须有空白——「3.14 圆周率」「U.S. 经济」
// 整体按译文处理，不误拆词性。
const POS_PREFIX = /^([A-Za-z][A-Za-z&/-]*)\.\s+(.+)$/

export function parseMeaningText(text: string): MeaningLike[] {
  const senses: MeaningLike[] = []
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line) continue

    const match = POS_PREFIX.exec(line)
    const body = (match?.[2] ?? line).trim()
    const translations = body
      .split(/[;；]/)
      .map((item) => item.trim())
      .filter((item) => item.length > 0)

    // 声明了词性但没有任何译文（如「adj. ；」）不算一条义项，直接丢弃；
    // 「n.」这类整体无前缀结构的行按译文兜底保留，不静默丢弃用户输入。
    if (translations.length === 0) continue
    senses.push(match ? { pos: match[1], translations } : { translations })
  }
  return senses
}
