// review Feature 的 Mutation（前端应用架构规范 §8.3）。
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import {
  startReviewSession,
  submitReviewResult,
} from '@lexi-loop/api-client'
import type { StartSessionRequest, SubmitResultRequestResult } from '@lexi-loop/api-client'

import { reviewKeys } from './keys'

// 开始新一轮时服务端会自动 abandon 旧 active session（D010）；
// 返回的 items 由页面直接消费并落地本地快照，无需失效既有缓存。
export function useStartReviewSessionMutation() {
  return useMutation({
    mutationFn: (request: StartSessionRequest) => startReviewSession(request),
  })
}

export interface SubmitReviewResultVariables {
  sessionId: string
  itemId: string
  result: SubmitResultRequestResult
}

// 提交幂等（D005）；失效 session 汇总，完成后的结果页取最新 remembered / forgotten。
export function useSubmitReviewResultMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sessionId, itemId, result }: SubmitReviewResultVariables) =>
      submitReviewResult(sessionId, itemId, { result }),
    onSuccess: (_data, { sessionId }) =>
      queryClient.invalidateQueries({ queryKey: reviewKeys.session(sessionId) }),
  })
}
