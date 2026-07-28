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
import type { Table } from '@tanstack/react-table'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'

import { batchDeleteRedemptions } from '../api'
import type { Redemption } from '../types'
import { useRedemptions } from './redemptions-provider'

type RedemptionsMultiDeleteDialogProps<TData> = {
  table: Table<TData>
  onOpenChange: (open: boolean) => void
}

export function RedemptionsMultiDeleteDialog<TData>({
  table,
  onOpenChange,
}: RedemptionsMultiDeleteDialogProps<TData>) {
  const { t } = useTranslation()
  const { triggerRefresh } = useRedemptions()
  const [secondsLeft, setSecondsLeft] = useState(5)
  const [isDeleting, setIsDeleting] = useState(false)
  const selectedRows = table.getFilteredSelectedRowModel().rows

  useEffect(() => {
    if (secondsLeft <= 0) return

    const timer = window.setTimeout(() => {
      setSecondsLeft((seconds) => seconds - 1)
    }, 1000)
    return () => window.clearTimeout(timer)
  }, [secondsLeft])

  const handleDelete = async () => {
    const ids = selectedRows.map((row) => (row.original as Redemption).id)
    if (ids.length === 0 || secondsLeft > 0) return

    setIsDeleting(true)
    try {
      const result = await batchDeleteRedemptions(ids)
      const deletedCount = result.success ? result.data || 0 : 0
      const failedCount = ids.length - deletedCount

      if (deletedCount > 0) {
        toast.success(
          t('Successfully deleted {{count}} redemption codes', {
            count: deletedCount,
          })
        )
      }
      if (failedCount > 0) {
        toast.error(
          t('Failed to delete {{count}} redemption codes', {
            count: failedCount,
          })
        )
      }

      table.resetRowSelection()
      triggerRefresh()
      onOpenChange(false)
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <ConfirmDialog
      destructive
      open
      onOpenChange={onOpenChange}
      handleConfirm={handleDelete}
      isLoading={isDeleting}
      disabled={secondsLeft > 0}
      className='max-w-md'
      title={t('Are you sure?')}
      desc={
        <>
          {t('You are about to delete {{count}} redemption codes.', {
            count: selectedRows.length,
          })}{' '}
          {t('This action cannot be undone.')}
        </>
      }
      confirmText={
        secondsLeft > 0
          ? t('Delete in {{seconds}}s', { seconds: secondsLeft })
          : t('Delete')
      }
    />
  )
}
