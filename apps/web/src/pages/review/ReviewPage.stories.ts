import type { Meta, StoryObj } from '@storybook/vue3-vite'
import type { SessionWordItem } from '@lexi-loop/api-client'
import { expect } from 'storybook/test'

import { saveReviewCountPreference } from '@/utils/review-count-preference'
import { saveReviewSnapshot } from '@/utils/review-snapshot'

import ReviewPage from './ReviewPage.vue'

// 复习页的 Storybook 覆盖：数量选择（预设行、「自定义」原地替换与 D009 截断提示）、
// 答题中、active session 恢复视图、可复习生词为 0 的空库分支。
// 页面挂载即发起请求（GET /words 取可复习量，存在本地快照时再 GET /reviews/:id 校验），
// Story 在真实 Chromium 中执行（前端测试规范 §12），因此只在 fetch 边界替身，
// 状态机、焦点、键盘与恢复判定全部走真实代码路径。

const SESSION_ID = '0f0f3d5a-0000-5000-8000-00000000c0de'

// 会话条目与线上同形（SessionWordItem）：释义是结构化义项，页面用 formatMeanings
// 按「；」合并同一义项、按换行分隔义项。
const items: SessionWordItem[] = [
  {
    itemId: '0f0f3d5a-0000-6000-8000-000000000001',
    word: 'ambiguous',
    phonetic: '/æmˈbɪɡjuəs/',
    effectiveReviewMeaning: [
      { pos: 'adj.', translations: ['模棱两可的', '含糊不清的'] },
    ],
  },
  {
    itemId: '0f0f3d5a-0000-6000-8000-000000000002',
    word: 'brisk',
    phonetic: '/brɪsk/',
    effectiveReviewMeaning: [
      { pos: 'adj.', translations: ['轻快的', '活跃的'] },
    ],
  },
  {
    itemId: '0f0f3d5a-0000-6000-8000-000000000003',
    word: 'cite',
    phonetic: '/saɪt/',
    effectiveReviewMeaning: [{ pos: 'verb', translations: ['引用', '举例'] }],
  },
]

interface ApiRoute {
  /** 命中判据：请求 URL 的子串。 */
  path: string
  data: unknown
}

/** GET /words 分页响应；页面只消费 pagination.total 作为可复习量（D009 提示）。 */
function availableWords(total: number): ApiRoute {
  return {
    path: '/api/v1/words',
    data: {
      list: [],
      pagination: {
        page: 1,
        pageSize: 1,
        total,
        totalPages: total,
        hasMore: false,
      },
    },
  }
}

/** GET /reviews/:id；恢复检查只消费 status 与逐词 result，与本地快照按 word 对齐。 */
function activeSession(results: string[]): ApiRoute {
  const countOf = (result: string) =>
    results.filter((item) => item === result).length
  return {
    path: `/api/v1/reviews/${SESSION_ID}`,
    data: {
      sessionId: SESSION_ID,
      status: 'active',
      total: items.length,
      remembered: countOf('remembered'),
      forgotten: countOf('forgotten'),
      completedAt: null,
      items: items.map((item, index) => ({
        word: item.word,
        result: results[index] ?? 'pending',
      })),
    },
  }
}

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

// 网络替身：命中已登记端点返回 { code, message, data } 信封（api-client mutator 的解包
// 契约），其余一律 404——Story 不能悄悄打到真实后端或 Storybook 自己的开发服务器。
// 以属性描述符替换 / 还原（与 AutoPlay Story 替换 speechSynthesis 同一模式），
// 不直接取用 window.fetch 方法引用。
function stubApi(routes: ApiRoute[]) {
  const descriptor = Object.getOwnPropertyDescriptor(window, 'fetch')
  Object.defineProperty(window, 'fetch', {
    configurable: true,
    writable: true,
    value: (input: RequestInfo | URL) => {
      const url =
        typeof input === 'string'
          ? input
          : input instanceof URL
            ? input.href
            : input.url
      const route = routes.find((candidate) => url.includes(candidate.path))
      return Promise.resolve(
        route
          ? jsonResponse(
              { code: 'OK', message: 'success', data: route.data },
              200,
            )
          : jsonResponse(
              { code: 'NOT_FOUND', message: 'Story 未登记该端点', data: null },
              404,
            ),
      )
    },
  })
  return () => {
    if (descriptor) Object.defineProperty(window, 'fetch', descriptor)
    else Reflect.deleteProperty(window, 'fetch')
  }
}

/** 每个 Story 从干净状态开始：清掉上一轮的 localStorage，再植入快照 / 偏好与端点替身。 */
function pageStory(options: {
  routes: ApiRoute[]
  seed?: () => void
  /** 额外的环境替身工厂：在 beforeEach 时调用，安装并返回还原函数。 */
  stubs?: Array<() => () => void>
}) {
  return () => {
    localStorage.clear()
    options.seed?.()
    const restores = [
      stubApi(options.routes),
      ...(options.stubs ?? []).map((install) => install()),
    ]
    return () => restores.forEach((restore) => restore())
  }
}

