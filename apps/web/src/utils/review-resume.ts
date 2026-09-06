// active session 恢复的进度对齐（review-flow.md §6「回到原进度」；D010）。
// GET /reviews/:id 的逐词结果只有 word + result、没有 itemId，与本地快照按 word 对齐
//（一轮内加权不放回抽样，word 唯一，review-flow.md §8）。服务端结果与本地快照无法
// 可靠对齐时返回 null，由页面回退到第 1 题继续——已答题的重复提交经服务端幂等短路
// 不计入统计（api/reviews.md §3），只是多做无效操作。

export interface ReviewResumeProgress {
  /** 已作答数量（服务端 result !== 'pending' 的条目数）。 */
  answered: number
  /** 本地快照中首个未作答条目的下标；全部已答时为 localWords.length。 */
  firstPendingIndex: number
}

export function computeResumeProgress(
  localWords: readonly string[],
  serverItems: ReadonlyArray<{ word: string; result: string }>,
): ReviewResumeProgress | null {
  if (serverItems.length !== localWords.length) return null

  const serverWords = serverItems.map((item) => item.word)
  // 一轮内 word 应唯一；重复说明数据异常，不强行对齐。
  if (new Set(serverWords).size !== serverWords.length) return null

  const localWordsSet = new Set(localWords)
  if (serverWords.some((word) => !localWordsSet.has(word))) return null

  const pendingWords = new Set(
    serverItems
      .filter((item) => item.result === 'pending')
      .map((item) => item.word),
  )
  const answered = serverItems.length - pendingWords.size
  const firstPendingIndex =
    pendingWords.size === 0
      ? localWords.length
      : localWords.findIndex((word) => pendingWords.has(word))
  return { answered, firstPendingIndex }
}
