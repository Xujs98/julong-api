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
import { Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { EmailCampaignUserPicker } from '@/features/system-settings/operations/email-campaign-user-picker'
import {
  listEmailTemplates,
  type EmailTemplate,
  type EmailTemplateLocale,
} from '@/features/system-settings/operations/email-templates-api'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'
import { formatQuota, parseQuotaFromDollars } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  batchAdjustUserQuota,
  resolveUserManagementOptions,
  searchUserManagementOptions,
} from '../api'
import { validateBatchQuotaInput } from '../lib/user-batch-quota'
import type { QuotaAdjustMode } from '../types'

const QUOTA_TEMPLATE_EVENT = 'user.quota_adjustment'
const QUOTA_MODE_LABEL_KEYS: Record<QuotaAdjustMode, string> = {
  add: 'Add',
  subtract: 'Subtract',
  override: 'Override',
}

type UserBatchQuotaDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export function UserBatchQuotaDialog(props: UserBatchQuotaDialogProps) {
  const { t, i18n } = useTranslation()
  const [mode, setMode] = useState<QuotaAdjustMode>('add')
  const [amount, setAmount] = useState('')
  const [allUsers, setAllUsers] = useState(false)
  const [selectedUserIds, setSelectedUserIds] = useState<number[]>([])
  const [sendEmail, setSendEmail] = useState(false)
  const [templateKey, setTemplateKey] = useState('')
  const [emailLocale, setEmailLocale] = useState<EmailTemplateLocale>('zh')
  const [emailSubject, setEmailSubject] = useState('')
  const [emailContent, setEmailContent] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const templatesQuery = useQuery({
    queryKey: ['email-templates'],
    enabled: props.open,
    queryFn: async () => {
      const response = await listEmailTemplates()
      if (!response.success) {
        throw new Error(response.message || t('Failed to load email templates'))
      }
      return response.data || []
    },
  })
  const quotaTemplates = (templatesQuery.data || []).filter(
    (template) => template.event === QUOTA_TEMPLATE_EVENT
  )

  const handleSendEmailChange = (checked: boolean) => {
    setSendEmail(checked)
    if (!checked || templateKey || quotaTemplates.length === 0) return
    const preferredLocale = i18n.language.startsWith('en') ? 'en' : 'zh'
    const template =
      quotaTemplates.find((item) => item.locale === preferredLocale) ??
      quotaTemplates[0]
    setTemplateKey(`${template.event}::${template.locale}`)
    setEmailLocale(template.locale)
    setEmailSubject(template.subject)
    setEmailContent(template.content)
  }

  const currencyLabel = getCurrencyLabel()
  const tokensOnly = getCurrencyDisplay().meta.kind === 'tokens'
  const amountNumber = Number(amount)
  const quotaValue = parseQuotaFromDollars(amountNumber)
  let quotaPrefix = ''
  if (mode === 'add') quotaPrefix = '+'
  if (mode === 'subtract') quotaPrefix = '-'

  const resetForm = () => {
    setMode('add')
    setAmount('')
    setAllUsers(false)
    setSelectedUserIds([])
    setSendEmail(false)
    setTemplateKey('')
    setEmailLocale('zh')
    setEmailSubject('')
    setEmailContent('')
  }

  const handleOpenChange = (open: boolean) => {
    if (!open) resetForm()
    props.onOpenChange(open)
  }

  const handleTemplateChange = (key: string) => {
    setTemplateKey(key)
    const template = quotaTemplates.find(
      (item) => `${item.event}::${item.locale}` === key
    )
    if (!template) return
    setEmailLocale(template.locale)
    setEmailSubject(template.subject)
    setEmailContent(template.content)
  }

  const handleSubmit = async () => {
    const validationKey = validateBatchQuotaInput({
      mode,
      amount,
      quotaValue,
      allUsers,
      selectedUserIds,
      sendEmail,
      emailSubject,
      emailContent,
    })
    if (validationKey) {
      toast.error(t(validationKey))
      return
    }

    setSubmitting(true)
    try {
      const response = await batchAdjustUserQuota({
        mode,
        value: quotaValue,
        all_users: allUsers,
        user_ids: allUsers ? [] : selectedUserIds,
        send_email: sendEmail,
        email_locale: emailLocale,
        email_subject: sendEmail ? emailSubject : '',
        email_content: sendEmail ? emailContent : '',
      })
      if (!response.success || !response.data) {
        toast.error(response.message || t('Failed to adjust quota'))
        return
      }
      if (sendEmail) {
        toast.success(
          t(
            'Adjusted {{count}} users; email sent {{sent}}, skipped {{skipped}}, failed {{failed}}',
            {
              count: response.data.adjusted_count,
              sent: response.data.email_success_count,
              skipped: response.data.email_skipped_count,
              failed: response.data.email_failed_count,
            }
          )
        )
      } else {
        toast.success(
          t('Quota adjusted for {{count}} users', {
            count: response.data.adjusted_count,
          })
        )
      }
      props.onSuccess()
      handleOpenChange(false)
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to adjust quota')
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleOpenChange}
      title={t('Batch Adjust Quota')}
      contentHeight='min(42rem, calc(100vh - 14rem))'
      contentClassName='sm:max-w-3xl'
      bodyClassName='space-y-5'
      footer={
        <>
          <Button variant='outline' onClick={() => handleOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {submitting && <Loader2 className='animate-spin' />}
            {submitting ? t('Processing...') : t('Confirm')}
          </Button>
        </>
      }
    >
      <section className='space-y-3'>
        <Label>{t('Users')}</Label>
        <div className='flex gap-1'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            className={cn(
              !allUsers &&
                'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
            )}
            aria-pressed={!allUsers}
            onClick={() => setAllUsers(false)}
          >
            {t('Selected users')}
          </Button>
          <Button
            type='button'
            variant='outline'
            size='sm'
            className={cn(
              allUsers &&
                'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
            )}
            aria-pressed={allUsers}
            onClick={() => setAllUsers(true)}
          >
            {t('All users')}
          </Button>
        </div>
        {!allUsers && (
          <EmailCampaignUserPicker
            id='batch-quota-users'
            labelKey='Selected users'
            queryKeyPrefix='batch-quota'
            selectedUserIds={selectedUserIds}
            onChange={setSelectedUserIds}
            searchUsers={searchUserManagementOptions}
            resolveUsers={resolveUserManagementOptions}
          />
        )}
      </section>

      <section className='space-y-4 border-t pt-5'>
        <div className='space-y-2'>
          <Label>{t('Mode')}</Label>
          <div className='flex gap-1'>
            {(['add', 'subtract', 'override'] as const).map((item) => (
              <Button
                key={item}
                type='button'
                variant='outline'
                size='sm'
                className={cn(
                  mode === item &&
                    'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
                )}
                aria-pressed={mode === item}
                onClick={() => {
                  setMode(item)
                  setAmount('')
                }}
              >
                {t(QUOTA_MODE_LABEL_KEYS[item])}
              </Button>
            ))}
          </div>
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor='batch-quota-amount'>
            {t('Amount')} ({currencyLabel})
          </Label>
          <Input
            id='batch-quota-amount'
            type='number'
            step={tokensOnly ? 1 : 0.000001}
            min={mode === 'override' ? undefined : 0}
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
          />
          {amount.trim() !== '' && Number.isFinite(amountNumber) && (
            <div className='text-muted-foreground text-xs tabular-nums'>
              {quotaPrefix}
              {formatQuota(
                mode === 'override' ? quotaValue : Math.abs(quotaValue)
              )}
            </div>
          )}
        </div>
      </section>

      <section className='space-y-4 border-t pt-5'>
        <div className='flex min-h-8 items-center justify-between gap-4'>
          <Label htmlFor='batch-quota-send-email'>{t('Send email')}</Label>
          <Switch
            id='batch-quota-send-email'
            checked={sendEmail}
            aria-label={t('Send email')}
            onCheckedChange={handleSendEmailChange}
          />
        </div>
        {sendEmail && (
          <div className='space-y-4'>
            <div className='space-y-1.5'>
              <Label htmlFor='batch-quota-template'>
                {t('Email template')}
              </Label>
              <NativeSelect
                id='batch-quota-template'
                className='w-full'
                value={templateKey}
                onChange={(event) => handleTemplateChange(event.target.value)}
              >
                <NativeSelectOption value=''>
                  {templatesQuery.isLoading
                    ? t('Loading...')
                    : t('Do not use a template')}
                </NativeSelectOption>
                {quotaTemplates.map((template: EmailTemplate) => (
                  <NativeSelectOption
                    key={`${template.event}::${template.locale}`}
                    value={`${template.event}::${template.locale}`}
                  >
                    {t(template.label)} ·{' '}
                    {template.locale === 'zh' ? t('Chinese') : t('English')}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='batch-quota-email-subject'>
                {t('Email subject')}
              </Label>
              <Input
                id='batch-quota-email-subject'
                maxLength={255}
                value={emailSubject}
                onChange={(event) => setEmailSubject(event.target.value)}
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='batch-quota-email-content'>
                {t('HTML template')}
              </Label>
              <Textarea
                id='batch-quota-email-content'
                className='min-h-48 resize-y font-mono text-xs leading-5'
                maxLength={200000}
                value={emailContent}
                onChange={(event) => setEmailContent(event.target.value)}
              />
            </div>
          </div>
        )}
      </section>
    </Dialog>
  )
}
