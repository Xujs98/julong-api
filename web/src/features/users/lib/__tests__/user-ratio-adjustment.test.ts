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
  buildUserGroupRatioPreviews,
  getSignedUserGroupRatioAdjustment,
} from '../user-ratio-adjustment.ts'

describe('user ratio adjustment', () => {
  test('stores decrease mode as a negative adjustment', () => {
    assert.equal(getSignedUserGroupRatioAdjustment(true, 'decrease', 0.1), -0.1)
  })

  test('previews only selected groups and applies special base ratios', () => {
    const previews = buildUserGroupRatioPreviews(
      { 'codex-v1': 0.08, SVIP: 0.4 },
      true,
      'increase',
      0.1
    )

    assert.deepEqual(previews, [
      {
        group: 'codex-v1',
        baseRatio: 0.08,
        adjustedRatio: 0.18,
        invalid: false,
      },
      {
        group: 'SVIP',
        baseRatio: 0.4,
        adjustedRatio: 0.5,
        invalid: false,
      },
    ])
  })

  test('marks a preview invalid when decrease would make it negative', () => {
    const [preview] = buildUserGroupRatioPreviews(
      { 'codex-v1': 0.08 },
      true,
      'decrease',
      0.1
    )

    assert.equal(preview?.invalid, true)
  })
})
