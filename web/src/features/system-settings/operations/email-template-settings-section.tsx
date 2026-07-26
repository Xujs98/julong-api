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
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { HtmlContent } from '@/components/html-content'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

import { SettingsSection } from '../components/settings-section'
import {
  listEmailTemplates,
  previewEmailTemplate,
  resetEmailTemplate,
  updateEmailTemplate,
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
}

type ActiveField = 'subject' | 'content'

export function EmailTemplateSettingsSection() {
  const { t } = useTranslation()
  const templatesQuery = useQuery({
    queryKey: ['email-templates'],
    queryFn: listEmailTemplates,
  })
  const [selectedEvent, setSelectedEvent] = useState(
    DEFAULT_EMAIL_TEMPLATE_EVENT
  )
  const [subject, setSubject] = useState('')
  const [content, setContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [previewing, setPreviewing] = useState(false)
  const [preview, setPreview] = useState<EmailTemplatePreview | null>(null)
  const activeField = useRef<ActiveField>('content')
  const subjectRef = useRef<HTMLInputElement>(null)
  const contentRef = useRef<HTMLTextAreaElement>(null)

  const templates = templatesQuery.data?.data ?? []
  const selectedTemplate = templates.find(
    (template) => template.event === selectedEvent
  )

  useEffect(() => {
    if (!selectedTemplate) return
    setSubject(selectedTemplate.subject)
    setContent(selectedTemplate.content)
  }, [selectedTemplate])

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

  const showPreview = async () => {
    if (!selectedTemplate) return
    setPreviewing(true)
    try {
      const response = await previewEmailTemplate(
        selectedTemplate.event,
        subject,
        content
      )
      if (!response.success || !response.data) throw new Error(response.message)
      setPreview(response.data)
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
      const response = await resetEmailTemplate(selectedTemplate.event)
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

  let editorContent: ReactNode
  if (templatesQuery.isLoading) {
    editorContent = (
      <div className='text-muted-foreground flex min-h-80 items-center justify-center text-sm'>
        {t('Loading...')}
      </div>
    )
  } else if (!selectedTemplate) {
    editorContent = (
      <div className='text-muted-foreground flex min-h-80 items-center justify-center text-sm'>
        {t('No email templates available')}
      </div>
    )
  } else {
    editorContent = (
      <div className='space-y-6'>
        <div className='flex flex-col gap-3 border-b pb-5 sm:flex-row sm:items-start sm:justify-between'>
          <div className='min-w-0'>
            <h3 className='text-base font-semibold'>
              {t(selectedTemplate.label)}
            </h3>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t(selectedTemplate.description)}
            </p>
          </div>
          <div className='flex shrink-0 flex-wrap gap-2'>
            <Button
              type='button'
              variant='outline'
              disabled={previewing}
              onClick={showPreview}
            >
              <Eye className='size-4' />
              {t('Preview')}
            </Button>
            <Button
              type='button'
              variant='outline'
              disabled={resetting || !selectedTemplate.is_custom}
              onClick={restoreDefault}
            >
              <RotateCcw className='size-4' />
              {t('Restore defaults')}
            </Button>
            <Button type='button' disabled={saving} onClick={saveTemplate}>
              <Save className='size-4' />
              {t('Save')}
            </Button>
          </div>
        </div>

        <div className='space-y-2'>
          <Label htmlFor='email-template-subject'>{t('Email subject')}</Label>
          <Input
            ref={subjectRef}
            id='email-template-subject'
            maxLength={255}
            value={subject}
            onFocus={recordActiveField('subject')}
            onChange={(event) => setSubject(event.target.value)}
          />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='email-template-content'>{t('HTML content')}</Label>
          <Textarea
            ref={contentRef}
            id='email-template-content'
            className='min-h-80 resize-y font-mono text-xs leading-5'
            value={content}
            onFocus={recordActiveField('content')}
            onChange={(event) => setContent(event.target.value)}
          />
        </div>

        <div className='space-y-3 border-t pt-5'>
          <div>
            <h4 className='text-sm font-medium'>{t('Available variables')}</h4>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t('Click a variable to insert it at the current cursor.')}
            </p>
          </div>
          <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-3'>
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
      </div>
    )
  }

  return (
    <SettingsSection title={t('Email templates')}>
      <div className='grid min-h-[620px] overflow-hidden rounded-lg border lg:grid-cols-[250px_minmax(0,1fr)]'>
        <div className='bg-muted/20 border-b p-2 lg:border-r lg:border-b-0'>
          <div className='grid gap-1 sm:grid-cols-3 lg:grid-cols-1'>
            {templates.map((template) => (
              <button
                key={template.event}
                type='button'
                className={cn(
                  'hover:bg-muted flex min-w-0 items-start justify-between gap-2 rounded-md px-3 py-2.5 text-left transition-colors',
                  template.event === selectedEvent &&
                    'bg-background text-foreground shadow-sm'
                )}
                onClick={() => setSelectedEvent(template.event)}
              >
                <span className='min-w-0'>
                  <span className='block truncate text-sm font-medium'>
                    {t(template.label)}
                  </span>
                  <span className='text-muted-foreground mt-0.5 block truncate text-xs'>
                    {template.event}
                  </span>
                </span>
                {template.is_custom && (
                  <Badge variant='outline' className='shrink-0'>
                    {t('Custom')}
                  </Badge>
                )}
              </button>
            ))}
          </div>
        </div>
        <div className='min-w-0 p-4 sm:p-6'>{editorContent}</div>
      </div>

      <Dialog
        open={preview !== null}
        onOpenChange={(open) => !open && setPreview(null)}
        title={t('Email template preview')}
        description={preview?.subject}
        contentClassName='sm:max-w-3xl'
        contentHeight='min(72vh, 760px)'
      >
        {preview && (
          <div className='overflow-hidden rounded-lg border bg-white p-3 text-black'>
            <HtmlContent content={preview.content} variant='isolated' />
          </div>
        )}
      </Dialog>
    </SettingsSection>
  )
}
