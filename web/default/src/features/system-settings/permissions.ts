import {
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import type { AuthUser } from '@/stores/auth-store'

export function systemSettingsAction(category: string, section: string) {
  return `${category}.${section}`
}

export function canAccessSystemSettingsSection(
  user: AuthUser | null | undefined,
  category: string,
  section: string
) {
  if (user?.role === ROLE.SUPER_ADMIN) return true
  return hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.SYSTEM_SETTINGS,
    systemSettingsAction(category, section)
  )
}

export function hasAnySystemSettingsPermission(
  user: AuthUser | null | undefined
) {
  if (user?.role === ROLE.SUPER_ADMIN) return true
  const permissions =
    user?.permissions?.admin_permissions?.[
      ADMIN_PERMISSION_RESOURCES.SYSTEM_SETTINGS
    ]
  return Object.values(permissions || {}).some(Boolean)
}

export function firstAllowedSystemSettingsPath(
  user: AuthUser | null | undefined
) {
  if (user?.role === ROLE.SUPER_ADMIN) return '/system-settings/site'
  const permissions =
    user?.permissions?.admin_permissions?.[
      ADMIN_PERMISSION_RESOURCES.SYSTEM_SETTINGS
    ] || {}
  const action = Object.entries(permissions).find(([, allowed]) => allowed)?.[0]
  if (!action) return null
  const [category, section] = action.split('.')
  return category && section ? `/system-settings/${category}/${section}` : null
}
