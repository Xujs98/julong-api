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

import {
  getActualGroupRatio,
  getDisplayedGroupRatio,
} from '../lib/group-ratio.ts'

describe('getActualGroupRatio', () => {
  test('uses the original user-specific ratio and preserves a real 1x ratio', () => {
    assert.equal(
      getActualGroupRatio({ group_ratio: 1, user_group_ratio: 0.095 }),
      0.095
    )
    assert.equal(getActualGroupRatio({ group_ratio: 1 }), 1)
  })
})

describe('getDisplayedGroupRatio', () => {
  test('uses the user-specific ratio when the system response includes one', () => {
    assert.equal(
      getDisplayedGroupRatio({
        group_ratio: 0.06,
        user_group_ratio: 0.095,
      }),
      0.095
    )
  })

  test('keeps the original system behavior of hiding a plain 1x ratio', () => {
    assert.equal(getDisplayedGroupRatio({ group_ratio: 1 }), null)
  })

  test('shows 1x when manual or pricing-group display explicitly requests it', () => {
    assert.equal(
      getDisplayedGroupRatio({
        group_ratio: 1,
        group_ratio_display_mode: 'manual',
      }),
      1
    )
  })

  test('shows no ratio after the backend strips ratio fields', () => {
    assert.equal(getDisplayedGroupRatio({ model_price: 0.004 }), null)
  })
})
