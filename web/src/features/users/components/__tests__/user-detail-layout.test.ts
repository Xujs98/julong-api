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

import { userDetailDialogLayoutClasses } from '../user-detail-layout.ts'

describe('user detail dialog layout', () => {
  test('keeps long tab content inside a scrollable remaining-height region', () => {
    const contentClasses = userDetailDialogLayoutClasses.content.split(' ')
    const headerClasses = userDetailDialogLayoutClasses.header.split(' ')
    const tabsClasses = userDetailDialogLayoutClasses.tabs.split(' ')
    const tabsListClasses = userDetailDialogLayoutClasses.tabsList.split(' ')
    const scrollAreaClasses =
      userDetailDialogLayoutClasses.scrollArea.split(' ')

    assert.ok(contentClasses.includes('flex'))
    assert.ok(contentClasses.includes('flex-col'))
    assert.ok(contentClasses.includes('overflow-hidden'))
    assert.ok(headerClasses.includes('shrink-0'))
    assert.ok(tabsClasses.includes('min-h-0'))
    assert.ok(tabsClasses.includes('flex-1'))
    assert.ok(tabsClasses.includes('overflow-hidden'))
    assert.ok(tabsListClasses.includes('shrink-0'))
    assert.ok(scrollAreaClasses.includes('min-h-0'))
    assert.ok(scrollAreaClasses.includes('flex-1'))
  })
})
