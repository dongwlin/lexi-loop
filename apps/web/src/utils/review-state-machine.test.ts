import { describe, expect, it } from 'vitest'

import {
  advanceAfterAnswer,
  enterReview,
  chooseReviewResult,
  correctReviewResult,
} from './review-state-machine'

describe('enterReview', () => {
  it('有卡时从第 0 张以 recalling 进入', () => {
    expect(enterReview(30)).toEqual({ mode: 'recalling', currentIndex: 0 })
  })

  it('恢复时从指定下标进入（review-flow §6 回到原进度）', () => {
    expect(enterReview(30, 11)).toEqual({ mode: 'recalling', currentIndex: 11 })
  })

  it('恢复下标越界时钳制到 [0, itemCount - 1]', () => {
    expect(enterReview(10, -3)).toEqual({ mode: 'recalling', currentIndex: 0 })
    expect(enterReview(10, 99)).toEqual({ mode: 'recalling', currentIndex: 9 })
  })

  it('空轮保持 idle（防御分支，无当前卡）', () => {
    expect(enterReview(0)).toEqual({ mode: 'idle', currentIndex: -1 })
    expect(enterReview(0, 5)).toEqual({ mode: 'idle', currentIndex: -1 })
  })
})

describe('chooseReviewResult', () => {
  it.each(['remembered', 'forgotten'] as const)(
    '初选 %s 只进入揭示阶段',
    (result) => {
      expect(chooseReviewResult('recalling', true, result)).toEqual({
        mode: 'revealed',
        pendingResult: result,
      })
    },
  )
  it('拒绝重复选择、未开始与无卡状态', () => {
    expect(chooseReviewResult('revealed', true, 'remembered')).toBeNull()
    expect(chooseReviewResult('idle', true, 'forgotten')).toBeNull()
    expect(chooseReviewResult('recalling', false, 'forgotten')).toBeNull()
  })
})

describe('correctReviewResult', () => {
  it('仅在揭示阶段将认识改为不认识', () => {
    expect(correctReviewResult('revealed', 'remembered')).toBe('forgotten')
    expect(correctReviewResult('revealed', 'forgotten')).toBeNull()
    expect(correctReviewResult('revealed', null)).toBeNull()
    expect(correctReviewResult('recalling', 'remembered')).toBeNull()
    expect(correctReviewResult('idle', 'remembered')).toBeNull()
  })
})

describe('advanceAfterAnswer', () => {
  it('非最后一张答完进入下一张 recalling', () => {
    expect(advanceAfterAnswer(0, 3)).toEqual({
      outcome: 'next',
      nextIndex: 1,
      mode: 'recalling',
    })
    expect(advanceAfterAnswer(1, 3)).toEqual({
      outcome: 'next',
      nextIndex: 2,
      mode: 'recalling',
    })
  })

  it('最后一张答完完成本轮（页面跳转结果页）', () => {
    expect(advanceAfterAnswer(2, 3)).toEqual({ outcome: 'finished' })
  })

  it('单卡轮次第 0 张答完即完成', () => {
    expect(advanceAfterAnswer(0, 1)).toEqual({ outcome: 'finished' })
  })
})
