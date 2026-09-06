// words Feature 的 Query（前端应用架构规范 §8.1）：DTO 直接使用 api-client 生成类型，
// queryFn 透传 signal 以接入全局超时与组件卸载取消；列表参数响应式（URL 搜索 / 分页），
// 参数变化经 queryKey 自动重新取数。
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { getWord, listWords } from '@lexi-loop/api-client'
import type { ListWordsParams } from '@lexi-loop/api-client'
import type { MaybeRefOrGetter } from 'vue'
import { computed, toValue } from 'vue'

import { wordsKeys } from './keys'

export function useWordsListQuery(params: MaybeRefOrGetter<ListWordsParams>) {
  return useQuery({
    queryKey: computed(() => wordsKeys.list(toValue(params))),
    queryFn: ({ signal }) => listWords(toValue(params), { signal }),
    // 翻页 / 修改搜索时保留上一份数据，表格不闪回 Loading 骨架，isFetching 标记刷新中。
    placeholderData: keepPreviousData,
  })
}

export function useWordDetailQuery(id: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => wordsKeys.detail(toValue(id))),
    queryFn: ({ signal }) => getWord(toValue(id), { signal }),
  })
}
