export type ApiResponse<T = unknown> = {
  success: boolean
  message?: string
  data?: T
}

export type ErrorReport = {
  id: number
  created_at: number
  user_id: number
  username: string
  title: string
  message: string
  page_url: string
  error_status: number
  user_agent: string
  stack: string
  ip: string
}

export type ErrorReportListResponse = {
  items: ErrorReport[]
  total: number
  page: number
  page_size: number
}

export type SubmitErrorReportData = {
  title: string
  message: string
  page_url: string
  error_status: number
  stack?: string
}
