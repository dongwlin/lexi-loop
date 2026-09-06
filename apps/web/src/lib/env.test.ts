import { afterEach, describe, expect, it, vi } from 'vitest'

import { env } from './env'

afterEach(() => {
  vi.unstubAllEnvs()
  vi.resetModules()
})

describe('apiBaseUrl 校验', () => {
  it('未设置 VITE_API_BASE_URL 时为同源空串', () => {
    expect(env.apiBaseUrl).toBe('')
  })

  it('合法 URL 保留 origin 与路径前缀并去掉末尾斜杠', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.net/base/')
    const { env: reloaded } = await import('./env')

    expect(reloaded.apiBaseUrl).toBe('https://api.example.net/base')
  })

  it('非法协议直接报错', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'ftp://api.example.net')

    await expect(import('./env')).rejects.toThrow('http/https')
  })

  it('携带 query 或 hash 直接报错', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.net?x=1')

    await expect(import('./env')).rejects.toThrow('query 或 hash')
  })
})
