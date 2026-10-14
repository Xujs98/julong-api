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
import type { TFunction } from 'i18next'
import {
  Box,
  CreditCard,
  Layout,
  Settings,
  Shield,
  ShieldAlert,
  Wrench,
} from 'lucide-react'

import { getAuthSectionNavItems } from '@/features/system-settings/auth/section-registry.tsx'
import { getBillingSectionNavItems } from '@/features/system-settings/billing/section-registry.tsx'
import { getContentSectionNavItems } from '@/features/system-settings/content/section-registry.tsx'
import { getModelsSectionNavItems } from '@/features/system-settings/models/section-registry.tsx'
import { getOperationsSectionNavItems } from '@/features/system-settings/operations/section-registry.tsx'
import { canAccessSystemSettingsSection } from '@/features/system-settings/permissions'
import { getSecuritySectionNavItems } from '@/features/system-settings/security/section-registry.tsx'
import { getSiteSectionNavItems } from '@/features/system-settings/site/section-registry.tsx'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import type { NavGroup, SidebarView } from '../types'

/**
 * Sidebar nav groups for the System Settings nested view.
 *
 * Kept as a single group because the workspace title in the sidebar
 * header already provides top-level context — the inner group label
 * scopes the items as "administration" actions.
 */
function getSystemSettingsNavGroups(t: TFunction): NavGroup[] {
  const user = useAuthStore.getState().auth.user
  const filterItems = <T extends { url: string }>(
    category: string,
    items: T[]
  ) => {
    if (user?.role === ROLE.SUPER_ADMIN) return items
    return items.filter((item) => {
      const section = item.url.split('/').reverse().find(Boolean) || ''
      return canAccessSystemSettingsSection(user, category, section)
    })
  }
  const items = [
    {
      title: t('Site & Branding'),
      icon: Settings,
      items: filterItems('site', getSiteSectionNavItems(t)),
    },
    {
      title: t('Authentication'),
      icon: Shield,
      items: filterItems('auth', getAuthSectionNavItems(t)),
    },
    {
      title: t('Billing & Payment'),
      icon: CreditCard,
      items: filterItems('billing', getBillingSectionNavItems(t)),
    },
    {
      title: t('Models & Routing'),
      icon: Box,
      items: filterItems('models', getModelsSectionNavItems(t)),
    },
    {
      title: t('Security & Limits'),
      icon: ShieldAlert,
      items: filterItems('security', getSecuritySectionNavItems(t)),
    },
    {
      title: t('Console Content'),
      icon: Layout,
      items: filterItems('content', getContentSectionNavItems(t)),
    },
    {
      title: t('Operations'),
      icon: Wrench,
      items: filterItems('operations', getOperationsSectionNavItems(t)),
    },
  ].filter((item) => item.items.length > 0)
  return [
    {
      id: 'system-administration',
      title: t('System Administration'),
      items,
    },
  ]
}

/**
 * Nested sidebar view for `/system-settings/*`.
 *
 * Activates the Vercel / Cloudflare-style drill-in sidebar:
 * the root navigation is replaced by the system administration
 * groups, with a "Back to Dashboard" affordance in the header.
 */
export const SYSTEM_SETTINGS_VIEW: SidebarView = {
  id: 'system-settings',
  pathPattern: /^\/system-settings(\/|$)/,
  parent: {
    to: '/dashboard/overview',
    label: 'Back to Dashboard',
  },
  getNavGroups: getSystemSettingsNavGroups,
}
