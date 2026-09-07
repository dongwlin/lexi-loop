// 词典导入全局指示的组件测试（前端测试规范 §4.1 四态 + §7.2 定时刷新
// Fake Timer）：只在 HTTP 边界 Mock（MSW），轮询与 toast 计时用 fake timers
// 推进（不依赖真实等待）；popover / toast 挂在 Portal，断言从 document.body
// 查询（§9.5）。fake timers 下 RTL 不会自动推进计时器，一律用
// advanceTimersByTimeAsync 冲刷请求与响应后做同步断言。

import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { HttpResponse, http } from 'msw'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { dictImportRefetchIntervalMs } from '../api/queries'
import DictImportIndicator from './DictImportIndicator.vue'

import { server } from '../../../../tests/mocks/server'
import { makeDictImportStatus } from '../../../../tests/mocks/handlers'
import type { DictImportStatus } from '@lexi-loop/api-client'

// 引入应用侧 API 装配（configureApiClient 注入测试 Base URL）。
import '@/lib/api'

function mountIndicator(fixture: DictImportStatus, onRequest?: () => void) {
  server.use(
    http.get('*/api/v1/dictionary-import', () => {
      onRequest?.()
      return HttpResponse.json({
        code: 'OK',
        message: 'success',
        data: fixture,
      })
    }),
  )
  return render(DictImportIndicator, {
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
      ],
    },
  })
}

// 冲刷挂载请求与响应：fake timers 下用微任务推进替代 RTL 自动等待。
async function settle(ms = 50): Promise<void> {
  await vi.advanceTimersByTimeAsync(ms)
}

describe('dictImportRefetchIntervalMs', () => {
  it('按状态机映射轮询节奏（issue #3 收敛策略）', () => {
    expect(dictImportRefetchIntervalMs(undefined)).toBe(3000)
    expect(dictImportRefetchIntervalMs('checking')).toBe(3000)
    expect(dictImportRefetchIntervalMs('importing')).toBe(3000)
    expect(dictImportRefetchIntervalMs('failed')).toBe(15000)
    expect(dictImportRefetchIntervalMs('completed')).toBe(false)
    expect(dictImportRefetchIntervalMs('idle')).toBe(false)
  })
})

describe('DictImportIndicator', () => {
  beforeEach(() => {
    vi.useFakeTimers({
      toFake: ['setTimeout', 'setInterval', 'clearTimeout', 'clearInterval'],
    })
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('idle 状态不渲染任何内容', async () => {
    mountIndicator(makeDictImportStatus({ state: 'idle' }))
    await settle()

    expect(screen.queryByRole('button', { name: /词典导入/ })).toBeNull()
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('importing 状态显示进度徽标，点击展开详情与 progressbar', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    mountIndicator(makeDictImportStatus())
    await settle()

    const badge = screen.getByRole('button', {
      name: '词典导入进度 42%，查看详情',
    })
    await user.click(badge)

    expect(screen.getByText('词典导入中')).toBeInTheDocument()
    expect(screen.getByText('已处理行数')).toBeInTheDocument()
    // fixture 的已处理行数与已写入词条相等，千分位文本出现两处。
    expect(screen.getAllByText('323,400')).toHaveLength(2)
    expect(screen.getByText('770,611')).toBeInTheDocument()
    expect(screen.getByText('已写入词条')).toBeInTheDocument()

    // progressbar 语义（可访问性规范 §7.3）：值与 aria-valuetext 同步。
    const progressbar = screen.getByRole('progressbar', {
      name: '词典导入进度',
    })
    expect(progressbar).toHaveAttribute('aria-valuemax', '770611')
    expect(progressbar).toHaveAttribute('aria-valuenow', '323400')
    expect(progressbar).toHaveAttribute('aria-valuetext', '42%')
  })

  it('failed 状态显示错误徽标、续传指引与错误信息', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    mountIndicator(
      makeDictImportStatus({
        state: 'failed',
        rowsProcessed: 12345,
        entriesWritten: 12300,
        error: 'ecdict: read csv row 42: boom',
      }),
    )
    await settle()

    await user.click(
      screen.getByRole('button', { name: '词典导入失败，查看详情' }),
    )

    expect(screen.getByText('词典导入失败')).toBeInTheDocument()
    expect(
      screen.getByText('导入失败不影响生词流程；重启容器将自动续传完成导入。'),
    ).toBeInTheDocument()
    expect(
      screen.getByText('ecdict: read csv row 42: boom'),
    ).toBeInTheDocument()
    expect(screen.getByText('12,300')).toBeInTheDocument()
  })

  it.each(['checking', 'importing'])(
    '%s → completed 时弹一次性「词典已就绪」toast，徽标消失',
    async (state) => {
      mountIndicator(makeDictImportStatus({ state }))
      await settle()
      expect(
        screen.getByRole('button', { name: /查看详情/ }),
      ).toBeInTheDocument()

      // 模拟下一次轮询拿到 completed：覆盖 MSW Handler 并推进一个轮询周期。
      server.use(
        http.get('*/api/v1/dictionary-import', () =>
          HttpResponse.json({
            code: 'OK',
            message: 'success',
            data: makeDictImportStatus({
              state: 'completed',
              rowsProcessed: 770611,
              entriesWritten: 770611,
              updatedAt: '2026-09-07T12:00:35Z',
            }),
          }),
        ),
      )
      await settle(3200)

      expect(screen.queryByRole('button', { name: /词典导入/ })).toBeNull()
      expect(screen.getByRole('status')).toBeInTheDocument()
      expect(screen.getByText('词典已就绪')).toBeInTheDocument()
      expect(
        screen.getByText('词典导入完成，现在可以正常添加生词了。'),
      ).toBeInTheDocument()
    },
  )

  it('toast 在约 4.5s 后自动消失', async () => {
    mountIndicator(makeDictImportStatus())
    await settle()

    server.use(
      http.get('*/api/v1/dictionary-import', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeDictImportStatus({ state: 'completed' }),
        }),
      ),
    )
    await settle(3100)
    expect(screen.getByRole('status')).toBeInTheDocument()

    // 自动消失：toast 计时走 fake timers（挂载约 3.1s 处 toast 出现，
    // 再推进 4.6s 越过 4.5s 定时器后应消失）。
    await settle(4600)
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('completed 为稳定态：首次读取后停止轮询，不发常驻请求', async () => {
    let requestCount = 0
    mountIndicator(makeDictImportStatus({ state: 'completed' }), () => {
      requestCount += 1
    })
    await settle()
    expect(requestCount).toBe(1)

    // 首次响应后立即检查，不能等到 toast 的自动关闭时间之后才断言。
    expect(screen.queryByRole('status')).toBeNull()

    await settle(20000)
    expect(requestCount).toBe(1)
    expect(screen.queryByRole('button', { name: /词典导入/ })).toBeNull()
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('importing 期间按 3s 节奏轮询', async () => {
    let requestCount = 0
    mountIndicator(makeDictImportStatus(), () => {
      requestCount += 1
    })
    await settle()
    expect(requestCount).toBe(1)

    await settle(3100)
    expect(requestCount).toBe(2)
    await settle(6000)
    expect(requestCount).toBe(4)
  })
})
