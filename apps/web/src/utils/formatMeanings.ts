/**
 * 把结构化释义（dictionary/data-model.md §4 的 Meaning 形态）格式化为单行展示文本：
 * 「adjective. 模棱两可的；含糊不清的; noun. 歧义」。义项之间用「; 」分隔，
 * 与义项内翻译的「；」区分。入参用结构化局部类型，utils 层不依赖 api-client。
 */
export interface MeaningLike {
  pos?: string
  translations?: string[] | null
}

export function formatMeanings(
  meanings: readonly MeaningLike[] | null | undefined,
): string {
  if (!meanings) return ''

  const senses: string[] = []
  for (const meaning of meanings) {
    const translations = (meaning.translations ?? [])
      .filter((text) => text.length > 0)
      .join('；')
    const parts: string[] = []
    if (meaning.pos) parts.push(`${meaning.pos}.`)
    if (translations) parts.push(translations)
    const sense = parts.join(' ')
    if (sense.length > 0) senses.push(sense)
  }
  return senses.join('; ')
}
