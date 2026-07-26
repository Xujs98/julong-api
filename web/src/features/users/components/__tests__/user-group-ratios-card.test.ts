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
  formatUserGroupRatio,
  getSortedUserGroupRatios,
  userGroupRatiosCardClassName,
} from '../user-group-ratios-card-layout.ts'

describe('UserGroupRatiosCard', () => {
  test('keeps every actual group ratio and sorts groups consistently', () => {
    const ratios = getSortedUserGroupRatios({
      vip: 0.35,
      default: 1,
      svip: 0.25,
    })

    assert.deepEqual(ratios, [
      { group: 'default', ratio: 1 },
      { group: 'svip', ratio: 0.25 },
      { group: 'vip', ratio: 0.35 },
    ])
    assert.equal(`${formatUserGroupRatio(ratios[2].ratio)}x`, '0.35x')
  })

  test('uses a bordered card container below the user identity', () => {
    const classes = userGroupRatiosCardClassName.split(' ')

    assert.ok(classes.includes('rounded-md'))
    assert.ok(classes.includes('border'))
    assert.ok(classes.includes('bg-muted/30'))
  })
})
