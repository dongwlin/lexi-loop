import { nextTick } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'

import { isPageLevelNavigation } from '@/utils/isPageLevelNavigation'

const BRAND = 'LexiLoop'

// 路由表显式维护（前端应用架构规范 §6.1），页面结构对应 docs/frontend/review-flow.md §1 的四个核心页面。
// meta.title 用于全局 afterEach 统一设置 document.title（交互与可访问性规范 §6.1）。
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    // review-flow.md §1 未定义 `/` 首页内容，先重定向到核心复习流程。
    { path: '/', redirect: { name: 'review' } },
    {
      path: '/words',
      name: 'words',
      component: () => import('@/pages/words/WordsPage.vue'),
      meta: { title: `生词库 · ${BRAND}` },
    },
    {
      path: '/words/:id',
      name: 'word-detail',
      component: () => import('@/pages/words/WordDetailPage.vue'),
      meta: { title: `单词详情 · ${BRAND}` },
    },
    {
      path: '/import',
      name: 'import',
      component: () => import('@/pages/import/ImportPage.vue'),
      meta: { title: `导入生词 · ${BRAND}` },
    },
    {
      path: '/review',
      name: 'review',
      component: () => import('@/pages/review/ReviewPage.vue'),
      meta: { title: `复习 · ${BRAND}` },
    },
    {
      path: '/review/result/:session',
      name: 'review-result',
      component: () => import('@/pages/review/ReviewResultPage.vue'),
      meta: { title: `复习结果 · ${BRAND}` },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/pages/error/NotFoundPage.vue'),
      meta: { title: `页面不存在 · ${BRAND}` },
    },
  ],
})

// 《前端交互与可访问性规范》§6.1：每次路由导航后同步更新 document.title。
// §5.3：页面级导航完成后把焦点移到新页面主标题（各页 h1 带 tabindex="-1"），
// 兜底主内容入口；搜索 / 分页等局部 query 更新与初次加载不移动焦点。
router.afterEach((to, from) => {
  const title = to.meta.title
  document.title = typeof title === 'string' ? title : BRAND
  if (!isPageLevelNavigation(to, from)) return
  void nextTick(() => {
    const main = document.getElementById('main-content')
    const target = main?.querySelector<HTMLElement>('h1') ?? main
    target?.focus()
  })
})
