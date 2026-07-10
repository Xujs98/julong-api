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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { api } from '@/lib/api'
import { formatQuota } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

type AgentUser = {
  id: number
  username: string
  display_name?: string
  quota: number
  used_quota: number
  request_count: number
  status: number
  subscriptions?: Array<{
    subscription?: {
      amount_total: number
      amount_used: number
      end_time: number
      status: string
    }
  }>
}

type AgentUsersResponse = {
  success: boolean
  data?: {
    items: AgentUser[]
  }
}

async function getAgentUsers() {
  const res = await api.get<AgentUsersResponse>(
    '/api/user/agent/users?p=1&page_size=100'
  )
  return res.data.data?.items ?? []
}

function AgentUsersPage() {
  const { t } = useTranslation()
  const { data: users = [] } = useQuery({
    queryKey: ['agent-users'],
    queryFn: getAgentUsers,
  })

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Agent')} {t('Users')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='overflow-hidden rounded-md border'>
          <table className='w-full text-sm'>
            <thead className='bg-muted/50'>
              <tr>
                <th className='px-3 py-2 text-left'>{t('ID')}</th>
                <th className='px-3 py-2 text-left'>{t('Username')}</th>
                <th className='px-3 py-2 text-left'>{t('Quota')}</th>
                <th className='px-3 py-2 text-left'>{t('Used:')}</th>
                <th className='px-3 py-2 text-left'>{t('Requests:')}</th>
                <th className='px-3 py-2 text-left'>{t('Subscriptions')}</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 ? (
                <tr>
                  <td
                    colSpan={6}
                    className='text-muted-foreground px-3 py-6 text-center'
                  >
                    {t('No data')}
                  </td>
                </tr>
              ) : (
                users.map((user) => {
                  const activeSubscriptions =
                    user.subscriptions?.filter(
                      (item) => item.subscription?.status === 'active'
                    ) ?? []
                  return (
                    <tr key={user.id} className='border-t'>
                      <td className='px-3 py-2 tabular-nums'>{user.id}</td>
                      <td className='px-3 py-2'>
                        <div className='font-medium'>{user.username}</div>
                        {user.display_name && user.display_name !== user.username && (
                          <div className='text-muted-foreground text-xs'>
                            {user.display_name}
                          </div>
                        )}
                      </td>
                      <td className='px-3 py-2 tabular-nums'>
                        {formatQuota(user.quota)}
                      </td>
                      <td className='px-3 py-2 tabular-nums'>
                        {formatQuota(user.used_quota)}
                      </td>
                      <td className='px-3 py-2 tabular-nums'>
                        {user.request_count.toLocaleString()}
                      </td>
                      <td className='px-3 py-2'>
                        <StatusBadge
                          label={String(activeSubscriptions.length)}
                          variant={
                            activeSubscriptions.length > 0
                              ? 'success'
                              : 'neutral'
                          }
                          copyable={false}
                        />
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

export const Route = createFileRoute('/_authenticated/agent-users/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!auth.user?.is_agent) {
      throw redirect({ to: '/403' })
    }
  },
  component: AgentUsersPage,
})
