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
  canManageUserTag,
  getAssignableUserTags,
  getUserTagFilterValue,
  USER_TAG_COLOR_PRESETS,
  USER_TAG_ROW_MARKER_CLASS_NAME,
} from '../user-tags.ts'

describe('user tag presentation', () => {
  test('keeps the requested color presets in display order', () => {
    assert.deepEqual(USER_TAG_COLOR_PRESETS, [
      '#EF4444',
      '#F97316',
      '#F59E0B',
      '#22C55E',
      '#3B82F6',
    ])
  })

  test('keeps the tag marker on the row edge before the checkbox', () => {
    const classes = USER_TAG_ROW_MARKER_CLASS_NAME.split(' ')

    assert.ok(classes.includes('absolute'))
    assert.ok(classes.includes('-left-2'))
    assert.ok(classes.includes('h-15'))
    assert.ok(classes.includes('w-1'))
    assert.ok(classes.includes('-translate-y-1/2'))
  })

  test('maps users without a tag to the no-tag filter value', () => {
    assert.equal(getUserTagFilterValue(), '0')
    assert.equal(getUserTagFilterValue(15), '15')
  })

  test('excludes built-in risk tags from manual tag management', () => {
    const tags = [
      {
        id: -1,
        name: 'Medium risk',
        color: '#C2410C',
        built_in: true,
        risk_level: 'medium' as const,
      },
      { id: 8, name: 'Follow up', color: '#3B82F6' },
    ]

    assert.equal(canManageUserTag(tags[0]), false)
    assert.equal(canManageUserTag(tags[1]), true)
    assert.deepEqual(getAssignableUserTags(tags), [tags[1]])
  })
})
