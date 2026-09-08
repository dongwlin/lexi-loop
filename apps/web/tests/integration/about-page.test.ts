// 「关于」页集成测试（前端测试规范 §11）：版本 / 构建时间 / Go 版本三个字段
// 的渲染、开发构建占位（buildTime 空展示「—」）、GitHub 仓库链接与加载失败
// 分支。契约见 docs/api/meta.md。
import { HttpResponse, http } from 'msw'
import { screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import AboutPage from '@/pages/about/AboutPage.vue'

import { makeVersionInfo } from '../mocks/handlers'
import { server } from '../mocks/server'
import { renderAtRoute } from './helpers'

describe('关于页', () => {
  it('展示应用版本、构建时间与 Go 版本', async () => {
    server.use(
      http.get('*/api/v1/version', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeVersionInfo(),
        }),
      ),
    )

    await renderAtRoute(AboutPage, '/about')

    expect(await screen.findByText('1.2.3')).toBeVisible()
    // formatDateTime 统一按东八区渲染（10:30Z → 18:30）。
    expect(screen.getByText('2026-09-01 18:30')).toBeVisible()
    expect(screen.getByText('go1.26.7')).toBeVisible()
  })

  it('开发构建（版本 dev、无构建时间）以 — 占位', async () => {
    server.use(
      http.get('*/api/v1/version', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeVersionInfo({ version: 'dev', buildTime: '' }),
        }),
      ),
    )

    await renderAtRoute(AboutPage, '/about')

    expect(await screen.findByText('dev')).toBeVisible()
    expect(screen.getByText('—')).toBeVisible()
  })

  it('Source Code 行的 GitHub 链接在新标签页打开仓库地址', async () => {
    server.use(
      http.get('*/api/v1/version', () =>
        HttpResponse.json({
          code: 'OK',
          message: 'success',
          data: makeVersionInfo(),
        }),
      ),
    )

    await renderAtRoute(AboutPage, '/about')

    expect(await screen.findByText('Source Code')).toBeVisible()
    const link = screen.getByRole('link', { name: 'GitHub' })
    expect(link).toBeVisible()
    expect(link).toHaveAttribute(
      'href',
      'https://github.com/dongwlin/lexi-loop',
    )
    expect(link).toHaveAttribute('target', '_blank')
    expect(link.getAttribute('rel')).toContain('noopener')
    expect(link.getAttribute('rel')).toContain('noreferrer')
  })

  it('加载失败时展示错误提示与重试入口', async () => {
    server.use(
      http.get('*/api/v1/version', () =>
        HttpResponse.json(
          { code: 'ERROR', message: '服务不可用', data: {} },
          { status: 500 },
        ),
      ),
    )

    await renderAtRoute(AboutPage, '/about')

    expect(await screen.findByRole('alert')).toBeVisible()
    expect(screen.getByRole('button', { name: '重试' })).toBeVisible()
  })
})
