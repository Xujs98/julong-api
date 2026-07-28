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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ListFilter, Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { EmailCampaignUserPicker } from '@/features/system-settings/operations/email-campaign-user-picker'

import {
  getUserQuotaSummarySettings,
  resolveUserQuotaSummaryOptions,
  searchUserQuotaSummaryOptions,
  updateUserQuotaSummarySettings,
} from '../api'
import type { UserQuotaSummary } from '../types'
import { UserQuotaSummaryBadge } from './user-quota-summary-badge'

type UserQuotaSummaryControlProps = {
  summary?: UserQuotaSummary
  isFetching: boolean
}

export function UserQuotaSummaryControl(props: UserQuotaSummaryControlProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [draftUserIds, setDraftUserIds] = useState<number[]>([])
  const [saving, setSaving] = useState(false)
  const settingsQuery = useQuery({
    queryKey: ['users', 'quota-summary-settings'],
    queryFn: async () => {
      const response = await getUserQuotaSummarySettings()
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load users'))
      }
      return response.data
    },
  })
  const excludedUserIds = settingsQuery.data?.excluded_user_ids ?? []

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) setDraftUserIds(excludedUserIds)
    setOpen(nextOpen)
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const response = await updateUserQuotaSummarySettings(draftUserIds)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to save'))
      }
      queryClient.setQueryData(
        ['users', 'quota-summary-settings'],
        response.data
      )
      await queryClient.invalidateQueries({
        queryKey: ['users', 'quota-summary'],
      })
      toast.success(t('Saved successfully'))
      setOpen(false)
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Failed to save'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className='flex w-full flex-wrap items-center gap-2 sm:w-auto'>
      <Dialog
        open={open}
        onOpenChange={handleOpenChange}
        title={t('Quota statistics')}
        contentClassName='sm:max-w-xl'
        trigger={
          <Button
            type='button'
            variant='outline'
            size='sm'
            className='h-9 gap-2'
            disabled={settingsQuery.isLoading}
          >
            {settingsQuery.isLoading ? (
              <Loader2 className='size-4 animate-spin' />
            ) : (
              <ListFilter className='size-4' />
            )}
            {t('Excluded users')}
            {excludedUserIds.length > 0 && (
              <Badge variant='secondary' className='rounded-sm px-1.5'>
                {excludedUserIds.length}
              </Badge>
            )}
          </Button>
        }
        footer={
          <>
            <Button variant='outline' onClick={() => setOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className='size-4 animate-spin' />}
              {t('Save')}
            </Button>
          </>
        }
      >
        <div className='flex flex-col gap-2'>
          <Label htmlFor='quota-summary-excluded-users'>
            {t('Excluded users')}
          </Label>
          <EmailCampaignUserPicker
            id='quota-summary-excluded-users'
            labelKey='Excluded users'
            queryKeyPrefix='quota-summary'
            selectedUserIds={draftUserIds}
            onChange={setDraftUserIds}
            searchUsers={searchUserQuotaSummaryOptions}
            resolveUsers={resolveUserQuotaSummaryOptions}
          />
        </div>
      </Dialog>
      <UserQuotaSummaryBadge
        summary={props.summary}
        isFetching={props.isFetching}
      />
    </div>
  )
}
