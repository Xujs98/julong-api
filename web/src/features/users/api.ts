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
import type { AxiosRequestConfig } from 'axios'

import type { PermissionCatalog } from '@/lib/admin-permissions'
import { api } from '@/lib/api'
import type { CustomOAuthBinding } from '@/lib/oauth'
import { requireServerSuccess } from '@/lib/server-error-message'

import { userQuotaSummaryQueryParams } from './lib/api-cache-version'
import type {
  User,
  GetUsersParams,
  GetUsersResponse,
  SearchUsersParams,
  UserFormData,
  ManageUserAction,
  ManageUserQuotaPayload,
  ApiResponse,
  AgentDetailData,
  UserQuotaIncreaseLogPage,
  UserUsageSummary,
  UserLoginIP,
  UserRequestContentLogList,
  UserRequestContentLogDetail,
  UserTag,
  UserManagementOption,
  PageData,
  BatchQuotaAdjustPayload,
  BatchQuotaAdjustResult,
  UserRiskReport,
  UserLoginDevice,
  UserQuotaSummary,
  UserQuotaSummaryParams,
  UserQuotaSummarySettings,
} from './types'

// A step-up proof is single-use, so the request carrying it must never be
// replayed by the auth-refresh interceptor; it refreshes first instead.
function securityProofConfig(proofToken?: string): AxiosRequestConfig {
  if (!proofToken) return {}
  return {
    headers: { 'X-Security-Proof': proofToken },
    singleUseAuthorization: true,
  }
}

// ============================================================================
// User Management APIs
// ============================================================================

/**
 * Get paginated users list
 */
export async function getUsers(
  params: GetUsersParams = {}
): Promise<GetUsersResponse> {
  const { p = 1, page_size = 10, sort_by, sort_order } = params
  const res = await api.get('/api/user/', {
    params: {
      p,
      page_size,
      sort_by,
      sort_order,
    },
  })
  return res.data
}

/**
 * Search users by keyword or group
 */
export async function searchUsers(
  params: SearchUsersParams
): Promise<GetUsersResponse> {
  const {
    keyword = '',
    group = '',
    role = '',
    status = '',
    tag_id = '',
    p = 1,
    page_size = 10,
    sort_by,
    sort_order,
  } = params
  const queryParams = new URLSearchParams()
  queryParams.set('keyword', keyword)
  queryParams.set('group', group)
  if (role) queryParams.set('role', role)
  if (status) queryParams.set('status', status)
  if (tag_id) queryParams.set('tag_id', tag_id)
  queryParams.set('p', String(p))
  queryParams.set('page_size', String(page_size))
  if (sort_by) queryParams.set('sort_by', sort_by)
  if (sort_order) queryParams.set('sort_order', sort_order)
  const res = await api.get(`/api/user/search?${queryParams.toString()}`)
  return res.data
}

export async function getUserQuotaSummary(
  params: UserQuotaSummaryParams
): Promise<ApiResponse<UserQuotaSummary>> {
  const res = await api.get('/api/user/quota-summary', {
    params: userQuotaSummaryQueryParams(params),
  })
  return res.data
}

export async function getUserQuotaSummarySettings(): Promise<
  ApiResponse<UserQuotaSummarySettings>
> {
  const res = await api.get('/api/user/quota-summary/settings', {
    params: userQuotaSummaryQueryParams({}),
  })
  return res.data
}

export async function updateUserQuotaSummarySettings(
  excludedUserIds: number[]
): Promise<ApiResponse<UserQuotaSummarySettings>> {
  const res = await api.put('/api/user/quota-summary/settings', {
    excluded_user_ids: excludedUserIds,
  })
  return res.data
}

export async function searchUserQuotaSummaryOptions(
  keyword: string,
  page: number,
  pageSize: number
): Promise<ApiResponse<PageData<UserManagementOption>>> {
  const res = await api.get('/api/user/quota-summary/users', {
    params: userQuotaSummaryQueryParams({
      keyword,
      p: page,
      page_size: pageSize,
    }),
  })
  return res.data
}

export async function resolveUserQuotaSummaryOptions(
  userIds: number[]
): Promise<ApiResponse<UserManagementOption[]>> {
  const res = await api.post('/api/user/quota-summary/users/resolve', {
    user_ids: userIds,
  })
  return res.data
}

/**
 * Get single user by ID
 */
