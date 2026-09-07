// 默认 Handler 工厂（前端测试规范 §8.1 / §10.3）：页面集成测试与 Story 共用。
// 保持为空数组 + `onUnhandledRequest: 'error'`：用例通过 `server.use()` 显式声明
// 当轮响应，命中未声明的请求一律视为测试缺陷（测试不得触碰真实网络）。
// 只从同构入口 'msw' 导入，浏览器端（Storybook MSW 集成）可安全复用。
import type {
  GetSessionResponse,
  ImportWordsResponse,
  VersionInfo,
  WordDetail,
  WordListItem,
} from '@lexi-loop/api-client'
import type { RequestHandler } from 'msw'

export const handlers: RequestHandler[] = []

// 响应数据工厂返回新对象，避免用例间共享可变状态（前端测试规范 §2.4）。
// 形状对齐 @lexi-loop/api-client 的 DTO 类型（缺省项以 ambiguous 为例），
// 契约变化时由类型检查暴露。

export function makeWordListItem(
  overrides: Partial<WordListItem> = {},
): WordListItem {
  return {
    id: '0f0f3d5a-0000-4000-8000-000000000001',
    word: 'ambiguous',
    phonetic: '/æmˈbɪɡjuəs/',
    effectiveReviewMeaning: [{ pos: 'adj.', translations: ['模棱两可的'] }],
    encounterCount: 3,
    reviewCount: 2,
    rememberCount: 1,
    forgetCount: 1,
    currentStreak: 1,
    lastReviewedAt: null,
    masteryScore: 33,
    reviewWeight: 4.62,
    ...overrides,
  }
}

export function makeWordDetail(
  overrides: Partial<WordDetail> = {},
): WordDetail {
  return {
    ...makeWordListItem(),
    definition: '',
    meaningSource: 'review',
    ...overrides,
  }
}

export function makeImportWordsResponse(
  overrides: Partial<ImportWordsResponse> = {},
): ImportWordsResponse {
  return {
    encounters: 4,
    created: 3,
    updated: 1,
    items: [
      { word: 'ambiguous', count: 1, result: 'created' },
      { word: 'constrain', count: 2, result: 'updated' },
      { word: 'derive', count: 1, result: 'created' },
    ],
    ...overrides,
  }
}

export function makeSessionResponse(
  overrides: Partial<GetSessionResponse> = {},
): GetSessionResponse {
  return {
    sessionId: '0f0f3d5a-0000-5000-8000-00000000c001',
    status: 'completed',
    total: 3,
    remembered: 2,
    forgotten: 1,
    completedAt: '2026-09-07T10:30:00Z',
    items: [
      { word: 'ambiguous', result: 'remembered' },
      { word: 'diligent', result: 'forgotten' },
      { word: 'constrain', result: 'remembered' },
    ],
    ...overrides,
  }
}

export function makeVersionInfo(
  overrides: Partial<VersionInfo> = {},
): VersionInfo {
  return {
    version: '1.2.3',
    buildTime: '2026-09-01T10:30:00Z',
    goVersion: 'go1.26.7',
    ...overrides,
  }
}
