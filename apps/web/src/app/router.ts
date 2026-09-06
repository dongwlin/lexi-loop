import { createRouter, createWebHistory } from 'vue-router'

// 路由表显式维护（前端应用架构规范 §6.1），页面结构对应 docs/frontend/review-flow.md §1 的四个核心页面。
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    // review-flow.md §1 未定义 `/` 首页内容，先重定向到核心复习流程。
    { path: '/', redirect: { name: 'review' } },
    {
      path: '/words',
      name: 'words',
      component: () => import('@/pages/words/WordsPage.vue'),
    },
    {
      path: '/words/:id',
      name: 'word-detail',
      component: () => import('@/pages/words/WordDetailPage.vue'),
    },
    {
      path: '/import',
      name: 'import',
      component: () => import('@/pages/import/ImportPage.vue'),
    },
    {
      path: '/review',
      name: 'review',
      component: () => import('@/pages/review/ReviewPage.vue'),
    },
    {
      path: '/review/result/:session',
      name: 'review-result',
      component: () => import('@/pages/review/ReviewResultPage.vue'),
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/pages/error/NotFoundPage.vue'),
    },
  ],
})
