// meta Feature 的 Query（前端应用架构规范 §8.1）：DTO 直接使用 api-client 生成类型，
// queryFn 透传 signal；版本信息以后端为单一来源（docs/api/meta.md），前端不重复注入。
// 词典导入进度是进程内存态（docs/api/meta.md §3），按状态机收敛轮询。
import { useQuery } from '@tanstack/vue-query'
import { getDictionaryImport, getVersion } from '@lexi-loop/api-client'

import { metaKeys } from './keys'

export function useVersionQuery() {
  return useQuery({
    queryKey: metaKeys.version(),
    queryFn: ({ signal }) => getVersion({ signal }),
  })
}

// 轮询节奏（issue #3 收敛策略）：checking / importing 活跃态高频轮询；
// failed 低频轮询以展示重启容器后的续传恢复；completed / idle 是稳定态，
// 停止轮询（不发常驻请求）。首次加载与请求失败时尚无状态可判，按活跃
// 节奏重试，拿到首个响应后即按状态收敛。
export const DICT_IMPORT_ACTIVE_INTERVAL_MS = 3000
export const DICT_IMPORT_FAILED_INTERVAL_MS = 15000

export function dictImportRefetchIntervalMs(
  state: string | undefined,
): number | false {
  if (state === undefined || state === 'checking' || state === 'importing') {
    return DICT_IMPORT_ACTIVE_INTERVAL_MS
  }
  if (state === 'failed') return DICT_IMPORT_FAILED_INTERVAL_MS
  return false // completed / idle
}

export function useDictImportStatusQuery(
  options: { refetchInterval?: boolean } = {},
) {
  return useQuery({
    queryKey: metaKeys.dictImport(),
    queryFn: ({ signal }) => getDictionaryImport({ signal }),
    // 进度是瞬时状态：不消费过期缓存，挂载即取最新值。
    staleTime: 0,
    refetchInterval:
      options.refetchInterval === false
        ? false
        : (query) => dictImportRefetchIntervalMs(query.state.data?.state),
  })
}
