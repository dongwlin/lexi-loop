import type { App } from 'vue'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { pinia } from './pinia'
import { queryClient } from './query-client'
import { router } from './router'

// 一次性的全局 Provider 装配（前端应用架构规范 §4.1）。
export function installApp(app: App): void {
  app.use(pinia)
  app.use(router)
  app.use(VueQueryPlugin, { queryClient })
}
