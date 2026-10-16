import { describe, expect, it } from 'vitest'

import {
  isOptionalTextReplacementConfig,
  normalizeTextReplacementConfig,
} from '@/lib/text-replacements'

describe('text replacement configuration', () => {
  it('accepts a valid request and response rule set', () => {
    expect(
      isOptionalTextReplacementConfig(
        JSON.stringify({
          request: [{ from: 'source', to: 'upstream' }],
          response: [{ from: 'upstream', to: 'source' }],
        })
      )
    ).toBe(true)
  })

  it('rejects empty sources, unknown fields, and more than 100 rules', () => {
    expect(
      isOptionalTextReplacementConfig(
        JSON.stringify({ request: [{ from: '', to: 'upstream' }] })
      )
    ).toBe(false)
    expect(isOptionalTextReplacementConfig('{"request":[],"extra":true}')).toBe(
      false
    )
    expect(
      isOptionalTextReplacementConfig(
        JSON.stringify({
          request: Array.from({ length: 101 }, () => ({ from: 'a', to: 'b' })),
        })
      )
    ).toBe(false)
  })

  it('normalizes empty configurations to an omitted value', () => {
    expect(normalizeTextReplacementConfig('{"request":[],"response":[]}')).toBe(
      ''
    )
    expect(normalizeTextReplacementConfig('')).toBe('')
  })
})
