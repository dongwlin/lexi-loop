/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** API Base URL（仅 origin，可选）。未设置时走同源（开发环境经 Vite proxy 转发）。 */
  readonly VITE_API_BASE_URL?: string
}
