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
import type { UserQuotaSummaryParams } from '../types'

export type UserQuotaSummarySearchState = {
  filter?: string
  group?: string
  role?: string[]
  status?: string[]
  tag?: string[]
}

export function getUserQuotaSummaryParams(
  search: UserQuotaSummarySearchState
): UserQuotaSummaryParams {
  return {
    keyword: search.filter ?? '',
    group: search.group ?? '',
    role: search.role?.[0] ?? '',
    status: search.status?.[0] ?? '',
    tag_id: search.tag?.[0] ?? '',
  }
}
