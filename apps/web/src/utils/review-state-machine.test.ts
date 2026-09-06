import { describe, expect, it } from 'vitest'

import {
  advanceAfterAnswer,
  enterReview,
  revealCard,
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

describe('revealCard', () => {
  it('recalling 且有当前卡时迁入 revealed', () => {
    expect(revealCard('recalling', true)).toBe('revealed')
  })

  it('已 revealed 不迁移（重复揭示被吸收）', () => {
    expect(revealCard('revealed', true)).toBeNull()
  })

  it('idle 不迁移（未开始）', () => {
    expect(revealCard('idle', true)).toBeNull()
  })

  it('无当前卡不迁移', () => {
    expect(revealCard('recalling', false)).toBeNull()
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
