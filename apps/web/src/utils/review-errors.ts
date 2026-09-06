/**
 * 开始复习失败的用户可读文案映射（review-flow §6 / §9）：服务端 message 面向排查
 * （英文原文），界面按错误分类给文案。复习页开始与结果页「再来一轮」共用同一映射。
 * 结构化局部类型判定 ApiError 形态，utils 层不依赖 api-client（同 formatMeanings）。
 */
interface ApiErrorLike {
  kind?: unknown
  code?: unknown
  httpStatus?: unknown
}

function asApiErrorLike(err: unknown): ApiErrorLike | null {
  if (typeof err !== 'object' || err === null || !('kind' in err)) return null
  return err
}

export function describeReviewStartError(err: unknown): string {
  const apiError = asApiErrorLike(err)
  if (apiError) {
    if (apiError.code === 'BASE.BIZ.USER_DISABLED')
      return '当前没有可复习的生词，请先导入生词。'
    if (apiError.kind === 'network' || apiError.kind === 'timeout')
      return '网络异常，请检查连接后重试。'
    if (
      apiError.kind === 'http' &&
      typeof apiError.httpStatus === 'number' &&
      apiError.httpStatus >= 500
    )
      return '服务暂时不可用，请稍后重试。'
  }
  return '开始复习失败，请稍后重试'
}
