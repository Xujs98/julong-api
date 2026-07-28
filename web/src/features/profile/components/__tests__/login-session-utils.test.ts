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

import { sessionDevice } from '../login-session-utils.ts'

describe('sessionDevice', () => {
  test('recognizes browser and operating system from the login user agent', () => {
    assert.equal(
      sessionDevice(
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/126.0 Safari/537.36',
        'Unknown device',
        'Browser'
      ),
      'Chrome · macOS'
    )
    assert.equal(
      sessionDevice(
        'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 Version/17.5 Mobile/15E148 Safari/604.1',
        'Unknown device',
        'Browser'
      ),
      'Safari · iOS'
    )
  })

  test('uses the fallback label when no user agent was recorded', () => {
    assert.equal(
      sessionDevice('', 'Unknown device', 'Browser'),
      'Unknown device'
    )
  })
})
