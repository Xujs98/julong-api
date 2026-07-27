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
import type { UserGroupUsage } from '../types'

export interface UserGroupUsageItem {
  group: string
  ratio: number
  quota: number
  tokenUsed: number
}

export const userGroupRatiosCardClassName =
  'mb-4 rounded-md border bg-muted/30 p-3 text-left'

export function getSortedUserGroupUsage(
  groupUsage: Record<string, UserGroupUsage>
): UserGroupUsageItem[] {
  return Object.entries(groupUsage)
    .map(([group, usage]) => ({
      group,
      ratio: usage.ratio,
      quota: usage.quota,
      tokenUsed: usage.token_used,
    }))
    .sort((left, right) => left.group.localeCompare(right.group))
}

export function formatUserGroupRatio(ratio: number): string {
  return ratio % 1 === 0
    ? String(ratio)
    : ratio.toFixed(4).replace(/\.?0+$/, '')
}
