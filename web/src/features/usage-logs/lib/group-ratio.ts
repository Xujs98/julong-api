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
import type { LogOtherData } from '../types'

export function getActualGroupRatio(other: LogOtherData | null): number | null {
  const userGroupRatio = other?.user_group_ratio
  if (
    userGroupRatio != null &&
    userGroupRatio !== -1 &&
    Number.isFinite(userGroupRatio)
  ) {
    return userGroupRatio
  }

  const groupRatio = other?.group_ratio
  return groupRatio != null && Number.isFinite(groupRatio) ? groupRatio : null
}

export function getDisplayedGroupRatio(
  other: LogOtherData | null,
  storedDisplayRatio?: number | null
): number | null {
  if (storedDisplayRatio != null && Number.isFinite(storedDisplayRatio)) {
    return storedDisplayRatio
  }

  const userGroupRatio = other?.user_group_ratio
  if (
    userGroupRatio != null &&
    userGroupRatio !== -1 &&
    Number.isFinite(userGroupRatio)
  ) {
    return userGroupRatio
  }

  const groupRatio = other?.group_ratio
  const forceDisplay = other?.group_ratio_display_mode != null
  if (
    groupRatio != null &&
    (groupRatio !== 1 || forceDisplay) &&
    Number.isFinite(groupRatio)
  ) {
    return groupRatio
  }

  return null
}

export interface AdminGroupRatioDetails {
  actual: number | null
  displayed: number | null
  displayEnabled: boolean
  applicable: boolean
}

export function getAdminGroupRatioDetails(
  other: LogOtherData | null,
  storedDisplayRatio?: number | null
): AdminGroupRatioDetails {
  const actual = getActualGroupRatio(other)
  const hasDisplaySetting = other?.user_group_ratio_display_enabled != null
  const displayEnabled = other?.user_group_ratio_display_enabled !== false
  const displayed = displayEnabled
    ? (storedDisplayRatio ?? other?.user_group_ratio_display_value ?? null)
    : null

  return {
    actual,
    displayed:
      displayed != null && Number.isFinite(displayed) ? displayed : null,
    displayEnabled,
    applicable: actual != null || hasDisplaySetting,
  }
}
