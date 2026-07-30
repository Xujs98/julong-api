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
export type LedgerPlatform =
  | 'Anthropic'
  | 'OpenAI'
  | 'Gemini'
  | 'Antigravity'
  | 'Grok'

export interface LedgerEntry {
  id: number
  platform: LedgerPlatform
  account: string
  email: string
  type: string
  quota: number
  cost_price: string
  quantity: number
  occurred_at: number
  created_by: number
  created_at: number
  updated_at: number
}

export interface LedgerMutation {
  platform: LedgerPlatform
  account: string
  email: string
  type: string
  quota: number
  cost_price: string
  quantity: number
  occurred_at: number
}

export interface LedgerListParams {
  p?: number
  page_size?: number
  start_timestamp?: number
  end_timestamp?: number
}

export interface LedgerPage {
  page: number
  page_size: number
  total: number
  items: LedgerEntry[]
}

export interface LedgerQuotaMetric {
  real: number
  estimated: number
}

export interface LedgerSettings {
  estimate_ratio: string
}

export interface LedgerSummary {
  user_quota: LedgerQuotaMetric
  usage_quota: LedgerQuotaMetric
  daily_operating_cost: string
  total_operating_cost: string
  operational_quota: {
    real: number
    cost_ratio: string | null
  }
  cost_ratios: {
    plus: string | null
    pro: string | null
    k12: string | null
  }
  estimate_ratio: string
  days: number
  ledger_entry_count: number
  included_user_count: number
}

export interface ApiResponse<T> {
  success: boolean
  message?: string
  data?: T
}
