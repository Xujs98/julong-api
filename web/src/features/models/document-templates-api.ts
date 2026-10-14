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
import type { QueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import { createServerError } from '@/lib/server-error-message'

export type DocumentTemplate = {
  id: number
  name: string
  format: 'markdown' | 'docx'
  original_filename: string
  content?: string
  created_time: number
  updated_time: number
}

export type DocumentTemplateListParams = {
  p?: number
  page_size?: number
  keyword?: string
}

type DocumentTemplateResponse<T> = { success: boolean; data: T }

export const documentTemplatesQueryKeys = {
  all: ['model-document-templates'] as const,
  lists: ['model-document-templates', 'list'] as const,
  list: (params: DocumentTemplateListParams) =>
    ['model-document-templates', 'list', params] as const,
  detail: (id: number) => ['model-document-templates', 'detail', id] as const,
}

export async function listDocumentTemplates(
  params: DocumentTemplateListParams = {}
) {
  const response = await api.get<
    DocumentTemplateResponse<{
      items: DocumentTemplate[]
      total: number
      page: number
      page_size: number
    }>
  >('/api/model-document-templates', { params })
  if (!response.data.success) throw createServerError(response.data)
  return response.data.data
}

export async function getDocumentTemplate(
  id: number
): Promise<DocumentTemplate> {
  const response = await api.get<DocumentTemplateResponse<DocumentTemplate>>(
    `/api/model-document-templates/${id}`
  )
  if (!response.data.success) throw createServerError(response.data)
  return response.data.data
}

export async function uploadDocumentTemplate(
  file: File,
  name: string,
  id?: number
): Promise<DocumentTemplate> {
  const data = new FormData()
  data.append('file', file)
  data.append('name', name)
  const response = id
    ? await api.put<DocumentTemplateResponse<DocumentTemplate>>(
        `/api/model-document-templates/${id}`,
        data
      )
    : await api.post<DocumentTemplateResponse<DocumentTemplate>>(
        '/api/model-document-templates',
        data
      )
  if (!response.data.success) throw createServerError(response.data)
  return response.data.data
}

export async function updateDocumentTemplate(
  id: number,
  patch: { name?: string; content?: string }
): Promise<DocumentTemplate> {
  const response = await api.put<DocumentTemplateResponse<DocumentTemplate>>(
    `/api/model-document-templates/${id}`,
    patch
  )
  if (!response.data.success) throw createServerError(response.data)
  return response.data.data
}

export async function deleteDocumentTemplate(id: number): Promise<void> {
  const response = await api.delete<DocumentTemplateResponse<null>>(
    `/api/model-document-templates/${id}`
  )
  if (!response.data.success) throw createServerError(response.data)
}

export async function invalidateDocumentTemplateData(client: QueryClient) {
  await Promise.all([
    client.invalidateQueries({ queryKey: documentTemplatesQueryKeys.all }),
    client.invalidateQueries({ queryKey: ['models'] }),
    client.invalidateQueries({ queryKey: ['model-documentation'] }),
  ])
}
