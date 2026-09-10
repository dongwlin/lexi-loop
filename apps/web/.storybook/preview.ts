// Story 预览加载应用唯一样式入口，保证 Design Token / Tailwind 与应用渲染一致。
import type { Preview } from '@storybook/vue3-vite'
import { setup } from '@storybook/vue3-vite'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createMemoryHistory, createRouter } from 'vue-router'

import '../src/styles/main.css'

// 页面 Story（src/pages/）需要应用的全局装配，而 Story 自身无法 app.use 插件：
// Storybook 的 Vue3 renderer 只在 preview 暴露 setup。这里只装页面真正依赖的两项，
// 且都自建而不 import src 下的模块——本文件属于 tsconfig.node 项目（nodenext 解析、
// 无 DOM lib），引入应用源码会连带把 DOM 依赖拖进这个项目：
// - 路由：页面用 useRouter。内存历史，不写地址栏、不改 document.title，
//   也不注册应用的全局 afterEach，Story 之间不共享导航状态。
// - TanStack Query：页面用 useQueryClient 做命令式取数。客户端随 setup 新建；
//   Story 里的请求全部由替身应答，失败即失败，不重试（避免掩盖缺失的替身）。
setup((app) => {
  // 占位组件：页面 Story 直接渲染页面组件，RouterView 不参与；路由记录缺少 component(s)
  // 时 vue-router 会对每条记录打印警告，占位只是为了保持控制台干净。
  const RouteStub = { template: '<div />' }
  app.use(
    createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/review', name: 'review', component: RouteStub },
        {
          path: '/review/result/:session',
          name: 'review-result',
          component: RouteStub,
        },
        { path: '/:pathMatch(.*)*', name: 'not-found', component: RouteStub },
      ],
    }),
  )
  app.use(VueQueryPlugin, {
    queryClient: new QueryClient({
      defaultOptions: { queries: { retry: false } },
    }),
  })
})

const preview: Preview = {
  parameters: {
    // axe 检查默认作为测试门禁（前端测试规范 §10.5）：违规即失败。
    // 个别 Story 需要放宽时须显式标注 'todo'（已登记待修）或 'off'（反例展示）。
    a11y: {
      test: 'error',
    },
  },
}

export default preview
