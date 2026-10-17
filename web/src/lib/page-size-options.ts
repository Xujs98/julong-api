/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const

const MIN_PAGE_SIZE = 1
const MAX_PAGE_SIZE = 100

function readRawPageSizeOptions(value: unknown): unknown {
  if (typeof value === 'string' || Array.isArray(value)) return value
  if (!value || typeof value !== 'object') return undefined

  const record = value as Record<string, unknown>
  if ('page_size_options' in record) return record.page_size_options
  if (record.data && typeof record.data === 'object') {
    return readRawPageSizeOptions(record.data)
  }
  return undefined
}

function parseValues(value: unknown): number[] | undefined {
  let values: unknown[] | undefined
  if (typeof value === 'string') {
    values = value.split(',').map((part) => part.trim())
  } else if (Array.isArray(value)) {
    values = value
  }
  if (!values || values.length === 0) return undefined

  const parsed: number[] = []
  const seen = new Set<number>()
  for (const item of values) {
    let pageSize = Number.NaN
    if (typeof item === 'number') {
      pageSize = item
    } else if (typeof item === 'string' && /^\d+$/.test(item)) {
      pageSize = Number(item)
    }
    if (
      !Number.isInteger(pageSize) ||
      pageSize < MIN_PAGE_SIZE ||
      pageSize > MAX_PAGE_SIZE ||
      seen.has(pageSize)
    ) {
      return undefined
    }
    seen.add(pageSize)
    parsed.push(pageSize)
  }
  return parsed
}

export function parsePageSizeOptions(value: unknown): number[] {
  const parsed = parseValues(readRawPageSizeOptions(value))
  return parsed ?? [...DEFAULT_PAGE_SIZE_OPTIONS]
}

export function isValidPageSizeOptionsString(value: string): boolean {
  return parseValues(value) !== undefined
}

export function resolvePageSize(
  pageSize: number | undefined,
  options: readonly number[] = DEFAULT_PAGE_SIZE_OPTIONS
): number {
  const fallback = options[0] ?? DEFAULT_PAGE_SIZE_OPTIONS[0]
  return pageSize !== undefined && options.includes(pageSize)
    ? pageSize
    : fallback
}
