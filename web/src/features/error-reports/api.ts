import { api } from '@/lib/api'

import type {
  ApiResponse,
  ErrorReport,
  ErrorReportListResponse,
  SubmitErrorReportData,
} from './types'

export async function submitErrorReport(
  data: SubmitErrorReportData
): Promise<ApiResponse<ErrorReport>> {
  const res = await api.post('/api/error-reports', data)
  return res.data
}

export async function getErrorReports(params: {
  p: number
  page_size: number
}): Promise<ApiResponse<ErrorReportListResponse>> {
  const search = new URLSearchParams()
  search.set('p', String(params.p))
  search.set('page_size', String(params.page_size))
  const res = await api.get(`/api/error-reports?${search.toString()}`)
  return res.data
}

export async function getErrorReport(
  id: number
): Promise<ApiResponse<ErrorReport>> {
  const res = await api.get(`/api/error-reports/${id}`)
  return res.data
}
