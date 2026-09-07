// meta Feature 的 Query（前端应用架构规范 §8.1）：DTO 直接使用 api-client 生成类型，
// queryFn 透传 signal；版本信息以后端为单一来源（docs/api/meta.md），前端不重复注入。
import { useQuery } from '@tanstack/vue-query'
import { getVersion } from '@lexi-loop/api-client'

import { metaKeys } from './keys'

export function useVersionQuery() {
  return useQuery({
    queryKey: metaKeys.version(),
    queryFn: ({ signal }) => getVersion({ signal }),
  })
}
