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
import { z } from 'zod'

import type { Redemption } from '@/features/redemption-codes/types'
import type { AdminPermissionMatrix } from '@/lib/admin-permissions'

// ============================================================================
// User Schema & Types
// ============================================================================

/** User status: 1 = enabled, 2 = disabled, 3+ = other states */
export const userStatusSchema = z.number()
export type UserStatus = z.infer<typeof userStatusSchema>

/** User role: 1 = common user, 10 = admin, 100 = root */
export const userRoleSchema = z.number()
export type UserRole = z.infer<typeof userRoleSchema>

export const userTagSchema = z.object({
  id: z.number(),
  name: z.string(),
  color: z.string(),
  built_in: z.boolean().optional(),
  risk_level: z.enum(['medium', 'high']).optional(),
  created_at: z.number().optional(),
  updated_at: z.number().optional(),
})
export type UserTag = z.infer<typeof userTagSchema>

export const userSchema = z.object({
  id: z.number(),
  username: z.string(),
  display_name: z.string(),
  password: z.string().optional(),
  github_id: z.string().optional(),
  oidc_id: z.string().optional(),
  wechat_id: z.string().optional(),
  telegram_id: z.string().optional(),
  email: z.string().optional(),
  quota: z.number(),
  used_quota: z.number(),
  request_count: z.number(),
  group: z.string(),
  group_ratio_adjustment_enabled: z.boolean().optional(),
  group_ratio_adjustment: z.number().optional(),
  aff_code: z.string().optional(),
  aff_count: z.number().optional(),
  aff_quota: z.number().optional(),
  aff_history_quota: z.number().optional(),
  inviter_id: z.number().optional(),
  is_agent: z.boolean().optional(),
  agent_discount: z.number().optional(),
  agent_topup_link: z.string().optional(),
  agent_id: z.number().optional(),
  agent_username: z.string().optional(),
  tag_id: z.number().optional(),
  tag: userTagSchema.nullable().optional(),
  risk_tag: userTagSchema.nullable().optional(),
  linux_do_id: z.string().optional(),
  status: userStatusSchema,
  role: userRoleSchema,
  created_at: z.number().optional(),
  updated_at: z.number().optional(),
  last_login_at: z.number().optional(),
  last_login_ip: z.string().optional(),
  shared_ip_user_count: z.number().optional(),
  last_login_ip_blocked: z.boolean().optional(),
  DeletedAt: z.any().nullable().optional(),
  remark: z.string().optional(),
  admin_permissions: z
    .record(z.string(), z.record(z.string(), z.boolean()))
    .optional(),
})
export type User = z.infer<typeof userSchema>

export const userListSchema = z.array(userSchema)

// ============================================================================
// API Request/Response Types
// ============================================================================

/** Generic API response */
export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export type UserSortBy =
  | 'id'
  | 'username'
  | 'quota'
  | 'group'
  | 'created_at'
  | 'last_login_at'

export type UserSortOrder = 'asc' | 'desc'

export interface GetUsersParams {
  p?: number
  page_size?: number
  sort_by?: UserSortBy
  sort_order?: UserSortOrder
}

export interface GetUsersResponse {
  success: boolean
  message?: string
  data?: {
    items: User[]
    total: number
    page: number
    page_size: number
  }
}

export interface SearchUsersParams {
  keyword?: string
  group?: string
  role?: string
  status?: string
  tag_id?: string
  p?: number
  page_size?: number
  sort_by?: UserSortBy
  sort_order?: UserSortOrder
}

export interface UserFormData {
  username: string
  display_name: string
  password?: string
  role?: number // Only used when creating user
  quota?: number // Only used when updating user
  group?: string // Only used when updating user
  group_ratio_adjustment_enabled?: boolean
  group_ratio_adjustment?: number
  remark?: string // Only used when updating user
  is_agent?: boolean
  agent_discount?: number
  agent_topup_link?: string
  admin_permissions?: AdminPermissionMatrix
}

