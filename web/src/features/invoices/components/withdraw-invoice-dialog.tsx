/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

export function WithdrawInvoiceDialog(props: {
  open: boolean
  withdrawing: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const { t } = useTranslation()

  return (
    <AlertDialog open={props.open} onOpenChange={props.onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t('Withdraw completed invoice?')}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              'The application will return to pending confirmation and the uploaded invoice attachment will be permanently deleted. The previously sent email is not recalled.'
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={props.withdrawing}>
            {t('Keep completed')}
          </AlertDialogCancel>
          <AlertDialogAction
            variant='destructive'
            disabled={props.withdrawing}
            onClick={props.onConfirm}
          >
            {props.withdrawing && <Loader2 className='animate-spin' />}
            {t('Confirm withdrawal')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
