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
import { describe, test } from 'node:test'

import type { UsageLog } from '../data/schema.ts'
import { getTokenUsageDisplay } from '../lib/model-token-adjustment.ts'
import type { LogOtherData } from '../types.ts'

const log: UsageLog = {
  id: 1,
  user_id: 1,
  created_at: 1,
  type: 2,
  content: '',
  username: 'user',
  token_name: 'token',
  model_name: 'gpt-test',
  quota: 0,
  prompt_tokens: 100,
  completion_tokens: 50,
  use_time: 1,
  is_stream: false,
  channel: 1,
  channel_name: '',
  token_id: 1,
  group: 'default',
  user_display_group_ratio: null,
  ip: '',
  other: '',
  request_id: '',
  upstream_request_id: '',
}

const other: LogOtherData = {
  cache_tokens: 20,
  cache_creation_tokens: 10,
  admin_info: {
    model_token_adjustment: {
      adjustments: { input: 0.1 },
      actual: {
        input: 100,
        output: 50,
        cache_read: 20,
        cache_creation: 10,
      },
      billed: {
        input: 127,
        output: 60,
        cache_read: 30,
        cache_creation: 20,
      },
    },
  },
}

describe('model token adjustment log display', () => {
  test('shows billed tokens to administrators and preserves actual tooltip values', () => {
    const display = getTokenUsageDisplay(log, other, true)

    assert.equal(display.hasAdjustment, true)
    assert.deepEqual(display.displayed, {
      input: 127,
      output: 60,
      cache_read: 30,
      cache_creation: 20,
      cache_creation_5m: 0,
      cache_creation_1h: 0,
    })
    assert.equal(display.actual.input, 100)
  })

  test('keeps real log tokens for non-admin viewers', () => {
    const display = getTokenUsageDisplay(log, other, false)

    assert.equal(display.hasAdjustment, false)
    assert.equal(display.displayed.input, 100)
    assert.equal(display.displayed.output, 50)
    assert.equal(display.displayed.cache_read, 20)
  })
})
