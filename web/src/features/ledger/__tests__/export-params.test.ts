/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { test } from 'node:test'

import { buildLedgerExportParams } from '../lib/export.ts'

test('current page ledger export preserves pagination and date filters', () => {
  assert.deepEqual(
    buildLedgerExportParams(
      { p: 3, page_size: 20, start_timestamp: 1000, end_timestamp: 2000 },
      { scope: 'page' }
    ),
    { p: 3, page_size: 20, start_timestamp: 1000, end_timestamp: 2000 }
  )
})

test('today ledger export replaces pagination and active dates with the local day', () => {
  const now = new Date(2026, 6, 30, 12, 30, 45)
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  const end = new Date(now)
  end.setHours(23, 59, 59, 999)

  assert.deepEqual(
    buildLedgerExportParams(
      { p: 2, page_size: 20, start_timestamp: 1000, end_timestamp: 2000 },
      { scope: 'today', now }
    ),
    {
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    }
  )
})

test('custom ledger export uses the selected range without pagination', () => {
  const start = new Date('2026-07-01T01:02:03Z')
  const end = new Date('2026-07-02T04:05:06Z')

  assert.deepEqual(
    buildLedgerExportParams(
      { p: 4, page_size: 20 },
      { scope: 'custom', start, end }
    ),
    {
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    }
  )
})

test('all-record ledger export removes pagination and date filters', () => {
  assert.deepEqual(
    buildLedgerExportParams(
      { p: 5, page_size: 20, start_timestamp: 1000, end_timestamp: 2000 },
      { scope: 'all' }
    ),
    {}
  )
})

test('custom ledger export rejects an inverted time range', () => {
  assert.throws(
    () =>
      buildLedgerExportParams(
        {},
        {
          scope: 'custom',
          start: new Date('2026-07-02T00:00:00Z'),
          end: new Date('2026-07-01T00:00:00Z'),
        }
      ),
    RangeError
  )
})
