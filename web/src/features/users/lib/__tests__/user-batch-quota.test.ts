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

import { validateBatchQuotaInput } from '../user-batch-quota.ts'

describe('validateBatchQuotaInput', () => {
  test('requires selected users when all-users mode is disabled', () => {
    assert.equal(
      validateBatchQuotaInput({
        mode: 'add',
        amount: '0.0002',
        quotaValue: 100,
        allUsers: false,
        selectedUserIds: [],
        sendEmail: false,
        emailSubject: '',
        emailContent: '',
      }),
      'Select at least one user'
    )
  })

  test('accepts all-users mode without explicit user IDs', () => {
    assert.equal(
      validateBatchQuotaInput({
        mode: 'subtract',
        amount: '0.0002',
        quotaValue: 100,
        allUsers: true,
        selectedUserIds: [],
        sendEmail: false,
        emailSubject: '',
        emailContent: '',
      }),
      null
    )
  })

  test('requires editable email fields only when notification is enabled', () => {
    assert.equal(
      validateBatchQuotaInput({
        mode: 'override',
        amount: '0',
        quotaValue: 0,
        allUsers: false,
        selectedUserIds: [7, 9],
        sendEmail: true,
        emailSubject: ' ',
        emailContent: '<p>Updated</p>',
      }),
      'Email subject and content are required'
    )
  })
})
