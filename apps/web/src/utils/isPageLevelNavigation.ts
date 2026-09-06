/** vue-router 路由位置的最小结构（RouteLocationNormalized 可直接传入，测试可构造最小对象）。 */
interface RouteLocationLike {
  path: string
  matched: readonly unknown[]
}

/**
 * 判断一次路由导航是否为页面级导航（《前端交互与可访问性规范》§5.3）：
 * 初次加载（from 为 START_LOCATION，matched 为空）不属于 SPA 导航，焦点交给浏览器管理；
 * 仅 query / hash 变化（搜索、分页、筛选等局部 URL 更新）不移动焦点；
 * 路径（含路由参数）变化即页面主体变化，导航完成后应把焦点移到新页面主标题或主内容入口。
 */
export function isPageLevelNavigation(
  to: RouteLocationLike,
  from: RouteLocationLike,
): boolean {
  if (from.matched.length === 0) return false
  return to.path !== from.path
}
