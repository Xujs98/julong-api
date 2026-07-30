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
  canConfirmLedgerDelete,
  LEDGER_DELETE_COUNTDOWN_SECONDS,
  nextLedgerDeleteCountdown,
} from '../lib/delete-confirmation.ts'

test('ledger deletion remains blocked until all ten countdown ticks complete', () => {
  let secondsLeft = LEDGER_DELETE_COUNTDOWN_SECONDS
  assert.equal(canConfirmLedgerDelete(secondsLeft, 2), false)

  for (let tick = 0; tick < LEDGER_DELETE_COUNTDOWN_SECONDS; tick += 1) {
    secondsLeft = nextLedgerDeleteCountdown(secondsLeft)
  }

  assert.equal(secondsLeft, 0)
  assert.equal(canConfirmLedgerDelete(secondsLeft, 2), true)
})

test('ledger deletion remains blocked without selected rows and countdown cannot underflow', () => {
  assert.equal(nextLedgerDeleteCountdown(0), 0)
  assert.equal(canConfirmLedgerDelete(0, 0), false)
})
