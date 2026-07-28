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

test('usage log export forwards the active filters without page limits', () => {
  const params = buildUsageLogExportParams({
    p: 3,
    page_size: 100,
    type: 5,
    model_name: 'gpt-test',
    username: 'risk-user',
    request_id: 'req-1',
    start_timestamp: 1000,
    end_timestamp: 2000,
  })

  assert.deepEqual(params, {
    type: 5,
    model_name: 'gpt-test',
    username: 'risk-user',
    request_id: 'req-1',
    start_timestamp: 1000,
    end_timestamp: 2000,
  })
  assert.equal('p' in params, false)
  assert.equal('page_size' in params, false)
})
