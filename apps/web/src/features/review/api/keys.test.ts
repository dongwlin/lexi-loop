import { describe, expect, it } from 'vitest'

import { reviewKeys } from './keys'

describe('reviewKeys 层级', () => {
  it('session 键挂在 sessions 之下，sessions 挂在 all 之下', () => {
    const session = reviewKeys.session('s-1')

    expect(session.slice(0, 2)).toEqual(reviewKeys.sessions())
    expect(reviewKeys.sessions().slice(0, 1)).toEqual(reviewKeys.all)
  })

  it('不同 session id 得到不同键', () => {
    expect(reviewKeys.session('s-1')).not.toEqual(reviewKeys.session('s-2'))
  })
})
