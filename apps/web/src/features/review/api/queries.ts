// review Feature 的 Query（前端应用架构规范 §8.1）：DTO 直接使用 api-client 生成类型，
// queryFn 透传 signal；active session 恢复（D010）与结果页汇总共用同一查询。
import { useQuery } from '@tanstack/vue-query'
import { getReviewSession } from '@lexi-loop/api-client'
import type { MaybeRefOrGetter } from 'vue'
import { computed, toValue } from 'vue'

import { reviewKeys } from './keys'

// 命令式取数（如复习页 active session 恢复检查）与 useReviewSessionQuery 共用同一
// key / queryFn，保证缓存与失效口径一致；调用方可用 staleTime 覆盖按需取新。
export function reviewSessionQueryOptions(id: string) {
  return {
    queryKey: reviewKeys.session(id),
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      getReviewSession(id, { signal }),
  }
}

export function useReviewSessionQuery(id: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => reviewKeys.session(toValue(id))),
    queryFn: ({ signal }) => getReviewSession(toValue(id), { signal }),
  })
}
