// words Feature 的 Query Key 工厂（前端应用架构规范 §8.1）：集中定义缓存键层级，
// 查询与 Mutation 失效共用，避免页面散落字符串键。层级约定：
// ['words'] → ['words', 'list', params] / ['words', 'detail', id]。
import type { ListWordsParams } from '@lexi-loop/api-client'

export const wordsKeys = {
  all: ['words'] as const,
  lists: () => [...wordsKeys.all, 'list'] as const,
  list: (params: ListWordsParams) => [...wordsKeys.lists(), params] as const,
  details: () => [...wordsKeys.all, 'detail'] as const,
  detail: (id: string) => [...wordsKeys.details(), id] as const,
}
