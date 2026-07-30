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

import {
  formatLedgerDateTimeInput,
  parseLedgerDateTimeInput,
} from '../lib/date-time.ts'

test('ledger date-time input preserves local hours, minutes, and seconds', () => {
  const timestamp = Math.floor(
    new Date(2026, 6, 30, 14, 35, 42).getTime() / 1000
  )
  const input = formatLedgerDateTimeInput(timestamp)

  assert.equal(input, '2026-07-30T14:35:42')
  assert.equal(parseLedgerDateTimeInput(input), timestamp)
})

test('ledger date-time parser rejects invalid input', () => {
  assert.equal(parseLedgerDateTimeInput(''), null)
  assert.equal(parseLedgerDateTimeInput('not-a-date'), null)
})
