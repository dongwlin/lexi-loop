// 环境变量的唯一读取与校验入口（前端应用架构规范 §4.7 / §12.1）。
// 其他模块不得直接读取 import.meta.env。

function resolveApiBaseUrl(): string {
  const raw = import.meta.env.VITE_API_BASE_URL

  // 未配置时走同源相对路径：开发环境由 Vite proxy 转发 /api，生产同域部署。
  if (raw === undefined || raw === '') {
    return ''
  }

  let parsed: URL
  try {
    parsed = new URL(raw)
  } catch {
    throw new Error('环境变量 VITE_API_BASE_URL 不是合法 URL')
  }

  if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
    throw new Error('环境变量 VITE_API_BASE_URL 必须使用 http/https 协议')
  }
  if (parsed.search || parsed.hash) {
    throw new Error('环境变量 VITE_API_BASE_URL 不能包含 query 或 hash')
  }

  // 只保留 origin + 路径前缀，去掉末尾斜杠，供 client 拼接 /api/v1/... 路径。
  return `${parsed.origin}${parsed.pathname.replace(/\/+$/, '')}`
}

export const env = {
  apiBaseUrl: resolveApiBaseUrl(),
} as const
