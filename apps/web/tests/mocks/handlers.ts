// 默认 Handler 工厂（前端测试规范 §8.1 / §10.3）：页面集成测试与 Story 共用。
// 保持为空数组 + `onUnhandledRequest: 'error'`：用例通过 `server.use()` 显式声明
// 当轮响应，命中未声明的请求一律视为测试缺陷（测试不得触碰真实网络）。
// 只从同构入口 'msw' 导入，浏览器端（Storybook MSW 集成）可安全复用。
import type { RequestHandler } from 'msw'

export const handlers: RequestHandler[] = []

// 响应数据工厂返回新对象，避免用例间共享可变状态（前端测试规范 §2.4）。
// 形状对齐 @lexi-loop/api-client 的 WordListItem（缺省项以 ambiguous 为例）。
export function makeWordListItem(
  overrides: Partial<{ word: string; phonetic: string }> = {},
) {
  return {
    id: '0f0f3d5a-0000-4000-8000-000000000001',
    word: 'ambiguous',
    phonetic: '/æmˈbɪɡjuəs/',
    effectiveReviewMeaning: [{ pos: 'adj', translation: '模棱两可的' }],
    encounterCount: 3,
    reviewCount: 2,
    rememberCount: 1,
    forgetCount: 1,
    currentStreak: 1,
    lastReviewedAt: null,
    ...overrides,
  }
}
