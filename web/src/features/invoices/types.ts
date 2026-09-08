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
export type InvoiceStatus =
  | 'pending'
  | 'processing'
  | 'completed'
  | 'rejected'
  | 'non_reusable'
  | 'cancelled'

export type InvoiceCredit = {
  request_id: string
  created_at: number
  quota: number
  amount: number
  source: 'online_recharge' | 'redemption' | 'admin_adjustment'
  content: string
  frozen_until?: number
}

export type InvoiceItem = {
  id: number
  application_id: number
  user_id: number
  source_request_id: string
  source: InvoiceCredit['source']
  quota: number
  source_created_at: number
  released_at: number
  frozen_until: number
  amount: number
  content: string
}

export type InvoiceApplication = {
  id: number
  user_id: number
  username?: string
  email: string
  amount_quota: number
  amount: number
  unit: string
  status: InvoiceStatus
  admin_note: string
  created_at: number
  updated_at: number
  completed_at: number
  has_attachment: boolean
  items?: InvoiceItem[]
}

export type InvoiceInfo = {
  enabled: boolean
  minimum_amount: number
  unit: string
  processing_days: string
  rejection_freeze_hours: number
  credits: InvoiceCredit[]
  credits_total: number
  applications: InvoiceApplication[]
  applications_total: number
  page_size: number
}

export type InvoiceEmailTemplate = {
  event: string
  locale: 'zh' | 'en'
  label: string
  subject: string
}

export type InvoiceApplicationPage = {
  page: number
  page_size: number
  total: number
  items: InvoiceApplication[]
}

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
