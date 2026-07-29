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
import { Eye, RotateCcw, Save } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { HtmlContent } from '@/components/html-content'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'

import { SettingsSection } from '../components/settings-section'
import { EmailAlertSettings } from './email-alert-settings'
import {
  listEmailTemplates,
  previewEmailTemplate,
  resetEmailTemplate,
  updateEmailTemplate,
  type EmailTemplateLocale,
  type EmailTemplatePreview,
} from './email-templates-api'

const DEFAULT_EMAIL_TEMPLATE_EVENT = 'auth.verify_code'

const PLACEHOLDER_LABELS: Record<string, string> = {
  system_name: 'System name',
  username: 'Username',
  display_name: 'Display name',
  email: 'User email',
  verification_code: 'Email verification code',
  expires_in_minutes: 'Validity period (minutes)',
  reset_url: 'Password reset link',
  subscription_name: 'Subscription plan',
  subscription_end_time: 'Expiry time',
  days_remaining: 'Days remaining',
  balance_type: 'Balance type',
  current_balance: 'Current balance',
  warning_threshold: 'Warning threshold',
  recharge_url: 'Recharge page URL',
  channel_id: 'Channel ID',
  channel_name: 'Channel name',
  channel_type: 'Channel type',
  channel_base_url: 'Channel base URL',
  checked_at: 'Balance check time',
  disabled_at: 'Automatic shutdown time',
  failure_reason: 'Failure reason',
  report_type: 'Report type',
  report_period: 'Report period',
  generated_at: 'Generated at',
  total_consumption: 'Total consumption',
  total_quota: 'Total quota',
  total_requests: 'Total requests',
  total_tokens: 'Total tokens',
  active_users: 'Active users',
  active_models: 'Active models',
  active_channels: 'Active channels',
  active_groups: 'Active groups',
  top_models: 'Top models',
  top_users: 'User Analytics',
  group_analysis: 'Group Data Analysis',
  alert_mode: 'Alert mode',
  risk_user_count: 'Risk user count',
  risk_levels: 'Risk levels',
  risk_users: 'Risk user details',
  window_days: 'Evaluation window (days)',
  detected_at: 'Detection time',
  monitored_user_id: 'Monitored user ID',
  monitored_username: 'Monitored username',
  monitored_display_name: 'Monitored display name',
  monitored_email: 'Monitored user email',
  presence_status: 'Presence status',
  activity_source: 'Activity source',
  activity_ip: 'Activity IP',
  user_agent: 'User agent',
  activity_at: 'Last activity time',
  offline_at: 'Offline detection time',
  inactivity_minutes: 'Inactive duration (minutes)',
  operation: 'Quota operation',
  adjustment_amount: 'Adjustment amount',
  previous_quota: 'Previous quota',
  current_quota: 'Current quota',
  operator_name: 'Operator name',
  adjusted_at: 'Adjustment time',
}

