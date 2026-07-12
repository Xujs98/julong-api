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
import { createFileRoute, redirect } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { ContentSettings } from '@/features/system-settings/content'
import {
  CONTENT_DEFAULT_SECTION,
  CONTENT_SECTION_IDS,
} from '@/features/system-settings/content/section-registry.tsx'
import { SupportContactsSection } from '@/features/system-settings/content/support-contacts-section'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute(
  '/_authenticated/system-settings/content/$section'
)({
  beforeLoad: ({ params }) => {
    const user = useAuthStore.getState().auth.user
    const isRoot = user?.role === ROLE.SUPER_ADMIN
    const canManageSupport = hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.SYSTEM_SETTINGS,
      ADMIN_PERMISSION_ACTIONS.SUPPORT_CONTACTS_WRITE
    )
    if (!isRoot && (!canManageSupport || params.section !== 'support')) {
      throw redirect({ to: '/403' })
    }
    const validSections = CONTENT_SECTION_IDS as unknown as string[]
    if (!validSections.includes(params.section)) {
      throw redirect({
        to: '/system-settings/content/$section',
        params: { section: CONTENT_DEFAULT_SECTION },
      })
    }
  },
  component: ContentSettingsRoute,
})

function ContentSettingsRoute() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  if (user?.role !== ROLE.SUPER_ADMIN) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Customer Support')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <SupportContactsSection />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }
  return <ContentSettings />
}