// 浏览器测试环境没有可用音源，真实 SpeechSynthesis 会走失败分支，在闪卡上留下一条与
// 所测状态无关的错误提示。这里替身成立即结束的语音：状态区保持为空，卡片只呈现本
// Story 要钉住的状态（与 Flashcard/AutoPlay Story 同一模式，额外让 onend 立即回调）。
function stubSpeech() {
  const descriptor = Object.getOwnPropertyDescriptor(window, 'speechSynthesis')
  Object.defineProperty(window, 'speechSynthesis', {
    configurable: true,
    value: {
      cancel: () => {},
      speak: (utterance: { onend?: (() => void) | null }) =>
        utterance.onend?.(),
    },
  })
  return () => {
    if (descriptor) Object.defineProperty(window, 'speechSynthesis', descriptor)
    else Reflect.deleteProperty(window, 'speechSynthesis')
  }
}

const meta = {
  title: 'Review/Page',
  component: ReviewPage,
  decorators: [
    () => ({
      // 还原 AppShell 的主内容列与页面画布（mx-auto max-w-2xl + bg-background），
      // 页面按真实版心评审，而不是贴在 Storybook 的白底上。
      template:
        '<div class="min-h-dvh bg-background px-4 py-6"><div class="mx-auto w-full max-w-2xl"><story /></div></div>',
    }),
  ],
} satisfies Meta<typeof ReviewPage>
export default meta
type Story = StoryObj<typeof meta>

// 数量选择视图：预设行（默认 30 选中）与开始按钮，可复习量已就绪。
export const CountSelection: Story = {
  beforeEach: pageStory({ routes: [availableWords(214)] }),
  play: async ({ canvas }) => {
    await expect(
      await canvas.findByText('今天想复习多少个单词？'),
    ).toBeVisible()
    await expect(canvas.getByRole('button', { name: '30' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await expect(canvas.getByRole('button', { name: '开始复习' })).toBeEnabled()
  },
}

// 「自定义」原地替换为数字输入框，且持久化的自定义值 50 超过可复习量 12，
// 因此同时钉住 D009 截断提示（review-flow §6）。
export const CustomCountTruncated: Story = {
  beforeEach: pageStory({
    seed: () => saveReviewCountPreference({ mode: 'custom', count: 50 }),
    routes: [availableWords(12)],
  }),
  play: async ({ canvas }) => {
    // 预设行里的「自定义」按钮被同槽位输入框原地替换，并带回上次输入的数量。
    await expect(
      canvas.queryByRole('button', { name: '自定义' }),
    ).not.toBeInTheDocument()
    await expect(canvas.getByLabelText('自定义数量')).toHaveValue(50)
    await expect(
      await canvas.findByText(
        '当前只有 12 个可复习生词，本轮将复习全部 12 个。',
      ),
    ).toBeVisible()
  },
}

// 答题中：本地快照带 autoResume（结果页「再来一轮」的进入路径，review-flow §9），
// 页面跳过恢复询问直接进入第一题的回忆阶段，无需在 Story 里模拟点击。
export const Answering: Story = {
  beforeEach: pageStory({
    seed: () =>
      saveReviewSnapshot({
        sessionId: SESSION_ID,
        totalCount: items.length,
        items,
        autoResume: true,
      }),
    routes: [
      availableWords(items.length),
      activeSession(['pending', 'pending', 'pending']),
    ],
    stubs: [stubSpeech],
  }),
  play: async ({ canvas }) => {
    await expect(await canvas.findByText('1 / 3')).toBeVisible()
    await expect(
      canvas.getByRole('heading', { name: 'ambiguous' }),
    ).toBeVisible()
    await expect(canvas.getByText('先回忆这个单词的意思')).toBeVisible()
    // 回忆阶段不得提前泄露释义。
    await expect(canvas.queryByText(/模棱两可的/)).not.toBeInTheDocument()
  },
}

// 恢复视图（D010）：服务端仍有 active 轮次，按逐词结果对齐到首个未答项，由用户选择继续或放弃。
export const Recovery: Story = {
  beforeEach: pageStory({
    seed: () =>
      saveReviewSnapshot({
        sessionId: SESSION_ID,
        totalCount: items.length,
        items,
      }),
    routes: [
      availableWords(items.length),
      activeSession(['remembered', 'pending', 'pending']),
    ],
  }),
  play: async ({ canvas }) => {
    await expect(
      await canvas.findByText('上次复习未完成（1 / 3）。'),
    ).toBeVisible()
    await expect(canvas.getByRole('button', { name: '继续复习' })).toBeEnabled()
    await expect(canvas.getByRole('button', { name: '放弃本轮' })).toBeEnabled()
  },
}

// 空库分支：availableCount === 0 时页面仍渲染数量选择行与可点的「开始复习」，
// 截断提示落到「本轮将复习全部 0 个」。这个 Story 钉住现状，供空库专属空状态
// 落地时改写（届时断言与 Story 名一起更新，而不是悄悄失去覆盖）。
export const NoAvailableWords: Story = {
  beforeEach: pageStory({ routes: [availableWords(0)] }),
  play: async ({ canvas }) => {
    // 截断提示要等 POST 前的 GET /words 落地，用 findBy 等待可复习量解析完成。
    await expect(
      await canvas.findByText('当前只有 0 个可复习生词，本轮将复习全部 0 个。'),
    ).toBeVisible()
    await expect(canvas.getByRole('button', { name: '开始复习' })).toBeEnabled()
  },
}
