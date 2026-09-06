// 页面集成测试装配（前端测试规范 §11）：真实 Router + 全新 QueryClient + MSW，
// 只在 HTTP 边界 Mock。每个用例获得独立的 QueryClient，缓存不跨用例泄漏；
// MSW 请求走 vitest 注入的测试 Base URL（helpers 引入 '@/lib/api' 完成装配）。
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { render } from '@testing-library/vue'
import type { Component } from 'vue'
import { defineComponent, h } from 'vue'
import { RouterView } from 'vue-router'

// 引入应用侧 API 装配：configureApiClient 注入 vitest.config.ts 的测试 Base URL。
import '@/lib/api'
import { router } from '@/app/router'

// 跨页面行为的挂载壳：以真实 RouterView 渲染当前路由页面，页面内 router.push 可见。
const RouterShell = defineComponent({
  name: 'RouterShell',
  setup: () => () => h(RouterView),
})

export async function renderAtRoute(component: Component, url: string) {
  await router.push(url)
  await router.isReady()

  return render(component, {
    global: {
      plugins: [
        [
          VueQueryPlugin,
          {
            queryClient: new QueryClient({
              defaultOptions: { queries: { retry: false } },
            }),
          },
        ],
        router,
      ],
    },
  })
}

export function renderAppAtRoute(url: string) {
  return renderAtRoute(RouterShell, url)
}
