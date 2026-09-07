import { ApiError } from '@lexi-loop/api-client'
import { describe, expect, it } from 'vitest'

import {
  describeReviewAbandonError,
  describeReviewStartError,
} from './review-errors'

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

describe('describeReviewAbandonError', () => {
  it('maps network and timeout failures to the connectivity copy', () => {
    expect(describeReviewAbandonError(new ApiError('network', 'offline'))).toBe(
      '网络异常，请检查连接后重试。',
    )
    expect(describeReviewAbandonError(new ApiError('timeout', 'slow'))).toBe(
      '网络异常，请检查连接后重试。',
    )
  })

  it('maps 5xx responses to the service-unavailable copy', () => {
    const err = new ApiError('http', 'boom', { httpStatus: 502 })
    expect(describeReviewAbandonError(err)).toBe('服务暂时不可用，请稍后重试。')
  })

  it('keeps contract-status errors (404 / 422) on the generic copy', () => {
    // 404 / 422 由页面按契约裁决（清快照回配置页），不走文案映射的分支。
    const notFound = new ApiError('http', 'not found', {
      httpStatus: 404,
      code: 'BASE.NOT_FOUND.USER',
    })
    const notActive = new ApiError('http', 'not active', {
      httpStatus: 422,
      code: 'BASE.BIZ.USER_DISABLED',
    })
    expect(describeReviewAbandonError(notFound)).toBe('放弃本轮失败，请重试')
    expect(describeReviewAbandonError(notActive)).toBe('放弃本轮失败，请重试')
  })

  it('falls back to the generic copy for unknown errors', () => {
    expect(describeReviewAbandonError(new Error('unexpected'))).toBe(
      '放弃本轮失败，请重试',
    )
  })
})