const SAMPLE_VALUES: Record<EmailTemplateLocale, Record<string, string>> = {
  zh: {
    system_name: '矩龙-API',
    username: 'demo_user',
    display_name: '示例用户',
    email: 'user@example.com',
    verification_code: '123456',
    expires_in_minutes: '15',
    reset_url: 'https://example.com/user/reset?token=preview',
    subscription_name: 'Pro',
    subscription_end_time: '2026-08-01 12:00:00',
    days_remaining: '3',
    balance_type: '钱包余额',
    current_balance: '$2.50',
    warning_threshold: '$5.00',
    recharge_url: 'https://example.com/wallet',
    channel_id: '18',
    channel_name: 'OpenAI Production',
    channel_type: 'OpenAI',
    channel_base_url: 'https://api.openai.com',
    checked_at: '2026-08-01 12:00:00',
    disabled_at: '2026-08-01 12:00:00',
    failure_reason: '上游连续返回 HTTP 401，系统已执行自动封禁。',
    report_type: '日报',
    report_period: '2026-07-26 00:00 - 2026-07-27 00:00',
    generated_at: '2026-07-27 08:00:00',
    total_consumption: '$128.50',
    total_quota: '64,250,000',
    total_requests: '12,580',
    total_tokens: '48,320,000',
    active_users: '328',
    active_models: '18',
    active_channels: '12',
    active_groups: '4',
    top_models:
      '1. gpt-4.1  $52.30\n2. claude-sonnet-4  $41.20\n3. gemini-2.5-pro  $23.80',
    top_users:
      '1. alice  消费 $62.10 | 请求 5,320 | Token 20,100,000\n2. bob  消费 $41.30 | 请求 3,210 | Token 15,200,000',
    group_analysis:
      '1. default  消费 $72.40 | 请求 7,100 | Token 27,800,000 | 用户 210\n2. vip  消费 $56.10 | 请求 5,480 | Token 20,520,000 | 用户 118',
    alert_mode: '测试邮件',
    risk_user_count: '2',
    risk_levels: '中风险、高风险',
    risk_users:
      '#1024 risk_medium_demo（风险测试用户） | 中风险 30 分 | 请求 12 | 错误 3 | 返还 0 | 信号：高错误率\n#1025 risk_high_demo（高风险测试用户） | 高风险 75 分 | 请求 18 | 错误 10 | 返还 3 | 信号：高错误率、输出后返还',
    window_days: '7',
    detected_at: '2026-07-28 12:00:00',
    monitored_user_id: '1024',
    monitored_username: 'monitored_user',
    monitored_display_name: '监控用户',
    monitored_email: 'monitored@example.com',
    presence_status: '离线',
    activity_source: 'API 调用',
    activity_ip: '203.0.113.10',
    user_agent: 'Codex CLI/1.0',
    activity_at: '2026-07-29 12:00:00',
    offline_at: '2026-07-29 12:05:00',
    inactivity_minutes: '5',
    operation: '增加',
    adjustment_amount: '$10.00',
    previous_quota: '$20.00',
    current_quota: '$30.00',
    operator_name: 'root',
    adjusted_at: '2026-07-27 12:00:00',
  },
  en: {
    system_name: 'Julong API',
    username: 'demo_user',
    display_name: 'Demo User',
    email: 'user@example.com',
    verification_code: '123456',
    expires_in_minutes: '15',
    reset_url: 'https://example.com/user/reset?token=preview',
    subscription_name: 'Pro',
    subscription_end_time: '2026-08-01 12:00:00',
    days_remaining: '3',
    balance_type: 'wallet balance',
    current_balance: '$2.50',
    warning_threshold: '$5.00',
    recharge_url: 'https://example.com/wallet',
    channel_id: '18',
    channel_name: 'OpenAI Production',
    channel_type: 'OpenAI',
    channel_base_url: 'https://api.openai.com',
    checked_at: '2026-08-01 12:00:00',
    disabled_at: '2026-08-01 12:00:00',
    failure_reason: 'The upstream returned HTTP 401 repeatedly.',
    report_type: 'Daily',
    report_period: '2026-07-26 00:00 - 2026-07-27 00:00',
    generated_at: '2026-07-27 08:00:00',
    total_consumption: '$128.50',
    total_quota: '64,250,000',
    total_requests: '12,580',
    total_tokens: '48,320,000',
    active_users: '328',
    active_models: '18',
    active_channels: '12',
    active_groups: '4',
    top_models:
      '1. gpt-4.1  $52.30\n2. claude-sonnet-4  $41.20\n3. gemini-2.5-pro  $23.80',
    top_users:
      '1. alice  Consumption $62.10 | Requests 5,320 | Tokens 20,100,000\n2. bob  Consumption $41.30 | Requests 3,210 | Tokens 15,200,000',
    group_analysis:
      '1. default  Consumption $72.40 | Requests 7,100 | Tokens 27,800,000 | Users 210\n2. vip  Consumption $56.10 | Requests 5,480 | Tokens 20,520,000 | Users 118',
    alert_mode: 'Test email',
    risk_user_count: '2',
    risk_levels: 'Medium risk, High risk',
    risk_users:
      '#1024 risk_medium_demo (Risk Demo User) | Medium risk 30 pts | Requests 12 | Errors 3 | Refunds 0 | Signals: High error rate\n#1025 risk_high_demo (High Risk Demo User) | High risk 75 pts | Requests 18 | Errors 10 | Refunds 3 | Signals: High error rate, Refund after output',
    window_days: '7',
    detected_at: '2026-07-28 12:00:00',
    monitored_user_id: '1024',
    monitored_username: 'monitored_user',
    monitored_display_name: 'Monitored User',
    monitored_email: 'monitored@example.com',
    presence_status: 'Offline',
    activity_source: 'API call',
    activity_ip: '203.0.113.10',
    user_agent: 'Codex CLI/1.0',
    activity_at: '2026-07-29 12:00:00',
    offline_at: '2026-07-29 12:05:00',
    inactivity_minutes: '5',
    operation: 'increase',
    adjustment_amount: '$10.00',
    previous_quota: '$20.00',
    current_quota: '$30.00',
    operator_name: 'root',
    adjusted_at: '2026-07-27 12:00:00',
  },
}

type ActiveField = 'subject' | 'content'

