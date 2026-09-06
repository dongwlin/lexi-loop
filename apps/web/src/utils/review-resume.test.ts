import { describe, expect, it } from 'vitest'

import { computeResumeProgress } from './review-resume'

function serverItem(word: string, result: string) {
  return { word, result }
}

describe('computeResumeProgress', () => {
  it('returns first pending index when partially answered', () => {
    const progress = computeResumeProgress(
      ['ambiguous', 'constrain', 'derive'],
      [serverItem('ambiguous', 'remembered'), serverItem('constrain', 'pending'), serverItem('derive', 'pending')],
    )
    expect(progress).toEqual({ answered: 1, firstPendingIndex: 1 })
  })

  it('returns first pending index regardless of answer order', () => {
    const progress = computeResumeProgress(
      ['ambiguous', 'constrain', 'derive'],
      [serverItem('ambiguous', 'pending'), serverItem('constrain', 'forgotten'), serverItem('derive', 'pending')],
    )
    expect(progress).toEqual({ answered: 1, firstPendingIndex: 0 })
  })

  it('returns localWords.length when all answered', () => {
    const progress = computeResumeProgress(
      ['ambiguous', 'constrain'],
      [serverItem('ambiguous', 'remembered'), serverItem('constrain', 'forgotten')],
    )
    expect(progress).toEqual({ answered: 2, firstPendingIndex: 2 })
  })

  it('returns index 0 when nothing answered', () => {
    const progress = computeResumeProgress(
      ['ambiguous', 'constrain'],
      [serverItem('ambiguous', 'pending'), serverItem('constrain', 'pending')],
    )
    expect(progress).toEqual({ answered: 0, firstPendingIndex: 0 })
  })

  it('returns null when server items length mismatches local words', () => {
    const progress = computeResumeProgress(
      ['ambiguous', 'constrain'],
      [serverItem('ambiguous', 'pending')],
    )
    expect(progress).toBeNull()
  })

  it('returns null when server item word is missing locally', () => {
    const progress = computeResumeProgress(
      ['ambiguous', 'constrain'],
      [serverItem('ambiguous', 'remembered'), serverItem('unknown', 'pending')],
    )
    expect(progress).toBeNull()
  })

  it('returns null when server items contain duplicate words', () => {
    const progress = computeResumeProgress(
      ['ambiguous', 'constrain'],
      [serverItem('ambiguous', 'remembered'), serverItem('ambiguous', 'pending')],
    )
    expect(progress).toBeNull()
  })

  it('handles both sides empty', () => {
    const progress = computeResumeProgress([], [])
    expect(progress).toEqual({ answered: 0, firstPendingIndex: 0 })
  })

  it('returns null for empty server items with non-empty local words', () => {
    const progress = computeResumeProgress(['ambiguous'], [])
    expect(progress).toBeNull()
  })
})
