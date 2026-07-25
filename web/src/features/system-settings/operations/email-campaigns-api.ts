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
import { api } from '@/lib/api'

export type EmailCampaignMode = 'immediate' | 'scheduled' | 'conditional'
export type EmailCampaignTarget =
  | 'all_users'
  | 'active_subscribers'
  | 'selected_users'
export type EmailCampaignStatus =
  | 'draft'
  | 'scheduled'
  | 'active'
  | 'running'
  | 'completed'
  | 'partial_failed'
  | 'paused'

export type EmailCampaign = {
  id: number
  name: string
  subject: string
  content: string
  mode: EmailCampaignMode
  target_type: EmailCampaignTarget
  target_user_ids: number[]
  trigger_type: string
  trigger_days: number
  scheduled_at: number
  next_run_at: number
  last_run_at: number
  status: EmailCampaignStatus
  created_by: number
  recipient_count: number
  success_count: number
  failed_count: number
  skipped_count: number
  last_error: string
  created_at: number
  updated_at: number
}

export type EmailDelivery = {
  id: number
  campaign_id: number
  user_id: number
  email: string
  username: string
  display_name: string
  subscription_id: number
  subscription_title: string
  subscription_end_time: number
  status: 'pending' | 'sending' | 'sent' | 'failed' | 'skipped'
  attempt_count: number
  last_error: string
  sent_at: number
  created_at: number
  updated_at: number
}

export type EmailCampaignPayload = {
  name: string
  subject: string
  content: string
  mode: EmailCampaignMode
  target_type: EmailCampaignTarget
  target_user_ids: number[]
  trigger_type: string
  trigger_days: number
  scheduled_at: number
  draft?: boolean
}

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export type PageData<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type EmailCampaignStats = {
  campaign_count: number
  recipient_count: number
  success_count: number
  failed_count: number
}

export async function listEmailCampaigns(page: number, pageSize: number) {
  const response = await api.get<ApiResponse<PageData<EmailCampaign>>>(
    '/api/email-campaigns',
    { params: { p: page, page_size: pageSize } }
  )
  return response.data
}

export async function getEmailCampaignStats() {
  const response = await api.get<ApiResponse<EmailCampaignStats>>(
    '/api/email-campaigns/stats'
  )
  return response.data
}

export async function createEmailCampaign(payload: EmailCampaignPayload) {
  const response = await api.post<ApiResponse<EmailCampaign>>(
    '/api/email-campaigns',
    payload
  )
  return response.data
}

export async function updateEmailCampaign(
  id: number,
  payload: EmailCampaignPayload
) {
  const response = await api.put<ApiResponse<EmailCampaign>>(
    `/api/email-campaigns/${id}`,
    payload
  )
  return response.data
}

export async function previewEmailCampaign(payload: EmailCampaignPayload) {
  const response = await api.post<ApiResponse<{ recipient_count: number }>>(
    '/api/email-campaigns/preview',
    payload
  )
  return response.data
}

export async function activateEmailCampaign(id: number) {
  const response = await api.post<ApiResponse<EmailCampaign>>(
    `/api/email-campaigns/${id}/activate`
  )
  return response.data
}

export async function pauseEmailCampaign(id: number) {
  const response = await api.post<ApiResponse<null>>(
    `/api/email-campaigns/${id}/pause`
  )
  return response.data
}

export async function retryEmailCampaign(id: number) {
  const response = await api.post<ApiResponse<null>>(
    `/api/email-campaigns/${id}/retry`
  )
  return response.data
}

export async function deleteEmailCampaign(id: number) {
  const response = await api.delete<ApiResponse<null>>(
    `/api/email-campaigns/${id}`
  )
  return response.data
}

export async function listEmailDeliveries(
  campaignId: number,
  page: number,
  pageSize: number
) {
  const response = await api.get<ApiResponse<PageData<EmailDelivery>>>(
    `/api/email-campaigns/${campaignId}/deliveries`,
    { params: { p: page, page_size: pageSize } }
  )
  return response.data
}
