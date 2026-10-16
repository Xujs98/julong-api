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

import { describe, test } from 'vitest'

import {
  canViewOtherTaskLogs,
  canOpenUserDetails,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'

import { resolveLogsViewAccess } from '../components/usage-logs-provider'

describe('usage log access tier', () => {
  test('keeps users and elevated self views on the self tier', () => {
    assert.equal(resolveLogsViewAccess(ROLE.USER, 'all'), 'self')
    assert.equal(resolveLogsViewAccess(ROLE.ADMIN, 'self'), 'self')
    assert.equal(resolveLogsViewAccess(ROLE.SUPER_ADMIN, 'self'), 'self')
  })

  test('distinguishes admin and root while viewing all logs', () => {
    assert.equal(resolveLogsViewAccess(ROLE.ADMIN, 'all'), 'admin')
    assert.equal(resolveLogsViewAccess(ROLE.SUPER_ADMIN, 'all'), 'root')
  })
})

test('task categories grant cross-account views independently and Root remains unrestricted', () => {
  const viewer = {
    id: 2,
    username: 'admin',
    role: ROLE.ADMIN,
    permissions: {
      admin_permissions: {
        task_logs: {
          drawing_read: true,
          image_read: false,
          task_read: true,
          assets_read: false,
        },
        privacy: { hide_root: true },
      },
    },
  }
  assert.equal(canViewOtherTaskLogs(viewer, 'drawing'), true)
  assert.equal(canViewOtherTaskLogs(viewer, 'image'), false)
  assert.equal(canViewOtherTaskLogs(viewer, 'task'), true)
  assert.equal(canViewOtherTaskLogs(viewer, 'assets'), false)
  assert.equal(canOpenUserDetails(viewer, ROLE.SUPER_ADMIN), false)
  assert.equal(canOpenUserDetails(viewer, ROLE.USER), true)
  const root = { ...viewer, role: ROLE.SUPER_ADMIN }
  assert.equal(canViewOtherTaskLogs(root, 'assets'), true)
  assert.equal(canOpenUserDetails(root, ROLE.SUPER_ADMIN), true)
  assert.equal(
    canViewOtherTaskLogs({ ...viewer, role: ROLE.USER }, 'drawing'),
    false
  )
})