function renderSample(source: string, locale: EmailTemplateLocale) {
  let result = source
  for (const [key, value] of Object.entries(SAMPLE_VALUES[locale])) {
    result = result.replaceAll(`{{${key}}}`, value)
  }
  return result
}

export function EmailTemplateSettingsSection() {
  const { t } = useTranslation()
  const templatesQuery = useQuery({
    queryKey: ['email-templates'],
    queryFn: listEmailTemplates,
  })
  const [selectedEvent, setSelectedEvent] = useState(
    DEFAULT_EMAIL_TEMPLATE_EVENT
  )
  const [selectedLocale, setSelectedLocale] =
    useState<EmailTemplateLocale>('zh')
  const [subject, setSubject] = useState('')
  const [content, setContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [previewing, setPreviewing] = useState(false)
  const [serverPreview, setServerPreview] =
    useState<EmailTemplatePreview | null>(null)
  const activeField = useRef<ActiveField>('content')
  const subjectRef = useRef<HTMLInputElement>(null)
  const contentRef = useRef<HTMLTextAreaElement>(null)

  const templates = useMemo(
    () => templatesQuery.data?.data ?? [],
    [templatesQuery.data?.data]
  )
  const events = useMemo(
    () => templates.filter((template) => template.locale === 'zh'),
    [templates]
  )
  const selectedTemplate = templates.find(
    (template) =>
      template.event === selectedEvent && template.locale === selectedLocale
  )
  const preview = serverPreview ?? {
    subject: renderSample(subject, selectedLocale),
    content: renderSample(content, selectedLocale),
  }

  useEffect(() => {
    if (!selectedTemplate) return
    setSubject(selectedTemplate.subject)
    setContent(selectedTemplate.content)
    setServerPreview(null)
  }, [selectedTemplate])

  useEffect(() => setServerPreview(null), [subject, content])

  const recordActiveField = (field: ActiveField) => () => {
    activeField.current = field
  }

  const insertPlaceholder = (placeholder: string) => {
    const token = `{{${placeholder}}}`
    const field = activeField.current
    const ref = field === 'subject' ? subjectRef : contentRef
    const value = field === 'subject' ? subject : content
    const setValue = field === 'subject' ? setSubject : setContent
    const start = ref.current?.selectionStart ?? value.length
    const end = ref.current?.selectionEnd ?? value.length
    setValue(`${value.slice(0, start)}${token}${value.slice(end)}`)
    requestAnimationFrame(() => {
      ref.current?.focus()
      ref.current?.setSelectionRange(start + token.length, start + token.length)
    })
  }

  const saveTemplate = async () => {
    if (!selectedTemplate) return
    setSaving(true)
    try {
      const response = await updateEmailTemplate(
        selectedTemplate.event,
        selectedLocale,
        subject,
        content
      )
      if (!response.success) throw new Error(response.message)
      toast.success(t('Email template saved'))
      await templatesQuery.refetch()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save email template')
      )
    } finally {
      setSaving(false)
    }
  }

  const refreshPreview = async () => {
    if (!selectedTemplate) return
    setPreviewing(true)
    try {
      const response = await previewEmailTemplate(
        selectedTemplate.event,
        selectedLocale,
        subject,
        content
      )
      if (!response.success || !response.data) throw new Error(response.message)
      setServerPreview(response.data)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to preview email template')
      )
    } finally {
      setPreviewing(false)
    }
  }

  const restoreDefault = async () => {
    if (!selectedTemplate) return
    setResetting(true)
    try {
      const response = await resetEmailTemplate(
        selectedTemplate.event,
        selectedLocale
      )
      if (!response.success) throw new Error(response.message)
      toast.success(t('Default email template restored'))
      await templatesQuery.refetch()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to restore default email template')
      )
    } finally {
      setResetting(false)
    }
  }

  return (
    <SettingsSection title={t('Email settings')}>
      <EmailAlertSettings />

      <div className='space-y-5 border-t pt-6'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
          <div>
            <h3 className='text-base font-semibold'>{t('Email templates')}</h3>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t(
                'Customize notification subjects and HTML by event and language.'
              )}
            </p>
          </div>
          <div className='flex flex-wrap gap-2'>
            <Button
              type='button'
              variant='outline'
              disabled={previewing || !selectedTemplate}
              onClick={refreshPreview}
            >
              <Eye className='size-4' />
              {t('Preview / Refresh')}
            </Button>
            <Button
              type='button'
              variant='outline'
              disabled={resetting || !selectedTemplate?.is_custom}
              onClick={restoreDefault}
            >
              <RotateCcw className='size-4' />
              {t('Restore official template')}
            </Button>
            <Button
              type='button'
              disabled={saving || !selectedTemplate}
              onClick={saveTemplate}
            >
              <Save className='size-4' />
              {t('Save template')}
            </Button>
          </div>
        </div>

        <div className='grid gap-4 lg:grid-cols-2'>
          <div className='space-y-1.5'>
            <Label htmlFor='email-template-event'>{t('Event')}</Label>
            <NativeSelect
              id='email-template-event'
              className='w-full'
              value={selectedEvent}
              onChange={(event) => setSelectedEvent(event.target.value)}
            >
              {events.map((template) => (
                <NativeSelectOption key={template.event} value={template.event}>
                  {t(template.label)}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='email-template-locale'>{t('Language')}</Label>
            <NativeSelect
              id='email-template-locale'
              className='w-full'
              value={selectedLocale}
              onChange={(event) =>
                setSelectedLocale(event.target.value as EmailTemplateLocale)
              }
            >
              <NativeSelectOption value='zh'>{t('Chinese')}</NativeSelectOption>
              <NativeSelectOption value='en'>{t('English')}</NativeSelectOption>
            </NativeSelect>
          </div>
        </div>

        {selectedTemplate ? (
          <>
            <div className='border-primary/20 bg-primary/5 rounded-lg border px-4 py-4'>
              <div className='flex flex-wrap items-center gap-2'>
                <h4 className='text-sm font-semibold'>
                  {t(selectedTemplate.label)}
                </h4>
                <Badge variant='outline'>{t(selectedTemplate.category)}</Badge>
                <Badge variant='secondary'>{t('Transactional email')}</Badge>
                {selectedTemplate.is_custom && (
                  <Badge variant='outline'>{t('Custom')}</Badge>
                )}
              </div>
              <p className='text-muted-foreground mt-2 text-sm'>
                {t(selectedTemplate.description)}
              </p>
            </div>

            <div className='grid min-h-[560px] gap-5 xl:grid-cols-2'>
              <div className='grid min-w-0 content-start gap-4'>
                <div className='space-y-1.5'>
                  <Label htmlFor='email-template-subject'>
                    {t('Email subject')}
                  </Label>
                  <Input
                    ref={subjectRef}
                    id='email-template-subject'
                    maxLength={255}
                    value={subject}
                    onFocus={recordActiveField('subject')}
                    onChange={(event) => setSubject(event.target.value)}
                  />
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='email-template-content'>
                    {t('HTML template')}
                  </Label>
                  <Textarea
                    ref={contentRef}
                    id='email-template-content'
                    className='min-h-[390px] resize-y font-mono text-xs leading-5'
                    value={content}
                    onFocus={recordActiveField('content')}
                    onChange={(event) => setContent(event.target.value)}
                  />
                </div>
              </div>

              <div className='min-w-0 overflow-hidden rounded-lg border'>
                <div className='border-b px-4 py-3'>
                  <div className='text-sm font-semibold'>
                    {t('Live preview')}
                  </div>
                  <div className='text-muted-foreground mt-1 truncate text-xs'>
                    {preview.subject}
                  </div>
                </div>
                <div className='bg-muted/25 min-h-[500px] p-3'>
                  <div className='overflow-hidden rounded-md border bg-white text-black'>
                    <HtmlContent content={preview.content} variant='isolated' />
                  </div>
                </div>
              </div>
            </div>

            <div className='space-y-3 border-t pt-5'>
              <div>
                <h4 className='text-sm font-medium'>
                  {t('Available variables')}
                </h4>
                <p className='text-muted-foreground mt-1 text-xs'>
                  {t('Click a variable to insert it at the current cursor.')}
                </p>
              </div>
              <div className='grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'>
                {selectedTemplate.placeholders.map((placeholder) => (
                  <Button
                    key={placeholder}
                    type='button'
                    variant='outline'
                    className='h-auto min-w-0 justify-start px-3 py-2 text-left'
                    onClick={() => insertPlaceholder(placeholder)}
                  >
                    <span className='min-w-0'>
                      <span className='block truncate text-xs font-medium'>
                        {t(PLACEHOLDER_LABELS[placeholder] ?? placeholder)}
                      </span>
                      <code className='text-muted-foreground block truncate text-[11px]'>
                        {`{{${placeholder}}}`}
                      </code>
                    </span>
                  </Button>
                ))}
              </div>
            </div>
          </>
        ) : (
          <div className='text-muted-foreground flex min-h-80 items-center justify-center rounded-lg border text-sm'>
            {templatesQuery.isLoading
              ? t('Loading...')
              : t('No email templates available')}
          </div>
        )}
      </div>
    </SettingsSection>
  )
}
