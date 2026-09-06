import { ApiError } from '@lexi-loop/api-client'
import { describe, expect, it } from 'vitest'

import { describeReviewStartError } from './review-errors'

// 开始复习失败文案映射（review-flow §6 / §9 共用）：按 ApiError 分类给文案，
// 服务端英文 message 不直接展示。

describe('describeReviewStartError', () => {
  it('maps BASE.BIZ.USER_DISABLED to the no-reviewable-words copy', () => {
    const err = new ApiError('http', 'no reviewable words available', {
      httpStatus: 422,
      code: 'BASE.BIZ.USER_DISABLED',
    })
    expect(describeReviewStartError(err)).toBe(
      '当前没有可复习的生词，请先导入生词。',
    )
  })

  it('maps network and timeout failures to the connectivity copy', () => {
    expect(describeReviewStartError(new ApiError('network', 'offline'))).toBe(
      '网络异常，请检查连接后重试。',
    )
    expect(describeReviewStartError(new ApiError('timeout', 'slow'))).toBe(
      '网络异常，请检查连接后重试。',
    )
  })

  it('maps 5xx responses to the service-unavailable copy', () => {
    const err = new ApiError('http', 'boom', { httpStatus: 503 })
    expect(describeReviewStartError(err)).toBe('服务暂时不可用，请稍后重试。')
  })

  it('keeps 4xx errors other than USER_DISABLED on the generic copy', () => {
    const err = new ApiError('http', 'bad request', { httpStatus: 400 })
    expect(describeReviewStartError(err)).toBe('开始复习失败，请稍后重试')
  })

  it('falls back to the generic copy for unknown errors', () => {
    expect(describeReviewStartError(new Error('unexpected'))).toBe(
      '开始复习失败，请稍后重试',
    )
    expect(describeReviewStartError(null)).toBe('开始复习失败，请稍后重试')
  })
})
