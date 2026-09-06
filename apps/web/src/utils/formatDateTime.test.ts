import { describe, expect, it } from 'vitest'
import { formatDateTime } from './formatDateTime'

describe('formatDateTime', () => {
  it('空值返回 null', () => {
    expect(formatDateTime(null)).toBeNull()
    expect(formatDateTime(undefined)).toBeNull()
    expect(formatDateTime('')).toBeNull()
  })

  it('非法时间串返回 null', () => {
    expect(formatDateTime('not-a-date')).toBeNull()
  })

  it('UTC 时间按东八区渲染为「YYYY-MM-DD HH:mm」', () => {
    expect(formatDateTime('2026-09-01T10:30:00Z')).toBe('2026-09-01 18:30')
  })

  it('时区换算跨日时日期随之变化', () => {
    expect(formatDateTime('2026-09-01T16:30:00Z')).toBe('2026-09-02 00:30')
    expect(formatDateTime('2026-09-01T15:59:00Z')).toBe('2026-09-01 23:59')
  })

  it('秒不参与展示', () => {
    expect(formatDateTime('2026-09-03T18:20:59Z')).toBe('2026-09-04 02:20')
  })
})
