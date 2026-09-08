/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import {
  Download,
  Loader2,
  Mail,
  Paperclip,
  RotateCcw,
  Upload,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { formatTimestamp } from '@/lib/format'

import { getInvoiceSourceLabel, getInvoiceStatusLabel } from '../labels'
import type { InvoiceApplication, InvoiceStatus } from '../types'
import { CompleteInvoiceDialog } from './complete-invoice-dialog'
import { WithdrawInvoiceDialog } from './withdraw-invoice-dialog'

const statuses: InvoiceStatus[] = [
  'pending',
  'processing',
  'completed',
  'rejected',
  'non_reusable',
  'cancelled',
]

function isTerminalStatus(status: InvoiceStatus) {
  return ['completed', 'rejected', 'non_reusable', 'cancelled'].includes(status)
}

export function ManageInvoiceDialog(props: {
  application: InvoiceApplication | null
  saving: boolean
  withdrawing?: boolean
  onOpenChange: (open: boolean) => void
  onSave: (status: InvoiceStatus, adminNote: string) => void
  onComplete: (locale: 'zh' | 'en', adminNote: string, attachment: File) => void
  onDownload?: (application: InvoiceApplication) => void
  onWithdraw?: (application: InvoiceApplication) => void
}) {
  const { t, i18n } = useTranslation()
  const [status, setStatus] = useState<InvoiceStatus>('pending')
  const [adminNote, setAdminNote] = useState('')
  const [attachment, setAttachment] = useState<File | null>(null)
  const [sendEmail, setSendEmail] = useState(true)
  const [emailLocale, setEmailLocale] = useState<'zh' | 'en'>('zh')
  const [emailSettingsOpen, setEmailSettingsOpen] = useState(false)
  const [withdrawOpen, setWithdrawOpen] = useState(false)

  useEffect(() => {
    if (!props.application) return
    setStatus(props.application.status)
    setAdminNote(props.application.admin_note || '')
    setAttachment(null)
    setSendEmail(true)
    setEmailLocale(i18n.language.startsWith('en') ? 'en' : 'zh')
    setEmailSettingsOpen(false)
    setWithdrawOpen(false)
  }, [i18n.language, props.application])

  if (!props.application) return null

  const application = props.application
  const items = application.items ?? []
  const terminal = isTerminalStatus(application.status)
  const completing = !terminal && status === 'completed'
  const attachmentValid =
    attachment !== null && attachment.size <= 10 * 1024 * 1024

  return (
    <>
      <Dialog open onOpenChange={props.onOpenChange}>
        <DialogContent className='max-h-[92vh] overflow-y-auto sm:max-w-4xl'>
          <DialogHeader>
            <DialogTitle>{t('Invoice details')}</DialogTitle>
            <DialogDescription>
              {t('Review billing details and update the application status.')}
            </DialogDescription>
          </DialogHeader>

          <div className='grid gap-3 rounded-lg border p-4 text-sm sm:grid-cols-2 lg:grid-cols-3'>
            <div>
              <span className='text-muted-foreground'>{t('Application')}</span>
              <p className='font-medium'>#{application.id}</p>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('User')}</span>
              <p className='font-medium'>
                {application.username || `#${application.user_id}`}
              </p>
            </div>
            <div>
              <span className='text-muted-foreground'>
                {t('Submitted email')}
              </span>
              <p className='font-medium break-all'>{application.email}</p>
            </div>
            <div>
              <span className='text-muted-foreground'>
                {t('Invoice amount')}
              </span>
              <p className='font-medium'>
                {application.unit} {application.amount.toFixed(2)}
              </p>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Applied at')}</span>
              <p className='font-medium'>
                {formatTimestamp(application.created_at)}
              </p>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Completed at')}</span>
              <p className='font-medium'>
                {application.completed_at
                  ? formatTimestamp(application.completed_at)
                  : '-'}
              </p>
            </div>
          </div>

          <section className='space-y-3'>
            <h3 className='font-medium'>{t('Billing details')}</h3>
            {items.length === 0 ? (
              <div className='text-muted-foreground rounded-lg border px-4 py-8 text-center text-sm'>
                {t('No billing details')}
              </div>
            ) : (
              <div className='grid gap-3'>
                {items.map((item) => (
                  <article
                    key={item.id}
                    className='min-w-0 rounded-lg border p-4'
                  >
                    <div className='flex flex-wrap items-start justify-between gap-3'>
                      <div className='min-w-0'>
                        <p className='font-medium'>
                          {getInvoiceSourceLabel(t, item.source)}
                        </p>
                        <p className='text-muted-foreground mt-1 text-xs'>
                          {formatTimestamp(item.source_created_at)}
                        </p>
                      </div>
                      <p className='shrink-0 font-semibold'>
                        {application.unit} {item.amount.toFixed(2)}
                      </p>
                    </div>
                    <div className='mt-4 grid min-w-0 gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]'>
                      <div className='bg-muted/40 min-w-0 rounded-md p-3'>
                        <p className='text-muted-foreground text-xs'>
                          {t('Source request ID')}
                        </p>
                        <CopyButton
                          value={item.source_request_id}
                          variant='outline'
                          size='sm'
                          className='mt-2 max-w-full'
                          tooltip={t('Copy source request ID')}
                          successTooltip={t('Source request ID copied')}
                          aria-label={t('Copy source request ID')}
                        >
                          <span className='truncate'>
                            {t('Copy request ID')}
                          </span>
                        </CopyButton>
                      </div>
                      <div className='bg-muted/40 min-w-0 rounded-md p-3'>
                        <p className='text-muted-foreground text-xs'>
                          {t('Source details')}
                        </p>
                        <p className='mt-2 max-w-full text-sm leading-6 [overflow-wrap:anywhere] whitespace-pre-wrap'>
                          {item.content || t('No source details')}
                        </p>
                      </div>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>

          <div className='grid gap-4 sm:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='invoice-status'>{t('Status')}</Label>
              <NativeSelect
                id='invoice-status'
                className='w-full'
                value={status}
                disabled={terminal || props.saving}
                onChange={(event) => {
                  setStatus(event.target.value as InvoiceStatus)
                  setAttachment(null)
                }}
              >
                {statuses.map((value) => (
                  <NativeSelectOption
                    key={value}
                    value={value}
                    disabled={value === 'cancelled'}
                  >
                    {getInvoiceStatusLabel(t, value)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='invoice-admin-note'>{t('Internal note')}</Label>
              <Textarea
                id='invoice-admin-note'
                maxLength={500}
                value={adminNote}
                disabled={terminal || props.saving}
                onChange={(event) => setAdminNote(event.target.value)}
                placeholder={t('Optional processing note')}
              />
            </div>
          </div>

          {completing && (
            <section className='space-y-4 rounded-lg border p-4'>
              <div className='flex flex-wrap items-center justify-between gap-3'>
                <div>
                  <h3 className='font-medium'>{t('Invoice delivery')}</h3>
                  <p className='text-muted-foreground mt-1 text-sm'>
                    {t(
                      'Upload the invoice and confirm its email delivery settings before saving.'
                    )}
                  </p>
                </div>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => setEmailSettingsOpen(true)}
                  disabled={!sendEmail}
                >
                  <Mail />
                  {t('Email settings')}
                </Button>
              </div>
              <div className='space-y-2'>
                <Label htmlFor='invoice-attachment'>
                  {t('Electronic invoice attachment')}
                </Label>
                <div className='flex flex-wrap items-center gap-3'>
                  <Button
                    variant='outline'
                    size='sm'
                    render={<label htmlFor='invoice-attachment' />}
                  >
                    <Upload />
                    {t('Upload invoice attachment')}
                  </Button>
                  <Input
                    id='invoice-attachment'
                    className='sr-only'
                    type='file'
                    accept='.pdf,.ofd,.xml,.png,.jpg,.jpeg'
                    onChange={(event) =>
                      setAttachment(event.target.files?.item(0) ?? null)
                    }
                  />
                  <span className='text-muted-foreground min-w-0 truncate text-sm'>
                    {attachment?.name || t('No file selected')}
                  </span>
                </div>
                <p className='text-muted-foreground flex items-center gap-1 text-xs'>
                  <Paperclip className='size-3' />
                  {t('PDF, OFD, XML, PNG, or JPG; up to 10 MB.')}
                </p>
              </div>
              <label className='flex cursor-pointer items-center gap-2 text-sm'>
                <Checkbox
                  checked={sendEmail}
                  onCheckedChange={(checked) => setSendEmail(checked === true)}
                />
                <span>{t('Send invoice by email')}</span>
                <span className='text-muted-foreground'>
                  · {emailLocale === 'zh' ? t('Chinese') : t('English')}
                </span>
              </label>
            </section>
          )}

          <DialogFooter>
            {application.status === 'completed' && (
              <Button
                variant='ghost'
                className='text-destructive hover:text-destructive'
                onClick={() => setWithdrawOpen(true)}
                disabled={props.saving || props.withdrawing}
              >
                <RotateCcw />
                {t('Withdraw')}
              </Button>
            )}
            {application.status === 'completed' &&
              application.has_attachment && (
                <Button
                  variant='outline'
                  onClick={() => props.onDownload?.(application)}
                >
                  <Download />
                  {t('Download invoice')}
                </Button>
              )}
            <Button
              variant='outline'
              onClick={() => props.onOpenChange(false)}
              disabled={props.saving}
            >
              {t('Close')}
            </Button>
            {!terminal && (
              <Button
                onClick={() => {
                  if (status === 'completed' && attachment) {
                    props.onComplete(emailLocale, adminNote.trim(), attachment)
                    return
                  }
                  props.onSave(status, adminNote.trim())
                }}
                disabled={
                  props.saving ||
                  (completing && (!attachmentValid || !sendEmail))
                }
              >
                {props.saving && <Loader2 className='animate-spin' />}
                {t('Save Changes')}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {emailSettingsOpen && (
        <CompleteInvoiceDialog
          application={application}
          open
          locale={emailLocale}
          onOpenChange={setEmailSettingsOpen}
          onSave={(locale) => {
            setEmailLocale(locale)
            setEmailSettingsOpen(false)
          }}
        />
      )}
      <WithdrawInvoiceDialog
        open={withdrawOpen}
        withdrawing={props.withdrawing ?? false}
        onOpenChange={setWithdrawOpen}
        onConfirm={() => props.onWithdraw?.(application)}
      />
    </>
  )
}
