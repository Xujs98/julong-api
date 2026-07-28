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

export type EmailTemplate = {
  event: string
  locale: EmailTemplateLocale
  label: string
  description: string
  category: string
  campaign_compatible: boolean
  placeholders: string[]
  subject: string
  content: string
  is_custom: boolean
}

export type EmailTemplateLocale = 'zh' | 'en'

export type DashboardReportEmailFrequency = 'daily' | 'weekly' | 'monthly'
export type RiskUserEmailLevel = 'medium' | 'high'

const DEFAULT_RISK_USER_EMAIL_LEVELS: RiskUserEmailLevel[] = ['medium', 'high']

export type DashboardReportEmailSchedule = {
  id: string
  frequency: DashboardReportEmailFrequency
  send_times: string[]
  weekday: number
  month_day: number
}

export type EmailSettingsConfig = {
  subscription_expiry_reminder_enabled: boolean
  low_balance_email_enabled: boolean
  low_balance_email_threshold: number
  low_balance_email_recharge_url: string
  account_quota_email_enabled: boolean
  account_quota_email_threshold: number
  account_quota_email_recipient_user_ids: number[]
  channel_anomaly_email_enabled: boolean
  channel_anomaly_email_recipient_user_ids: number[]
  dashboard_report_email_enabled: boolean
  dashboard_report_email_frequency: DashboardReportEmailFrequency
  dashboard_report_email_send_time: string
  dashboard_report_email_weekday: number
  dashboard_report_email_month_day: number
  dashboard_report_email_recipient_user_ids: number[]
  dashboard_report_email_schedules: DashboardReportEmailSchedule[]
  risk_user_email_enabled: boolean
  risk_user_email_levels: RiskUserEmailLevel[]
  risk_user_email_recipient_user_ids: number[]
}

export type EmailRecipientOption = {
  id: number
  username: string
  display_name: string
  email: string
}

export type PageData<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type EmailTemplatePreview = {
  subject: string
  content: string
}

type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export async function listEmailTemplates() {
  const response = await api.get<ApiResponse<EmailTemplate[]>>(
    '/api/email-settings/templates'
  )
  return response.data
}

export async function updateEmailTemplate(
  event: string,
  locale: EmailTemplateLocale,
  subject: string,
  content: string
) {
  const response = await api.put<ApiResponse<EmailTemplate>>(
    `/api/email-settings/templates/${encodeURIComponent(event)}`,
    { locale, subject, content }
  )
  return response.data
}

export async function previewEmailTemplate(
  event: string,
  locale: EmailTemplateLocale,
  subject: string,
  content: string
) {
  const response = await api.post<ApiResponse<EmailTemplatePreview>>(
    `/api/email-settings/templates/${encodeURIComponent(event)}/preview`,
    { locale, subject, content }
  )
  return response.data
}

export async function resetEmailTemplate(
  event: string,
  locale: EmailTemplateLocale
) {
  const response = await api.post<ApiResponse<EmailTemplate>>(
    `/api/email-settings/templates/${encodeURIComponent(event)}/reset`,
    { locale }
  )
  return response.data
}

export async function getEmailSettingsConfig() {
  const response = await api.get<ApiResponse<EmailSettingsConfig>>(
    '/api/email-settings/config',
    { params: { _: Date.now() } }
  )
  const config = response.data.data
  let dashboardReportSchedules: DashboardReportEmailSchedule[] = []
  if (config) {
    if (config.dashboard_report_email_schedules?.length) {
      dashboardReportSchedules = config.dashboard_report_email_schedules
    } else {
      dashboardReportSchedules = [
        {
          id: crypto.randomUUID(),
          frequency: config.dashboard_report_email_frequency ?? 'daily',
          send_times: [config.dashboard_report_email_send_time ?? '08:00'],
          weekday: config.dashboard_report_email_weekday ?? 1,
          month_day: config.dashboard_report_email_month_day ?? 1,
        },
      ]
    }
  }
  return {
    ...response.data,
    data: config
      ? {
          ...config,
          account_quota_email_recipient_user_ids:
            config.account_quota_email_recipient_user_ids ?? [],
          channel_anomaly_email_recipient_user_ids:
            config.channel_anomaly_email_recipient_user_ids ?? [],
          dashboard_report_email_frequency:
            config.dashboard_report_email_frequency ?? 'daily',
          dashboard_report_email_send_time:
            config.dashboard_report_email_send_time ?? '08:00',
          dashboard_report_email_weekday:
            config.dashboard_report_email_weekday ?? 1,
          dashboard_report_email_month_day:
            config.dashboard_report_email_month_day ?? 1,
          dashboard_report_email_recipient_user_ids:
            config.dashboard_report_email_recipient_user_ids ?? [],
          dashboard_report_email_schedules: dashboardReportSchedules,
          risk_user_email_enabled: config.risk_user_email_enabled ?? false,
          risk_user_email_levels: config.risk_user_email_levels?.length
            ? config.risk_user_email_levels
            : [...DEFAULT_RISK_USER_EMAIL_LEVELS],
          risk_user_email_recipient_user_ids:
            config.risk_user_email_recipient_user_ids ?? [],
        }
      : undefined,
  }
}

export async function updateEmailSettingsConfig(config: EmailSettingsConfig) {
  const response = await api.put<ApiResponse<EmailSettingsConfig>>(
    '/api/email-settings/config',
    config
  )
  return response.data
}

export async function searchEmailSettingsRecipients(
  keyword: string,
  page: number,
  pageSize: number
) {
  const response = await api.get<ApiResponse<PageData<EmailRecipientOption>>>(
    '/api/email-settings/recipients',
    { params: { keyword, p: page, page_size: pageSize } }
  )
  return response.data
}

export async function resolveEmailSettingsRecipients(userIds: number[]) {
  const response = await api.post<ApiResponse<EmailRecipientOption[]>>(
    '/api/email-settings/recipients/resolve',
    { user_ids: userIds }
  )
  return response.data
}

export async function sendChannelAnomalyTestEmail(recipientUserIds: number[]) {
  const response = await api.post<ApiResponse<{ recipient_count: number }>>(
    '/api/email-settings/channel-anomaly/test',
    { recipient_user_ids: recipientUserIds }
  )
  return response.data
}

export async function sendDashboardReportTestEmail(recipientUserIds: number[]) {
  const response = await api.post<
    ApiResponse<{ recipient_count: number; period: string }>
  >('/api/email-settings/dashboard-report/test', {
    recipient_user_ids: recipientUserIds,
  })
  return response.data
}

export async function sendRiskUserTestEmail(
  recipientUserIds: number[],
  riskLevels: RiskUserEmailLevel[]
) {
  const response = await api.post<
    ApiResponse<{
      recipient_count: number
      risk_user_count: number
      levels: RiskUserEmailLevel[]
    }>
  >('/api/email-settings/risk-user/test', {
    recipient_user_ids: recipientUserIds,
    risk_levels: riskLevels,
  })
  return response.data
}
