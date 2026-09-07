import { describe, expect, it } from 'vitest'

import { metaKeys } from './keys'

describe('metaKeys 层级', () => {
  it('version 键挂在 all 之下', () => {
    expect(metaKeys.version().slice(0, 1)).toEqual(metaKeys.all)
  })

  it('version 键稳定可复现', () => {
    expect(metaKeys.version()).toEqual(metaKeys.version())
  })
})
