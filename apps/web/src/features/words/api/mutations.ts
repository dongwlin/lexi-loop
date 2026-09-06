// words Feature 的 Mutation（前端应用架构规范 §8.3）：写操作成功后精确失效受影响的
// Query，页面通过 Mutation 状态反馈，不自行维护远端数据副本。
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import {
  deleteWord,
  importWords,
  updateWordReviewMeaning,
} from '@lexi-loop/api-client'
import type { ImportWordsRequest, Meaning } from '@lexi-loop/api-client'

import { wordsKeys } from './keys'

// 导入响应只含聚合统计（created / updated / encounters），拿不到受影响 id，
// 且重复导入会累计已有单词的 encounterCount → 全量失效 words 下的列表与详情。
export function useImportWordsMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: ImportWordsRequest) => importWords(request),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: wordsKeys.all }),
  })
}

export interface UpdateReviewMeaningVariables {
  id: string
  /** null 表示清除自定义释义（D007 三层取值回退 review ?? raw） */
  customReviewMeaning: Meaning[] | null
}

// 自定义释义参与三层取值（D007），列表与详情的 effectiveReviewMeaning 都随之变化。
export function useUpdateReviewMeaningMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, customReviewMeaning }: UpdateReviewMeaningVariables) =>
      updateWordReviewMeaning(id, { customReviewMeaning }),
    onSuccess: (_data, { id }) => {
      void queryClient.invalidateQueries({ queryKey: wordsKeys.detail(id) })
      void queryClient.invalidateQueries({ queryKey: wordsKeys.lists() })
    },
  })
}

// 软删除（D004）：删除后 GET 详情会 404，用 remove 清掉缓存条目而非 invalidate。
export function useDeleteWordMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteWord(id),
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: wordsKeys.detail(id) })
      void queryClient.invalidateQueries({ queryKey: wordsKeys.lists() })
    },
  })
}
