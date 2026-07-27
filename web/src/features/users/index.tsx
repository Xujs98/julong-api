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
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'

import { listUserTags } from './api'
import { AgentDetailDialog } from './components/agent-detail-dialog'
import { UserBatchQuotaDialog } from './components/user-batch-quota-dialog'
import { UserDetailDialog } from './components/user-detail-dialog'
import { UserTagsDialog } from './components/user-tags-dialog'
import { UsersDeleteDialog } from './components/users-delete-dialog'
import { UsersMutateDrawer } from './components/users-mutate-drawer'
import { UsersPrimaryButtons } from './components/users-primary-buttons'
import { UsersProvider, useUsers } from './components/users-provider'
import { UsersTable } from './components/users-table'

function UsersContent() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow, triggerRefresh } = useUsers()
  const tagsQuery = useQuery({
    queryKey: ['user-tags'],
    queryFn: async () => {
      const response = await listUserTags()
      if (!response.success) {
        throw new Error(response.message || t('Failed to load tags'))
      }
      return response.data || []
    },
  })
  const tags = tagsQuery.data || []

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Users')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <UsersPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <UsersTable tags={tags} />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <UsersMutateDrawer
        open={open === 'create' || open === 'update'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        currentRow={open === 'update' ? currentRow || undefined : undefined}
      />
      <UsersDeleteDialog />
      <AgentDetailDialog />
      <UserDetailDialog />
      <UserTagsDialog
        open={open === 'tags'}
        tags={tags}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        onChanged={async () => {
          await tagsQuery.refetch()
          triggerRefresh()
        }}
      />
      <UserBatchQuotaDialog
        open={open === 'batch-quota'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        onSuccess={triggerRefresh}
      />
    </>
  )
}

export function Users() {
  return (
    <UsersProvider>
      <UsersContent />
    </UsersProvider>
  )
}