export async function getUser(id: number): Promise<ApiResponse<User>> {
  const res = await api.get(`/api/user/${id}`)
  return res.data
}

/**
 * Create a new user. Creating an administrator requires an
 * `admin.user.create` proof.
 */
export async function createUser(
  data: UserFormData,
  proofToken?: string
): Promise<ApiResponse<User>> {
  const res = await api.post(
    '/api/user/',
    data,
    securityProofConfig(proofToken)
  )
  return res.data
}

/**
 * Update an existing user. Changing the password or the admin permission
 * matrix requires an `admin.user.update` proof.
 */
export async function updateUser(
  data: UserFormData & { id: number },
  proofToken?: string
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.put('/api/user/', data, securityProofConfig(proofToken))
  return res.data
}

/**
 * Delete a single user (hard delete); requires an `admin.user.delete` proof
 */
export async function deleteUser(
  id: number,
  proofToken: string
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/user/${id}/`,
    securityProofConfig(proofToken)
  )
  return res.data
}

/**
 * Manage user (promote, demote, enable, disable, delete); requires an
 * `admin.user.manage` proof (`admin.user.delete` for deletion)
 */
export async function manageUser(
  id: number,
  action: ManageUserAction,
  proofToken: string
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post(
    '/api/user/manage',
    { id, action },
    securityProofConfig(proofToken)
  )
  return res.data
}

/**
 * Adjust user quota atomically (add/subtract/override)
 */
export async function adjustUserQuota(
  payload: ManageUserQuotaPayload
): Promise<ApiResponse<Partial<User>>> {
  const res = await api.post('/api/user/manage', payload)
  return res.data
}

export async function listUserTags(): Promise<ApiResponse<UserTag[]>> {
  const res = await api.get('/api/user/tags')
  return res.data
}

export async function createUserTag(
  name: string,
  color: string
): Promise<ApiResponse<UserTag>> {
  const res = await api.post('/api/user/tags', { name, color })
  return res.data
}

export async function updateUserTag(
  tagId: number,
  name: string,
  color: string
): Promise<ApiResponse<UserTag>> {
  const res = await api.put(`/api/user/tags/${tagId}`, { name, color })
  return res.data
}

export async function deleteUserTag(tagId: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/tags/${tagId}`)
  return res.data
}

export async function assignUserTag(
  userId: number,
  tagId: number
): Promise<ApiResponse> {
  const res = await api.put(`/api/user/${userId}/tag`, { tag_id: tagId })
  return res.data
}

export async function searchUserManagementOptions(
  keyword: string,
  page: number,
  pageSize: number
): Promise<ApiResponse<PageData<UserManagementOption>>> {
  const res = await api.get('/api/user/options', {
    params: { keyword, p: page, page_size: pageSize },
  })
  return res.data
}

export async function resolveUserManagementOptions(
  userIds: number[]
): Promise<ApiResponse<UserManagementOption[]>> {
  const res = await api.post('/api/user/options/resolve', {
    user_ids: userIds,
  })
  return res.data
}

export async function batchAdjustUserQuota(
  payload: BatchQuotaAdjustPayload
): Promise<ApiResponse<BatchQuotaAdjustResult>> {
  const res = await api.post('/api/user/batch-quota', payload)
  return res.data
}

/**
 * Reset user's Passkey registration; requires an `admin.user.passkey.reset` proof
 */
export async function resetUserPasskey(
  id: number,
  proofToken: string
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/user/${id}/reset_passkey`,
    securityProofConfig(proofToken)
  )
  return res.data
}

/**
 * Reset user's Two-Factor Authentication setup; requires an
 * `admin.user.2fa.disable` proof
 */
export async function resetUserTwoFA(
  id: number,
  proofToken: string
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/user/${id}/2fa`,
    securityProofConfig(proofToken)
  )
  return res.data
}

/**
 * Get all available groups
 */
export async function getGroups(): Promise<ApiResponse<string[]>> {
  const res = await api.get('/api/group/')
  return res.data
}

/**
 * Get the permission catalog (resources, actions, and role baselines).
 * Source of truth lives in the backend authz package.
 */
export async function getPermissionCatalog(): Promise<PermissionCatalog> {
  const res = await api.get('/api/authz/catalog')
  requireServerSuccess(res.data)
  return {
    resources: res.data?.data?.resources ?? [],
    roles: res.data?.data?.roles ?? [],
  }
}

