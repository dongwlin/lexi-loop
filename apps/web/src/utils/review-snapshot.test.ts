import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  clearReviewSnapshot,
  loadReviewSnapshot,
  saveReviewSnapshot,
  type ReviewSnapshot,
} from './review-snapshot'

function memoryStorage(): Storage {
  const map = new Map<string, string>()
  return {
    get length() {
      return map.size
    },
    clear: () => map.clear(),
    getItem: (key) => map.get(key) ?? null,
    key: (index) => [...map.keys()][index] ?? null,
    removeItem: (key) => {
      map.delete(key)
    },
    setItem: (key, value) => {
      map.set(key, value)
    },
  }
}

const STORAGE_KEY = 'lexi-loop.review-snapshot'

const sampleSnapshot: ReviewSnapshot = {
  sessionId: '01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001',
  totalCount: 30,
  items: [
    {
      itemId: '01991f3e-7b4c-7a21-8e3f-2c5d7a9b2002',
      word: 'ambiguous',
      phonetic: '/æmˈbɪɡjuəs/',
      effectiveReviewMeaning: [
        { pos: 'adjective', translations: ['模棱两可的', '含糊不清的'] },
      ],
    },
  ],
}

beforeEach(() => {
  vi.stubGlobal('localStorage', memoryStorage())
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('saveReviewSnapshot', () => {
  it('stores snapshot as JSON with version field', () => {
    saveReviewSnapshot(sampleSnapshot)
    const raw = localStorage.getItem(STORAGE_KEY)
    expect(raw).not.toBeNull()

    const parsed = JSON.parse(raw!)
    expect(parsed.version).toBe(1)
    expect(parsed.sessionId).toBe(sampleSnapshot.sessionId)
    expect(parsed.totalCount).toBe(sampleSnapshot.totalCount)
    expect(parsed.items).toEqual(sampleSnapshot.items)
  })
})

describe('loadReviewSnapshot', () => {
  it('returns null when no snapshot exists', () => {
    expect(loadReviewSnapshot()).toBeNull()
  })

  it('returns snapshot when valid', () => {
    saveReviewSnapshot(sampleSnapshot)
    const loaded = loadReviewSnapshot()
    expect(loaded).toEqual(sampleSnapshot)
  })

  it('returns null for corrupted JSON', () => {
    localStorage.setItem(STORAGE_KEY, 'not-json')
    expect(loadReviewSnapshot()).toBeNull()
  })

  it('returns null for missing version', () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ sessionId: 'x', totalCount: 1, items: [] }),
    )
    expect(loadReviewSnapshot()).toBeNull()
  })

  it('returns null for wrong version', () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ version: 999, sessionId: 'x', totalCount: 1, items: [] }),
    )
    expect(loadReviewSnapshot()).toBeNull()
  })

  it('returns null for missing sessionId', () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ version: 1, totalCount: 1, items: [] }),
    )
    expect(loadReviewSnapshot()).toBeNull()
  })

  it('returns null for non-array items', () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ version: 1, sessionId: 'x', totalCount: 1, items: 'not-array' }),
    )
    expect(loadReviewSnapshot()).toBeNull()
  })

  it('returns null for non-number totalCount', () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ version: 1, sessionId: 'x', totalCount: 'not-number', items: [] }),
    )
    expect(loadReviewSnapshot()).toBeNull()
  })
})

describe('clearReviewSnapshot', () => {
  it('removes the stored snapshot', () => {
    saveReviewSnapshot(sampleSnapshot)
    expect(loadReviewSnapshot()).not.toBeNull()

    clearReviewSnapshot()
    expect(loadReviewSnapshot()).toBeNull()
  })

  it('is safe to call when nothing is stored', () => {
    expect(() => clearReviewSnapshot()).not.toThrow()
  })
})
