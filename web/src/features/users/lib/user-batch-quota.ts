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
import type { QuotaAdjustMode } from '../types'

type BatchQuotaValidationInput = {
  mode: QuotaAdjustMode
  amount: string
  quotaValue: number
  allUsers: boolean
  selectedUserIds: number[]
  sendEmail: boolean
  emailSubject: string
  emailContent: string
}

export function validateBatchQuotaInput(
  input: BatchQuotaValidationInput
): string | null {
  if (
    input.amount.trim() === '' ||
    !Number.isFinite(Number(input.amount)) ||
    !Number.isInteger(input.quotaValue)
  ) {
    return 'Enter a valid quota amount'
  }
  if (input.mode !== 'override' && input.quotaValue <= 0) {
    return 'Quota amount must be greater than zero'
  }
  if (!input.allUsers && input.selectedUserIds.length === 0) {
    return 'Select at least one user'
  }
  if (
    input.sendEmail &&
    (!input.emailSubject.trim() || !input.emailContent.trim())
  ) {
    return 'Email subject and content are required'
  }
  return null
}
