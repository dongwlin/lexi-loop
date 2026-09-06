// 设施冒烟：生产构建入口可加载，SPA 重定向与外壳 / 主导航语义渲染正常。
// 不依赖后端——复习页的可用量 / 恢复检查失败时页面仍保持外壳与 h1（状态分支）。
import { expect, test } from '@playwright/test'

test.describe('应用外壳', () => {
  test('根路径重定向到 /review 且页面骨架可交互', async ({ page }) => {
    await page.goto('/')

    await expect(page).toHaveURL(/\/review$/)
    await expect(
      page.getByRole('heading', { level: 1, name: '开始复习' }),
    ).toBeVisible()
    await expect(page.getByRole('navigation', { name: '主导航' })).toBeVisible()
    await expect(
      page.getByRole('navigation', { name: '主导航' }).getByRole('link', {
        name: '生词库',
      }),
    ).toBeVisible()
  })

  test('主导航可跳转到生词库页', async ({ page }) => {
    await page.goto('/')

    await page
      .getByRole('navigation', { name: '主导航' })
      .getByRole('link', { name: '生词库' })
      .click()

    await expect(page).toHaveURL(/\/words$/)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  })
})