export type ManageUserAction =
  | 'promote'
  | 'demote'
  | 'enable'
  | 'disable'
  | 'delete'
  | 'add_quota'

export type QuotaAdjustMode = 'add' | 'subtract' | 'override'

export interface ManageUserQuotaPayload {
  id: number
  action: 'add_quota'
  mode: QuotaAdjustMode
  value: number
}

export interface UserManagementOption {
  id: number
  username: string
  display_name: string
  email: string
}

export interface PageData<T> {
  page: number
  page_size: number
  total: number
  items: T[]
}

export interface BatchQuotaAdjustPayload {
  mode: QuotaAdjustMode
  value: number
  all_users: boolean
  user_ids: number[]
  send_email: boolean
  email_locale: 'zh' | 'en'
  email_subject: string
  email_content: string
}

export interface BatchQuotaAdjustResult {
  adjusted_count: number
  email_success_count: number
  email_skipped_count: number
  email_failed_count: number
}

// ============================================================================
// Dialog Types
// ============================================================================

export interface AgentDetailData {
  agent: User
  users: User[]
  redemptions: Redemption[]
}

export interface UserUsageSummary {
  total_tokens: number
  today_tokens: number
  today_quota: number
  base_group_ratios: Record<string, number>
  group_ratios: Record<string, number>
  group_usage?: Record<string, UserGroupUsage>
}

export type UserRiskLevel = 'low' | 'medium' | 'high'

export type UserRiskSignalCode =
  | 'sensitive_word_attempts'
  | 'failed_request_rate'
  | 'client_abort'
  | 'abnormal_stream'
  | 'failed_refunds'
  | 'refund_after_output'
  | 'multiple_ips'

export interface UserRiskSignal {
  code: UserRiskSignalCode
  severity: UserRiskLevel
  score: number
  count: number
  last_seen: number
}

export interface UserRiskReport {
  user_id: number
  enabled: boolean
  global_enabled: boolean
  user_enabled: boolean
  window_days: 1 | 7 | 30
  start_time: number
  end_time: number
  generated_at: number
  score: number
  level: UserRiskLevel
  summary: {
    total_requests: number
    error_count: number
    error_rate: number
    refund_count: number
    refund_quota: number
    failed_refund_count: number
    refund_after_output_count: number
    sensitive_word_attempts: number
    client_abort_count: number
    abnormal_stream_count: number
    unique_ip_count: number
  }
  signals: UserRiskSignal[]
}

export interface UserGroupUsage {
  ratio: number
  quota: number
  token_used: number
}

export interface UserQuotaIncreaseLog {
  id: number
  request_id: string
  created_at: number
  quota: number
  source: string
  content: string
}

export interface UserQuotaIncreaseLogPage {
  items: UserQuotaIncreaseLog[]
  total: number
  page: number
  page_size: number
}

export interface UserLoginIP {
  ip: string
  last_login_at: number
  login_count: number
  blocked: boolean
  shared_user_count: number
}

export interface UserLoginDevice {
  device_id: string
  user_agent: string
  last_login_at: number
  login_count: number
  active_session_count: number
  ips: string[]
  blocked: boolean
}

export type UserRequestContentStatus = 'pending' | 'success' | 'error'

export interface UserRequestContentLog {
  id: number
  user_id: number
  request_id: string
  created_at: number
  model_name: string
  token_name: string
  request_path: string
  status: UserRequestContentStatus
  error_message?: string
  original_size: number
  captured_size: number
  truncated: boolean
}

export interface UserRequestContentLogList {
  enabled: boolean
  items: UserRequestContentLog[]
  max_items: number
}

export interface UserRequestContentLogDetail {
  log: UserRequestContentLog
  content: unknown
}

export type UsersDialogType =
  | 'create'
  | 'update'
  | 'delete'
  | 'agent-detail'
  | 'user-detail'
  | 'tags'
  | 'batch-quota'
