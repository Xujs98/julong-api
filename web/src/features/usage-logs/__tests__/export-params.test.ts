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

import { buildUsageLogExportParams } from '../lib/export.ts'

test('current page export preserves active filters and pagination', () => {
  const params = buildUsageLogExportParams(
    {
      p: 3,
      page_size: 100,
      type: 5,
      model_name: 'gpt-test',
      username: 'risk-user',
      request_id: 'req-1',
      start_timestamp: 1000,
      end_timestamp: 2000,
    },
    { scope: 'page' }
  )

  assert.deepEqual(params, {
    p: 3,
    page_size: 100,
    type: 5,
    model_name: 'gpt-test',
    username: 'risk-user',
    request_id: 'req-1',
    start_timestamp: 1000,
    end_timestamp: 2000,
  })
})

test('today export replaces pagination and time filters with the local day', () => {
  const now = new Date(2026, 6, 28, 12, 30, 45)
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  const end = new Date(now)
  end.setHours(23, 59, 59, 999)

  const params = buildUsageLogExportParams(
    {
      p: 2,
      page_size: 100,
      model_name: 'gpt-test',
      start_timestamp: 1000,
      end_timestamp: 2000,
    },
    { scope: 'today', now }
  )

  assert.deepEqual(params, {
    model_name: 'gpt-test',
    start_timestamp: Math.floor(start.getTime() / 1000),
    end_timestamp: Math.floor(end.getTime() / 1000),
  })
})

test('custom export uses the selected time range without pagination', () => {
  const start = new Date('2026-07-01T01:02:03Z')
  const end = new Date('2026-07-02T04:05:06Z')

  const params = buildUsageLogExportParams(
    { p: 4, page_size: 20, group: 'default' },
    { scope: 'custom', start, end }
  )

  assert.deepEqual(params, {
    group: 'default',
    start_timestamp: Math.floor(start.getTime() / 1000),
    end_timestamp: Math.floor(end.getTime() / 1000),
  })
})

test('all records export removes pagination and time filters', () => {
  const params = buildUsageLogExportParams(
    {
      p: 5,
      page_size: 100,
      username: 'risk-user',
      start_timestamp: 1000,
      end_timestamp: 2000,
    },
    { scope: 'all' }
  )

  assert.deepEqual(params, { username: 'risk-user' })
})

test('custom export rejects an inverted time range', () => {
  assert.throws(
    () =>
      buildUsageLogExportParams(
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
