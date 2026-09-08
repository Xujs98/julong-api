/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
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

import type { InvoiceApplication } from '../types'

export function CancelInvoiceDialog(props: {
  application: InvoiceApplication | null
  cancelling: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}) {
  const { t } = useTranslation()

  return (
    <AlertDialog
      open={props.application !== null}
      onOpenChange={props.onOpenChange}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('Cancel invoice application')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              'The application will remain in history, and its selected credits will become eligible immediately.'
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={props.cancelling}>
            {t('Keep application')}
          </AlertDialogCancel>
          <AlertDialogAction
            variant='destructive'
            disabled={props.cancelling}
            onClick={props.onConfirm}
          >
            {t('Cancel application')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
