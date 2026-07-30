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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'

import {
  canConfirmLedgerDelete,
  LEDGER_DELETE_COUNTDOWN_SECONDS,
  nextLedgerDeleteCountdown,
} from '../lib/delete-confirmation'

interface LedgerDeleteDialogProps {
  open: boolean
  count: number
  deleting: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => Promise<void>
}

export function LedgerDeleteDialog(props: LedgerDeleteDialogProps) {
  const { t } = useTranslation()
  const [secondsLeft, setSecondsLeft] = useState(
    LEDGER_DELETE_COUNTDOWN_SECONDS
  )

  useEffect(() => {
    if (!props.open) return
    setSecondsLeft(LEDGER_DELETE_COUNTDOWN_SECONDS)
  }, [props.open])

  useEffect(() => {
    if (!props.open || secondsLeft <= 0) return
    const timer = window.setTimeout(() => {
      setSecondsLeft(nextLedgerDeleteCountdown)
    }, 1000)
    return () => window.clearTimeout(timer)
  }, [props.open, secondsLeft])

  return (
    <ConfirmDialog
      destructive
      open={props.open}
      onOpenChange={props.onOpenChange}
      handleConfirm={() => void props.onConfirm()}
      isLoading={props.deleting}
      disabled={!canConfirmLedgerDelete(secondsLeft, props.count)}
      className='max-w-md'
      title={t('Delete ledger entries?')}
      desc={
        <>
          {t('You are about to delete {{count}} ledger entries.', {
            count: props.count,
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