export async function getAgentDetail(
  userId: number
): Promise<ApiResponse<AgentDetailData>> {
  const res = await api.get(`/api/user/agent-detail/${userId}`)
  return res.data
}

export async function getUserUsageSummary(
  userId: number,
  userGroup?: string
): Promise<ApiResponse<UserUsageSummary>> {
  const res = await api.get(`/api/user/${userId}/usage-summary`, {
    params: userGroup ? { user_group: userGroup } : undefined,
  })
  return res.data
}

export async function getUserRiskReport(
  userId: number,
  days: 1 | 7 | 30
): Promise<ApiResponse<UserRiskReport>> {
  const res = await api.get(`/api/user/${userId}/risk`, {
    params: { days },
  })
  return res.data
}

export async function updateUserRiskDetection(
  userId: number,
  enabled: boolean
): Promise<
  ApiResponse<{
    enabled: boolean
    global_enabled: boolean
    user_enabled: boolean
  }>
> {
  const res = await api.put(`/api/user/${userId}/risk`, { enabled })
  return res.data
}

export async function getUserQuotaIncreaseLogs(
  userId: number,
  page: number,
  pageSize: number
): Promise<ApiResponse<UserQuotaIncreaseLogPage>> {
  const res = await api.get(`/api/user/${userId}/quota-increases`, {
    params: { p: page, page_size: pageSize },
  })
  return res.data
}

export async function getUserLoginIPs(
  userId: number
): Promise<ApiResponse<UserLoginIP[]>> {
  const res = await api.get(`/api/user/${userId}/login-ips`)
  return res.data
}

export async function updateUserLoginIPs(
  userId: number,
  ips: string[],
  blocked: boolean
): Promise<ApiResponse> {
  const res = await api.put(`/api/user/${userId}/login-ips`, {
    ips,
    blocked,
  })
  return res.data
}

export async function getUserLoginDevices(
  userId: number
): Promise<ApiResponse<UserLoginDevice[]>> {
  const res = await api.get(`/api/user/${userId}/login-devices`)
  return res.data
}

export async function updateUserLoginDevices(
  userId: number,
  deviceIds: string[],
  blocked: boolean
): Promise<ApiResponse<{ revoked_count: number }>> {
  const res = await api.put(`/api/user/${userId}/login-devices`, {
    device_ids: deviceIds,
    blocked,
  })
  return res.data
}

export async function getUserRequestContentLogs(
  userId: number
): Promise<ApiResponse<UserRequestContentLogList>> {
  const res = await api.get(`/api/user/${userId}/request-content`)
  return res.data
}

export async function updateUserRequestContentLogging(
  userId: number,
  enabled: boolean
): Promise<ApiResponse<{ enabled: boolean }>> {
  const res = await api.put(`/api/user/${userId}/request-content`, {
    enabled,
  })
  return res.data
}

export async function getUserRequestContentLog(
  userId: number,
  logId: number
): Promise<ApiResponse<UserRequestContentLogDetail>> {
  const res = await api.get(`/api/user/${userId}/request-content/${logId}`)
  return res.data
}

export async function deleteUserRequestContentLogs(
  userId: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/${userId}/request-content`)
  return res.data
}

// ============================================================================
// Admin Binding Management APIs
// ============================================================================

/**
 * Get user's custom OAuth bindings (admin)
 */
export async function getUserOAuthBindings(
  userId: number
): Promise<ApiResponse<CustomOAuthBinding[]>> {
  const res = await api.get(`/api/user/${userId}/oauth/bindings`)
  return res.data
}

/**
 * Clear a user's built-in binding (admin); requires an
 * `admin.user.binding.clear` proof bound to the binding type
 */
export async function adminClearUserBinding(
  userId: number,
  bindingType: string,
  proofToken: string
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/user/${userId}/bindings/${bindingType}`,
    securityProofConfig(proofToken)
  )
  return res.data
}

/**
 * Unbind custom OAuth for a user (admin); requires an
 * `admin.user.binding.clear` proof bound to the provider ID
 */
export async function adminUnbindCustomOAuth(
  userId: number,
  providerId: number,
  proofToken: string
): Promise<ApiResponse> {
  const res = await api.delete(
    `/api/user/${userId}/oauth/bindings/${providerId}`,
    securityProofConfig(proofToken)
  )
  return res.data
}
