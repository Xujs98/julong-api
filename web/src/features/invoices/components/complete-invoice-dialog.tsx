/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { getInvoiceEmailTemplates } from '../api'
import type { InvoiceApplication } from '../types'

export function CompleteInvoiceDialog(props: {
  application: InvoiceApplication
  open: boolean
  locale: 'zh' | 'en'
  onOpenChange: (open: boolean) => void
  onSave: (locale: 'zh' | 'en') => void
}) {
  const { t } = useTranslation()
  const [locale, setLocale] = useState<'zh' | 'en'>(props.locale)
  const templatesQuery = useQuery({
    queryKey: ['invoice', 'email-templates'],
    enabled: props.open,
    queryFn: getInvoiceEmailTemplates,
  })

  useEffect(() => {
    if (props.open) setLocale(props.locale)
  }, [props.locale, props.open])

  const templates = templatesQuery.data?.data ?? []
  const selectedTemplate = templates.find(
    (template) => template.locale === locale
  )

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>{t('Email settings')}</DialogTitle>
          <DialogDescription>
            {t('Select the email template used to deliver this invoice.')}
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-4'>
          <div className='rounded-lg border px-4 py-3 text-sm'>
            <div className='font-medium break-all'>
              {props.application.email}
            </div>
            <div className='text-muted-foreground mt-1'>
              {props.application.unit} {props.application.amount.toFixed(2)} · #
              {props.application.id}
            </div>
          </div>
          <div className='space-y-2'>
            <Label htmlFor='invoice-email-template'>
              {t('Email template')}
            </Label>
            <NativeSelect
              id='invoice-email-template'
              className='w-full'
              value={locale}
              onChange={(event) => setLocale(event.target.value as 'zh' | 'en')}
              disabled={templatesQuery.isPending}
            >
              {templates.map((template) => (
                <NativeSelectOption
                  key={template.locale}
                  value={template.locale}
                >
                  {template.locale === 'zh' ? t('Chinese') : t('English')} ·{' '}
                  {t(template.label)}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            {selectedTemplate && (
              <div className='bg-muted/50 rounded-md border px-3 py-2'>
                <p className='text-muted-foreground text-xs'>
                  {t('Email subject')}
                </p>
                <p className='mt-1 text-sm break-words'>
                  {selectedTemplate.subject}
                </p>
              </div>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button
            disabled={!selectedTemplate}
            onClick={() => props.onSave(locale)}
          >
            {t('Save email settings')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
