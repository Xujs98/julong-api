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
import {
  ChevronLeft,
  ChevronRight,
  Eye,
  MailPlus,
  Pause,
  Pencil,
  Play,
  RefreshCw,
  RotateCcw,
  Trash2,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { HtmlContent } from '@/components/html-content'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { formatTimestampToDate } from '@/lib/format'

import { SettingsSection } from '../components/settings-section'
import { EmailCampaignUserPicker } from './email-campaign-user-picker'
import {
  activateEmailCampaign,
  createEmailCampaign,
  deleteEmailCampaign,
  getEmailCampaignStats,
  listEmailCampaigns,
  listEmailDeliveries,
  pauseEmailCampaign,
  previewEmailCampaign,
  retryEmailCampaign,
  updateEmailCampaign,
  type EmailCampaign,
  type EmailCampaignMode,
  type EmailCampaignPayload,
  type EmailCampaignStatus,
  type EmailCampaignTarget,
} from './email-campaigns-api'
import {
  listEmailTemplates,
  previewEmailTemplate,
  type EmailTemplate,
  type EmailTemplatePreview,
} from './email-templates-api'

const PAGE_SIZE = 20
const DELIVERY_PAGE_SIZE = 50

type CampaignForm = {
  name: string
  subject: string
  content: string
  mode: EmailCampaignMode
  targetType: EmailCampaignTarget
  targetUserIds: number[]
  triggerDays: string
  scheduledAt: string
}

const emptyForm = (): CampaignForm => ({
  name: '',
  subject: '',
  content: '',
  mode: 'immediate',
  targetType: 'all_users',
  targetUserIds: [],
  triggerDays: '3',
  scheduledAt: '',
})

function timestampToLocalInput(timestamp: number) {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

function campaignToForm(campaign: EmailCampaign): CampaignForm {
  return {
    name: campaign.name,
    subject: campaign.subject,
    content: campaign.content,
    mode: campaign.mode,
    targetType: campaign.target_type,
    targetUserIds: [...campaign.target_user_ids],
    triggerDays: String(campaign.trigger_days || 3),
    scheduledAt: timestampToLocalInput(campaign.scheduled_at),
  }
}

function formToPayload(form: CampaignForm): EmailCampaignPayload {
  return {
    name: form.name.trim(),
    subject: form.subject.trim(),
    content: form.content.trim(),
    mode: form.mode,
    target_type: form.targetType,
    target_user_ids: [...new Set(form.targetUserIds)],
    trigger_type: form.mode === 'conditional' ? 'subscription_expiring' : '',
    trigger_days:
      form.mode === 'conditional'
        ? Number.parseInt(form.triggerDays, 10) || 0
        : 0,
    scheduled_at:
      form.mode === 'scheduled' && form.scheduledAt
        ? Math.floor(new Date(form.scheduledAt).getTime() / 1000)
        : 0,
  }
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className='min-w-0 border-r px-3 last:border-r-0'>
      <div className='text-muted-foreground truncate text-xs'>{label}</div>
      <div className='mt-1 text-lg font-semibold tabular-nums'>{value}</div>
    </div>
  )
}

export function EmailCampaignsSection() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<EmailCampaign | null>(null)
  const [form, setForm] = useState<CampaignForm>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [previewing, setPreviewing] = useState(false)
  const [previewCount, setPreviewCount] = useState<number | null>(null)
  const [templateKey, setTemplateKey] = useState('')
  const [templatePreview, setTemplatePreview] =
    useState<EmailTemplatePreview | null>(null)
  const [templatePreviewing, setTemplatePreviewing] = useState(false)
  const [selected, setSelected] = useState<EmailCampaign | null>(null)
  const [deliveryPage, setDeliveryPage] = useState(1)
  const [deleting, setDeleting] = useState<EmailCampaign | null>(null)

  const campaignsQuery = useQuery({
    queryKey: ['email-campaigns', page],
    queryFn: async () => {
      const response = await listEmailCampaigns(page, PAGE_SIZE)
      if (!response.success) {
        throw new Error(response.message || t('Failed to load email campaigns'))
      }
      return (
        response.data || {
          page,
          page_size: PAGE_SIZE,
          total: 0,
          items: [],
        }
      )
    },
    placeholderData: (previous) => previous,
  })

  const deliveriesQuery = useQuery({
    queryKey: ['email-campaign-deliveries', selected?.id, deliveryPage],
    enabled: selected !== null,
    queryFn: async () => {
      if (!selected) return null
      const response = await listEmailDeliveries(
        selected.id,
        deliveryPage,
        DELIVERY_PAGE_SIZE
      )
      if (!response.success) {
        throw new Error(
          response.message || t('Failed to load delivery records')
        )
      }
      return response.data || null
    },
    placeholderData: (previous) => previous,
  })

  const statsQuery = useQuery({
    queryKey: ['email-campaign-stats'],
    queryFn: async () => {
      const response = await getEmailCampaignStats()
      if (!response.success) {
        throw new Error(response.message || t('Failed to load email campaigns'))
      }
      return response.data
    },
  })

  const templatesQuery = useQuery({
    queryKey: ['email-templates', 'campaign-picker'],
    enabled: formOpen,
    queryFn: async () => {
      const response = await listEmailTemplates()
      if (!response.success) {
        throw new Error(response.message || t('Failed to load email templates'))
      }
      return (response.data || []).filter(
        (template) => template.campaign_compatible
      )
    },
  })

  const campaignTemplates = templatesQuery.data || []
  const selectedCampaignTemplate = campaignTemplates.find(
    (template) => `${template.event}::${template.locale}` === templateKey
  )

  const totalPages = Math.max(
    1,
    Math.ceil((campaignsQuery.data?.total || 0) / PAGE_SIZE)
  )
  const deliveryPages = Math.max(
    1,
    Math.ceil((deliveriesQuery.data?.total || 0) / DELIVERY_PAGE_SIZE)
  )
  const statusLabel = (status: EmailCampaignStatus | string) => {
    const labels: Record<string, string> = {
      draft: t('Draft'),
      scheduled: t('Scheduled'),
      active: t('Active'),
      running: t('Sending'),
      completed: t('Completed'),
      partial_failed: t('Partially failed'),
      paused: t('Paused'),
      pending: t('Pending'),
      sending: t('Sending'),
      sent: t('Sent'),
      failed: t('Failed'),
      skipped: t('Skipped'),
    }
    return labels[status] || status
  }

  const modeLabel = (mode: EmailCampaignMode) => {
    if (mode === 'scheduled') return t('Scheduled send')
    if (mode === 'conditional') return t('Conditional trigger')
    return t('Send now')
  }

  const targetLabel = (target: EmailCampaignTarget) => {
    if (target === 'active_subscribers') return t('Active subscribers')
    if (target === 'selected_users') return t('Selected users')
    return t('All active users')
  }

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm())
    setPreviewCount(null)
    setTemplateKey('')
    setTemplatePreview(null)
    setFormOpen(true)
  }

  const openEdit = (campaign: EmailCampaign) => {
    setEditing(campaign)
    setForm(campaignToForm(campaign))
    setPreviewCount(null)
    setTemplateKey('')
    setTemplatePreview(null)
    setFormOpen(true)
  }

  const applySelectedTemplate = () => {
    if (!selectedCampaignTemplate) return
    setForm((current) => ({
      ...current,
      subject: selectedCampaignTemplate.subject,
      content: selectedCampaignTemplate.content,
    }))
    toast.success(t('Email template applied'))
  }

  const previewSelectedTemplate = async () => {
    if (!selectedCampaignTemplate) return
    setTemplatePreviewing(true)
    try {
      const response = await previewEmailTemplate(
        selectedCampaignTemplate.event,
        selectedCampaignTemplate.locale,
        selectedCampaignTemplate.subject,
        selectedCampaignTemplate.content
      )
      if (!response.success || !response.data) throw new Error(response.message)
      setTemplatePreview(response.data)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to preview email template')
      )
    } finally {
      setTemplatePreviewing(false)
    }
  }

  const validateForm = () => {
    if (!form.name.trim() || !form.subject.trim() || !form.content.trim()) {
      toast.error(t('Name, subject, and content are required'))
      return false
    }
    if (form.mode === 'scheduled' && !form.scheduledAt) {
      toast.error(t('Select a scheduled send time'))
      return false
    }
    if (
      form.mode !== 'conditional' &&
      form.targetType === 'selected_users' &&
      form.targetUserIds.length === 0
    ) {
      toast.error(t('Enter at least one user ID'))
      return false
    }
    return true
  }

  const handlePreview = async () => {
    if (!validateForm()) return
    setPreviewing(true)
    try {
      const response = await previewEmailCampaign(formToPayload(form))
      if (!response.success) throw new Error(response.message)
      setPreviewCount(response.data?.recipient_count ?? 0)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to preview recipients')
      )
    } finally {
      setPreviewing(false)
    }
  }

  const handleSave = async (draft: boolean) => {
    if (!validateForm()) return
    setSaving(true)
    try {
      const payload = { ...formToPayload(form), draft }
      const response = editing
        ? await updateEmailCampaign(editing.id, payload)
        : await createEmailCampaign(payload)
      if (!response.success) throw new Error(response.message)
      toast.success(
        editing ? t('Email campaign updated') : t('Email campaign created')
      )
      setFormOpen(false)
      await Promise.all([campaignsQuery.refetch(), statsQuery.refetch()])
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save email campaign')
      )
    } finally {
      setSaving(false)
    }
  }

  const runAction = async (
    action: () => Promise<{ success: boolean; message?: string }>,
    successMessage: string
  ) => {
    try {
      const response = await action()
      if (!response.success) throw new Error(response.message)
      toast.success(successMessage)
      await Promise.all([campaignsQuery.refetch(), statsQuery.refetch()])
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Operation failed')
      )
    }
  }

  let campaignRows: ReactNode
  if (campaignsQuery.isLoading) {
    campaignRows = (
      <TableRow>
        <TableCell colSpan={7} className='h-28 text-center'>
          {t('Loading...')}
        </TableCell>
      </TableRow>
    )
  } else if ((campaignsQuery.data?.items || []).length === 0) {
    campaignRows = (
      <TableRow>
        <TableCell
          colSpan={7}
          className='text-muted-foreground h-36 text-center'
        >
          {t('No email campaigns')}
        </TableCell>
      </TableRow>
    )
  } else {
    campaignRows = campaignsQuery.data?.items.map((campaign) => (
      <TableRow key={campaign.id}>
        <TableCell className='max-w-[240px]'>
          <div className='truncate font-medium'>{campaign.name}</div>
          <div className='text-muted-foreground truncate text-xs'>
            {campaign.subject}
          </div>
        </TableCell>
        <TableCell>{modeLabel(campaign.mode)}</TableCell>
        <TableCell>{targetLabel(campaign.target_type)}</TableCell>
        <TableCell>
          <Badge
            variant={
              campaign.status === 'partial_failed' ? 'destructive' : 'outline'
            }
          >
            {statusLabel(campaign.status)}
          </Badge>
        </TableCell>
        <TableCell>
          <span className='text-emerald-600'>{campaign.success_count}</span>
          {' / '}
          <span className='text-destructive'>{campaign.failed_count}</span>
          {' / '}
          {campaign.recipient_count}
        </TableCell>
        <TableCell>
          {campaign.next_run_at
            ? formatTimestampToDate(campaign.next_run_at)
            : '-'}
        </TableCell>
        <TableCell>
          <div className='flex justify-end gap-1'>
            <Button
              type='button'
              size='icon-sm'
              variant='ghost'
              title={t('View')}
              aria-label={t('View')}
              onClick={() => {
                setSelected(campaign)
                setDeliveryPage(1)
              }}
            >
              <Eye className='size-4' />
            </Button>
            {(campaign.status === 'draft' || campaign.status === 'paused') && (
              <>
                <Button
                  type='button'
                  size='icon-sm'
                  variant='ghost'
                  title={t('Edit')}
                  aria-label={t('Edit')}
                  onClick={() => openEdit(campaign)}
                >
                  <Pencil className='size-4' />
                </Button>
                <Button
                  type='button'
                  size='icon-sm'
                  variant='ghost'
                  title={t('Activate')}
                  aria-label={t('Activate')}
                  onClick={() =>
                    runAction(
                      () => activateEmailCampaign(campaign.id),
                      t('Email campaign activated')
                    )
                  }
                >
                  <Play className='size-4' />
                </Button>
              </>
            )}
            {(campaign.status === 'scheduled' ||
              campaign.status === 'active') && (
              <Button
                type='button'
                size='icon-sm'
                variant='ghost'
                title={t('Pause')}
                aria-label={t('Pause')}
                onClick={() =>
                  runAction(
                    () => pauseEmailCampaign(campaign.id),
                    t('Email campaign paused')
                  )
                }
              >
                <Pause className='size-4' />
              </Button>
            )}
            {campaign.status === 'partial_failed' && (
              <Button
                type='button'
                size='icon-sm'
                variant='ghost'
                title={t('Retry failed deliveries')}
                aria-label={t('Retry failed deliveries')}
                onClick={() =>
                  runAction(
                    () => retryEmailCampaign(campaign.id),
                    t('Failed deliveries queued for retry')
                  )
                }
              >
                <RotateCcw className='size-4' />
              </Button>
            )}
            {campaign.status !== 'running' && (
              <Button
                type='button'
                size='icon-sm'
                variant='ghost'
                className='text-destructive'
                title={t('Delete')}
                aria-label={t('Delete')}
                onClick={() => setDeleting(campaign)}
              >
                <Trash2 className='size-4' />
              </Button>
            )}
          </div>
        </TableCell>
      </TableRow>
    ))
  }

  let deliveryRows: ReactNode
  if (deliveriesQuery.isLoading) {
    deliveryRows = (
      <TableRow>
        <TableCell colSpan={7} className='h-24 text-center'>
          {t('Loading...')}
        </TableCell>
      </TableRow>
    )
  } else if ((deliveriesQuery.data?.items || []).length === 0) {
    deliveryRows = (
      <TableRow>
        <TableCell
          colSpan={7}
          className='text-muted-foreground h-28 text-center'
        >
          {t('No delivery records')}
        </TableCell>
      </TableRow>
    )
  } else {
    deliveryRows = deliveriesQuery.data?.items.map((delivery) => (
      <TableRow key={delivery.id}>
        <TableCell>
          {delivery.display_name || delivery.username || delivery.user_id}
        </TableCell>
        <TableCell>{delivery.email}</TableCell>
        <TableCell>{delivery.subscription_title || '-'}</TableCell>
        <TableCell>
          <Badge
            variant={delivery.status === 'failed' ? 'destructive' : 'outline'}
          >
            {statusLabel(delivery.status)}
          </Badge>
        </TableCell>
        <TableCell>{delivery.attempt_count}</TableCell>
        <TableCell>
          {delivery.sent_at ? formatTimestampToDate(delivery.sent_at) : '-'}
        </TableCell>
        <TableCell
          className='max-w-[260px] truncate'
          title={delivery.last_error}
        >
          {delivery.last_error || '-'}
        </TableCell>
      </TableRow>
    ))
  }

  return (
    <SettingsSection title={t('Email Campaigns')}>
      <div className='space-y-4'>
        <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <div className='grid grid-cols-3 divide-x rounded-lg border py-3 sm:min-w-[420px]'>
            <Stat
              label={t('Recipients')}
              value={statsQuery.data?.recipient_count || 0}
            />
            <Stat
              label={t('Sent')}
              value={statsQuery.data?.success_count || 0}
            />
            <Stat
              label={t('Failed')}
              value={statsQuery.data?.failed_count || 0}
            />
          </div>
          <div className='flex justify-end gap-2'>
            <Button
              type='button'
              variant='outline'
              size='icon'
              aria-label={t('Refresh')}
              title={t('Refresh')}
              onClick={() =>
                Promise.all([campaignsQuery.refetch(), statsQuery.refetch()])
              }
            >
              <RefreshCw className='size-4' />
            </Button>
            <Button type='button' onClick={openCreate}>
              <MailPlus className='size-4' />
              {t('Create email campaign')}
            </Button>
          </div>
        </div>

        <div className='overflow-hidden rounded-lg border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Campaign')}</TableHead>
                <TableHead>{t('Mode')}</TableHead>
                <TableHead>{t('Audience')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead>{t('Delivery')}</TableHead>
                <TableHead>{t('Next run')}</TableHead>
                <TableHead className='text-right'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>{campaignRows}</TableBody>
          </Table>
          <div className='bg-muted/30 flex items-center justify-between border-t p-3'>
            <span className='text-muted-foreground text-sm'>
              {t('Total')} {campaignsQuery.data?.total || 0}
            </span>
            <div className='flex items-center gap-2'>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Previous')}
                disabled={page <= 1}
                onClick={() => setPage((value) => Math.max(1, value - 1))}
              >
                <ChevronLeft className='size-4' />
              </Button>
              <span className='text-muted-foreground min-w-14 text-center text-sm'>
                {page} / {totalPages}
              </span>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Next')}
                disabled={page >= totalPages}
                onClick={() => setPage((value) => value + 1)}
              >
                <ChevronRight className='size-4' />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <Dialog
        open={formOpen}
        onOpenChange={setFormOpen}
        title={editing ? t('Edit email campaign') : t('Create email campaign')}
        contentClassName='sm:max-w-3xl'
        contentHeight='min(68vh, 720px)'
        footer={
          <>
            <Button
              type='button'
              variant='secondary'
              onClick={() => setFormOpen(false)}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              variant='outline'
              disabled={previewing || saving}
              onClick={handlePreview}
            >
              <Eye className='size-4' />
              {previewCount === null
                ? t('Preview recipients')
                : t('{{count}} recipients', { count: previewCount })}
            </Button>
            {!editing && (
              <Button
                type='button'
                variant='outline'
                disabled={saving}
                onClick={() => handleSave(true)}
              >
                {t('Save draft')}
              </Button>
            )}
            <Button
              type='button'
              disabled={saving}
              onClick={() => handleSave(false)}
            >
              {editing ? t('Save') : t('Create task')}
            </Button>
          </>
        }
      >
        <div className='grid gap-5'>
          <div className='grid gap-4 sm:grid-cols-2'>
            <div className='space-y-1.5'>
              <Label htmlFor='campaign-name'>{t('Campaign name')}</Label>
              <Input
                id='campaign-name'
                maxLength={128}
                value={form.name}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='campaign-mode'>{t('Sending mode')}</Label>
              <NativeSelect
                id='campaign-mode'
                className='w-full'
                value={form.mode}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    mode: event.target.value as EmailCampaignMode,
                  }))
                }
              >
                <NativeSelectOption value='immediate'>
                  {t('Send now')}
                </NativeSelectOption>
                <NativeSelectOption value='scheduled'>
                  {t('Scheduled send')}
                </NativeSelectOption>
                <NativeSelectOption value='conditional'>
                  {t('Conditional trigger')}
                </NativeSelectOption>
              </NativeSelect>
            </div>
          </div>

          <div className='space-y-2 rounded-lg border p-3'>
            <div className='flex flex-col gap-2 sm:flex-row sm:items-end'>
              <div className='min-w-0 flex-1 space-y-1.5'>
                <Label htmlFor='campaign-template'>{t('Email template')}</Label>
                <NativeSelect
                  id='campaign-template'
                  className='w-full'
                  value={templateKey}
                  onChange={(event) => {
                    setTemplateKey(event.target.value)
                    setTemplatePreview(null)
                  }}
                >
                  <NativeSelectOption value=''>
                    {t('Do not use a template')}
                  </NativeSelectOption>
                  {campaignTemplates.map((template: EmailTemplate) => (
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
              <div className='flex shrink-0 gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  disabled={!selectedCampaignTemplate || templatePreviewing}
                  onClick={previewSelectedTemplate}
                >
                  <Eye className='size-4' />
                  {t('Preview template')}
                </Button>
                <Button
                  type='button'
                  variant='secondary'
                  disabled={!selectedCampaignTemplate}
                  onClick={applySelectedTemplate}
                >
                  {t('Apply template')}
                </Button>
              </div>
            </div>
            <p className='text-muted-foreground text-xs'>
              {t(
                'Applying a template replaces the current subject and HTML content.'
              )}
            </p>
            {templatePreview && (
              <div className='overflow-hidden rounded-md border'>
                <div className='bg-muted/30 border-b px-3 py-2 text-xs font-medium'>
                  {templatePreview.subject}
                </div>
                <div className='max-h-72 overflow-auto bg-white p-2 text-black'>
                  <HtmlContent
                    content={templatePreview.content}
                    variant='isolated'
                  />
                </div>
              </div>
            )}
          </div>

          {form.mode === 'scheduled' && (
            <div className='space-y-1.5'>
              <Label htmlFor='campaign-scheduled-at'>
                {t('Scheduled time')}
              </Label>
              <Input
                id='campaign-scheduled-at'
                type='datetime-local'
                value={form.scheduledAt}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    scheduledAt: event.target.value,
                  }))
                }
              />
            </div>
          )}

          {form.mode === 'conditional' ? (
            <div className='grid gap-4 sm:grid-cols-2'>
              <div className='space-y-1.5'>
                <Label>{t('Trigger condition')}</Label>
                <Input value={t('Subscription expiring')} disabled />
              </div>
              <div className='space-y-1.5'>
                <Label htmlFor='campaign-trigger-days'>
                  {t('Days before expiry')}
                </Label>
                <Input
                  id='campaign-trigger-days'
                  type='number'
                  min={1}
                  max={90}
                  value={form.triggerDays}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      triggerDays: event.target.value,
                    }))
                  }
                />
              </div>
            </div>
          ) : (
            <div className='space-y-1.5'>
              <Label htmlFor='campaign-target'>{t('Audience')}</Label>
              <NativeSelect
                id='campaign-target'
                className='w-full'
                value={form.targetType}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    targetType: event.target.value as EmailCampaignTarget,
                  }))
                }
              >
                <NativeSelectOption value='all_users'>
                  {t('All active users')}
                </NativeSelectOption>
                <NativeSelectOption value='active_subscribers'>
                  {t('Active subscribers')}
                </NativeSelectOption>
                <NativeSelectOption value='selected_users'>
                  {t('Selected users')}
                </NativeSelectOption>
              </NativeSelect>
            </div>
          )}

          {form.mode !== 'conditional' &&
            form.targetType === 'selected_users' && (
              <div className='space-y-1.5'>
                <Label htmlFor='campaign-user-picker'>
                  {t('Selected users')}
                </Label>
                <EmailCampaignUserPicker
                  selectedUserIds={form.targetUserIds}
                  onChange={(targetUserIds) =>
                    setForm((current) => ({ ...current, targetUserIds }))
                  }
                />
              </div>
            )}

          <div className='space-y-1.5'>
            <Label htmlFor='campaign-subject'>{t('Email subject')}</Label>
            <Input
              id='campaign-subject'
              maxLength={255}
              value={form.subject}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  subject: event.target.value,
                }))
              }
            />
          </div>
          <div className='space-y-1.5'>
            <div className='flex items-center justify-between gap-2'>
              <Label htmlFor='campaign-content'>{t('Email content')}</Label>
              <NativeSelect
                aria-label={t('Insert variable')}
                value=''
                onChange={(event) => {
                  if (!event.target.value) return
                  setForm((current) => ({
                    ...current,
                    content: current.content + event.target.value,
                  }))
                }}
              >
                <NativeSelectOption value=''>
                  {t('Insert variable')}
                </NativeSelectOption>
                <NativeSelectOption value='{{username}}'>
                  {t('Username')}
                </NativeSelectOption>
                <NativeSelectOption value='{{display_name}}'>
                  {t('Display name')}
                </NativeSelectOption>
                <NativeSelectOption value='{{email}}'>
                  {t('User email')}
                </NativeSelectOption>
                <NativeSelectOption value='{{system_name}}'>
                  {t('System name')}
                </NativeSelectOption>
                <NativeSelectOption value='{{subscription_name}}'>
                  {t('Subscription plan')}
                </NativeSelectOption>
                <NativeSelectOption value='{{subscription_end_time}}'>
                  {t('Expiry time')}
                </NativeSelectOption>
                <NativeSelectOption value='{{days_remaining}}'>
                  {t('Days remaining')}
                </NativeSelectOption>
              </NativeSelect>
            </div>
            <Textarea
              id='campaign-content'
              rows={12}
              value={form.content}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  content: event.target.value,
                }))
              }
            />
          </div>
        </div>
      </Dialog>

      <Dialog
        open={selected !== null}
        onOpenChange={(open) => !open && setSelected(null)}
        title={selected?.name || t('Email campaign details')}
        description={selected?.subject}
        contentClassName='sm:max-w-5xl'
        contentHeight='min(72vh, 760px)'
      >
        {selected && (
          <div className='space-y-5'>
            <div className='bg-border grid gap-px overflow-hidden rounded-lg border sm:grid-cols-4'>
              {[
                [t('Status'), statusLabel(selected.status)],
                [t('Recipients'), selected.recipient_count],
                [t('Sent'), selected.success_count],
                [t('Failed'), selected.failed_count],
              ].map(([label, value]) => (
                <div key={String(label)} className='bg-background p-3'>
                  <div className='text-muted-foreground text-xs'>{label}</div>
                  <div className='mt-1 font-medium'>{value}</div>
                </div>
              ))}
            </div>
            {selected.last_error && (
              <div className='border-destructive/30 bg-destructive/5 text-destructive rounded-lg border p-3 text-sm break-words'>
                {selected.last_error}
              </div>
            )}
            <div className='overflow-hidden rounded-lg border'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('User')}</TableHead>
                    <TableHead>{t('Email')}</TableHead>
                    <TableHead>{t('Subscription plan')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('Attempts')}</TableHead>
                    <TableHead>{t('Sent at')}</TableHead>
                    <TableHead>{t('Error')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>{deliveryRows}</TableBody>
              </Table>
              <div className='bg-muted/30 flex items-center justify-end gap-2 border-t p-3'>
                <Button
                  type='button'
                  variant='outline'
                  size='icon-sm'
                  aria-label={t('Previous')}
                  disabled={deliveryPage <= 1}
                  onClick={() =>
                    setDeliveryPage((value) => Math.max(1, value - 1))
                  }
                >
                  <ChevronLeft className='size-4' />
                </Button>
                <span className='text-muted-foreground min-w-14 text-center text-sm'>
                  {deliveryPage} / {deliveryPages}
                </span>
                <Button
                  type='button'
                  variant='outline'
                  size='icon-sm'
                  aria-label={t('Next')}
                  disabled={deliveryPage >= deliveryPages}
                  onClick={() => setDeliveryPage((value) => value + 1)}
                >
                  <ChevronRight className='size-4' />
                </Button>
              </div>
            </div>
          </div>
        )}
      </Dialog>

      <Dialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={t('Delete email campaign')}
        description={t(
          'This will permanently delete the campaign and all delivery records.'
        )}
        footer={
          <>
            <Button
              type='button'
              variant='secondary'
              onClick={() => setDeleting(null)}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              variant='destructive'
              onClick={async () => {
                if (!deleting) return
                const id = deleting.id
                setDeleting(null)
                await runAction(
                  () => deleteEmailCampaign(id),
                  t('Email campaign deleted')
                )
              }}
            >
              <Trash2 className='size-4' />
              {t('Delete')}
            </Button>
          </>
        }
      >
        <div />
      </Dialog>
    </SettingsSection>
  )
}
