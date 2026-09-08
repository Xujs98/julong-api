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

import type {
  ApiResponse,
  InvoiceApplication,
  InvoiceApplicationPage,
  InvoiceInfo,
  InvoiceEmailTemplate,
  InvoiceStatus,
} from './types'

export async function getInvoiceInfo(
  creditPage: number,
  applicationPage: number,
  pageSize: number
): Promise<ApiResponse<InvoiceInfo>> {
  const response = await api.get('/api/user/invoice', {
    params: {
      credit_page: creditPage,
      application_page: applicationPage,
      page_size: pageSize,
    },
  })
  return response.data
}

export async function getInvoiceEmailTemplates(): Promise<
  ApiResponse<InvoiceEmailTemplate[]>
> {
  const response = await api.get('/api/user/invoice/email-templates')
  return response.data
}

export async function applyForInvoice(
  requestIds: string[]
): Promise<ApiResponse<InvoiceApplication>> {
  const response = await api.post('/api/user/invoice/apply', {
    request_ids: requestIds,
  })
  return response.data
}

export async function cancelInvoiceApplication(
  id: number
): Promise<ApiResponse<InvoiceApplication>> {
  const response = await api.post(`/api/user/invoice/applications/${id}/cancel`)
  return response.data
}

export async function downloadInvoiceAttachment(id: number) {
  const response = await api.get<Blob>(
    `/api/user/invoice/applications/${id}/attachment`,
    {
      responseType: 'blob',
      skipBusinessError: true,
    }
  )
  if (String(response.headers['content-type']).includes('application/json')) {
    const error = JSON.parse(await response.data.text()) as ApiResponse<never>
    throw new Error(error.message || 'Failed to download invoice')
  }
  return response
}

export async function getAdminInvoiceApplications(
  page: number,
  pageSize: number
): Promise<ApiResponse<InvoiceApplicationPage>> {
  const response = await api.get('/api/user/invoice/applications', {
    params: { p: page, page_size: pageSize },
  })
  return response.data
}

export async function updateInvoiceApplication(
  id: number,
  status: InvoiceStatus,
  adminNote: string
): Promise<ApiResponse<InvoiceApplication>> {
  const response = await api.put(`/api/user/invoice/applications/${id}`, {
    status,
    admin_note: adminNote,
  })
  return response.data
}

export async function completeInvoiceApplication(
  id: number,
  locale: 'zh' | 'en',
  adminNote: string,
  attachment: File
): Promise<ApiResponse<InvoiceApplication>> {
  const form = new FormData()
  form.append('locale', locale)
  form.append('admin_note', adminNote)
  form.append('attachment', attachment)
  const response = await api.post(
    `/api/user/invoice/applications/${id}/complete`,
    form
  )
  return response.data
}

export async function withdrawCompletedInvoiceApplication(
  id: number
): Promise<ApiResponse<InvoiceApplication>> {
  const response = await api.post(
    `/api/user/invoice/applications/${id}/withdraw`
  )
  return response.data
}
