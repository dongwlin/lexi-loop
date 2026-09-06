// review Feature 的 Query（前端应用架构规范 §8.1）：DTO 直接使用 api-client 生成类型，
// queryFn 透传 signal；active session 恢复（D010）与结果页汇总共用同一查询。
import { useQuery } from '@tanstack/vue-query'
import { getReviewSession } from '@lexi-loop/api-client'
import type { MaybeRefOrGetter } from 'vue'
import { computed, toValue } from 'vue'

import { reviewKeys } from './keys'

export function useReviewSessionQuery(id: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => reviewKeys.session(toValue(id))),
    queryFn: ({ signal }) => getReviewSession(toValue(id), { signal }),
  })
}
