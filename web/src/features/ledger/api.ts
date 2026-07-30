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
  LedgerEntry,
  LedgerListParams,
  LedgerMutation,
  LedgerPage,
  LedgerSettings,
  LedgerSummary,
} from './types'

const LEDGER_API_CACHE_VERSION = 2

function ledgerQueryParams<T extends LedgerListParams>(params: T) {
  return {
    ...params,
    ledger_api_version: LEDGER_API_CACHE_VERSION,
  }
}

export async function getLedgerEntries(
  params: LedgerListParams
): Promise<ApiResponse<LedgerPage>> {
  const response = await api.get('/api/ledger', {
    params: ledgerQueryParams(params),
  })
  return response.data
}

export async function getLedgerSummary(
  params: Pick<LedgerListParams, 'start_timestamp' | 'end_timestamp'>
): Promise<ApiResponse<LedgerSummary>> {
  const response = await api.get('/api/ledger/summary', {
    params: ledgerQueryParams(params),
  })
  return response.data
}

export async function getLedgerSettings(): Promise<
  ApiResponse<LedgerSettings>
> {
  const response = await api.get('/api/ledger/settings', {
    params: ledgerQueryParams({}),
  })
  return response.data
}

export async function updateLedgerSettings(
  estimateRatio: number
): Promise<ApiResponse<LedgerSettings>> {
  const response = await api.put('/api/ledger/settings', {
    estimate_ratio: String(estimateRatio),
  })
  return response.data
}

export async function createLedgerEntry(
  data: LedgerMutation
): Promise<ApiResponse<LedgerEntry>> {
  const response = await api.post('/api/ledger', data)
  return response.data
}

export async function updateLedgerEntry(
  id: number,
  data: LedgerMutation
): Promise<ApiResponse<LedgerEntry>> {
  const response = await api.put(`/api/ledger/${id}`, data)
  return response.data
}

export async function deleteLedgerEntries(
  ids: number[]
): Promise<ApiResponse<number>> {
  const response = await api.post('/api/ledger/batch-delete', { ids })
  return response.data
}

export async function downloadLedgerEntries(params: LedgerListParams): Promise<{
  blob: Blob
  filename: string
  truncated: boolean
  limit: number
}> {
  const response = await api.get('/api/ledger/export', {
    params: ledgerQueryParams(params),
    responseType: 'blob',
    disableDuplicate: true,
  })
  const disposition = String(response.headers['content-disposition'] || '')
  const filenameMatch = disposition.match(/filename="?([^";]+)"?/i)
  return {
    blob: response.data as Blob,
    filename:
      filenameMatch?.[1] ||
      `ledger-${new Date().toISOString().slice(0, 10)}.csv`,
    truncated: response.headers['x-export-truncated'] === 'true',
    limit: Number(response.headers['x-export-row-limit']) || 5000,
  }
}
