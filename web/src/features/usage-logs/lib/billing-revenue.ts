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

export type BillingRevenueLabelKey =
  | 'Original cost'
  | 'Group special ratio revenue'
  | 'Model ratio revenue'

export interface BillingRevenueItem {
  key: 'original_quota' | 'group_special_ratio' | 'model_token_adjustment'
  labelKey: BillingRevenueLabelKey
  quota: number
}

export function getBillingRevenueItems(
  other: LogOtherData | null,
  isAdmin: boolean
): BillingRevenueItem[] {
  if (!isAdmin) return []

  const revenue = other?.admin_info?.billing_revenue
  if (!revenue) return []

  const items: BillingRevenueItem[] = []
  if (
    revenue.original_quota != null &&
    Number.isFinite(revenue.original_quota)
  ) {
    items.push({
      key: 'original_quota',
      labelKey: 'Original cost',
      quota: revenue.original_quota,
    })
  }
  if (
    revenue.group_special_ratio != null &&
    Number.isFinite(revenue.group_special_ratio)
  ) {
    items.push({
      key: 'group_special_ratio',
      labelKey: 'Group special ratio revenue',
      quota: revenue.group_special_ratio,
    })
  }
  if (
    revenue.model_token_adjustment != null &&
    Number.isFinite(revenue.model_token_adjustment)
  ) {
    items.push({
      key: 'model_token_adjustment',
      labelKey: 'Model ratio revenue',
      quota: revenue.model_token_adjustment,
    })
  }
  return items
}
