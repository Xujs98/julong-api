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
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { deleteInvalidRedemptions, updateAgentTopUpLink } from '../api'
import { ERROR_MESSAGES } from '../constants'
import { useRedemptions } from './redemptions-provider'

export function RedemptionsPrimaryButtons() {
  const { t } = useTranslation()
  const currentUser = useAuthStore((s) => s.auth.user)
  const setUser = useAuthStore((s) => s.auth.setUser)
  const isAdmin = (currentUser?.role ?? 0) >= ROLE.ADMIN
  const isAgent = currentUser?.is_agent === true
  const { setOpen, triggerRefresh } = useRedemptions()
  const [showDeleteInvalidConfirm, setShowDeleteInvalidConfirm] =
    useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [agentTopUpLink, setAgentTopUpLink] = useState(
    currentUser?.agent_topup_link || ''
  )
  const [isSavingLink, setIsSavingLink] = useState(false)

  useEffect(() => {
    setAgentTopUpLink(currentUser?.agent_topup_link || '')
  }, [currentUser?.agent_topup_link])

  const handleDeleteInvalid = async () => {
    setIsDeleting(true)
    try {
      const result = await deleteInvalidRedemptions()
      if (result.success) {
        const count = result.data || 0
        toast.success(
          t('Successfully deleted {{count}} invalid redemption codes', {
            count,
          })
        )
        triggerRefresh()
        setShowDeleteInvalidConfirm(false)
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.DELETE_INVALID_FAILED))
      }
    } finally {
      setIsDeleting(false)
    }
  }

  const handleSaveAgentLink = async () => {
    setIsSavingLink(true)
    try {
      const result = await updateAgentTopUpLink(agentTopUpLink)
      if (result.success) {
        if (currentUser) {
          setUser({
            ...currentUser,
            agent_topup_link: result.data?.agent_topup_link || agentTopUpLink,
          })
        }
        toast.success(t('Saved successfully'))
      } else {
        toast.error(result.message || t('Update failed'))
      }
    } finally {
      setIsSavingLink(false)
    }
  }

  return (
    <>
      <div className='flex flex-wrap items-center gap-2'>
        {isAgent && (
          <>
            <Input
              className='h-8 w-[280px]'
              value={agentTopUpLink}
              onChange={(event) => setAgentTopUpLink(event.target.value)}
              placeholder={t('Agent top-up link')}
            />
            <Button
              size='sm'
              variant='outline'
              onClick={handleSaveAgentLink}
              disabled={isSavingLink}
            >
              {isSavingLink ? t('Saving...') : t('Save changes')}
            </Button>
          </>
        )}
        {isAdmin && (
          <Button
            size='sm'
            variant='outline'
            onClick={() => setShowDeleteInvalidConfirm(true)}
          >
            <Trash2 className='text-destructive h-4 w-4' />
            {t('Delete Invalid')}
          </Button>
        )}
        <Button size='sm' onClick={() => setOpen('create')}>
          <Plus className='h-4 w-4' />
          {t('Create Code')}
        </Button>
      </div>

      <ConfirmDialog
        destructive
        open={showDeleteInvalidConfirm}
        onOpenChange={setShowDeleteInvalidConfirm}
        handleConfirm={handleDeleteInvalid}
        isLoading={isDeleting}
        className='max-w-md'
        title={t('Delete Invalid Redemption Codes?')}
        desc={
          <>
            {t('This will delete all')} <strong>{t('used')}</strong>,{' '}
            <strong>{t('disabled')}</strong>
            {t(', and')} <strong>{t('expired')}</strong>{' '}
            {t('redemption codes.')}
            <br />
            {t('This action cannot be undone.')}
          </>
        }
        confirmText={t('Delete Invalid')}
      />
    </>
  )
}
