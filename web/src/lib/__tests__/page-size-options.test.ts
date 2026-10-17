import { expect, it } from 'vitest'

import {
  DEFAULT_PAGE_SIZE_OPTIONS,
  isValidPageSizeOptionsString,
  parsePageSizeOptions,
  resolvePageSize,
} from '../page-size-options'

it('parses valid status payloads while preserving order', () => {
  expect(parsePageSizeOptions({ page_size_options: [25, 10, 50] })).toEqual([
    25, 10, 50,
  ])
  expect(
    parsePageSizeOptions({ data: { page_size_options: [25, 10] } })
  ).toEqual([25, 10])
})

it('falls back to defaults for invalid or missing values', () => {
  expect(parsePageSizeOptions('10,10')).toEqual(DEFAULT_PAGE_SIZE_OPTIONS)
  expect(parsePageSizeOptions([0, 20])).toEqual(DEFAULT_PAGE_SIZE_OPTIONS)
  expect(parsePageSizeOptions(undefined)).toEqual(DEFAULT_PAGE_SIZE_OPTIONS)
})

it('validates strings and resolves unavailable page sizes', () => {
  expect(isValidPageSizeOptionsString('25, 50,100')).toBe(true)
  expect(isValidPageSizeOptionsString('25,25')).toBe(false)
  expect(resolvePageSize(20, [25, 50])).toBe(25)
  expect(resolvePageSize(50, [25, 50])).toBe(50)
})
