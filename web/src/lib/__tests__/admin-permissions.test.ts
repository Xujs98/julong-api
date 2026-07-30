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
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '../admin-permissions.ts'
import { ROLE } from '../roles.ts'

test('root can access ledger actions without an explicit permission matrix', () => {
  const root = { id: 1, username: 'root', role: ROLE.SUPER_ADMIN }

  assert.equal(
    hasPermission(
      root,
      ADMIN_PERMISSION_RESOURCES.LEDGER,
      ADMIN_PERMISSION_ACTIONS.DELETE
    ),
    true
  )
})

test('administrator ledger access follows the granted action matrix', () => {
  const admin = {
    id: 2,
    username: 'admin',
    role: ROLE.ADMIN,
    permissions: {
      admin_permissions: {
        ledger: { read: true, write: false, delete: false },
      },
    },
  }

  assert.equal(
    hasPermission(
      admin,
      ADMIN_PERMISSION_RESOURCES.LEDGER,
      ADMIN_PERMISSION_ACTIONS.READ
    ),
    true
  )
  assert.equal(
    hasPermission(
      admin,
      ADMIN_PERMISSION_RESOURCES.LEDGER,
      ADMIN_PERMISSION_ACTIONS.WRITE
    ),
    false
  )
  assert.equal(
    hasPermission(
      admin,
      ADMIN_PERMISSION_RESOURCES.LEDGER,
      ADMIN_PERMISSION_ACTIONS.DELETE
    ),
    false
  )
})
