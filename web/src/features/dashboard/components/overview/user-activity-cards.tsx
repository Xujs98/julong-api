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
import { useQuery } from '@tanstack/react-query'
import { Activity, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { requireServerSuccess } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import { getUserActivitySummary } from '../../api'
import { StatCard } from '../ui/stat-card'

export function UserActivityCards() {
  const { t, i18n } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const allowed = Boolean(user && user.role >= ROLE.ADMIN)
  const query = useQuery({
    queryKey: ['dashboard', 'user-activity', user?.id],
    enabled: allowed,
    queryFn: async () =>
      requireServerSuccess(await getUserActivitySummary()).data,
    refetchInterval: 30 * 1000,
    refetchIntervalInBackground: false,
  })
  if (!allowed) return null
  return (
    <div className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
      <StatCard
        title={t('Daily active users')}
        value={
          query.data ? formatNumber(query.data.daily_active_users, locale) : '-'
        }
        description={t('Unique active accounts today (UTC+8), excluding Root')}
        icon={Users}
        loading={query.isLoading}
        error={query.isError}
      />
      <StatCard
        title={t('Online users')}
        value={query.data ? formatNumber(query.data.online_users, locale) : '-'}
        description={t('Active in the last 5 minutes, excluding Root')}
        icon={Activity}
        tone='accent-2'
        loading={query.isLoading}
        error={query.isError}
      />
    </div>
  )
}
