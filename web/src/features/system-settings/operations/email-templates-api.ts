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
  label: string
  description: string
  placeholders: string[]
  subject: string
  content: string
  is_custom: boolean
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
  subject: string,
  content: string
) {
  const response = await api.put<ApiResponse<EmailTemplate>>(
    `/api/email-settings/templates/${encodeURIComponent(event)}`,
    { subject, content }
  )
  return response.data
}

export async function previewEmailTemplate(
  event: string,
  subject: string,
  content: string
) {
  const response = await api.post<ApiResponse<EmailTemplatePreview>>(
    `/api/email-settings/templates/${encodeURIComponent(event)}/preview`,
    { subject, content }
  )
  return response.data
}

export async function resetEmailTemplate(event: string) {
  const response = await api.post<ApiResponse<EmailTemplate>>(
    `/api/email-settings/templates/${encodeURIComponent(event)}/reset`
  )
  return response.data
}
