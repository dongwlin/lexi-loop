import { QueryClient } from '@tanstack/vue-query'

// 全局默认保持保守（前端技术栈 §7.3）；各 Feature 按数据性质覆盖 staleTime、
// 重试与聚焦刷新。认证类错误由 API Client 处理，不靠 Query 通用重试。
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 30 * 60_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})
