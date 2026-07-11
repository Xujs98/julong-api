import { useNavigate } from '@tanstack/react-router'
import { AlertTriangle, Send } from 'lucide-react'
import type React from 'react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { submitErrorReport } from './api'

function getInitialSearchValue(key: string, fallback: string) {
  if (typeof window === 'undefined') return fallback
  return new URLSearchParams(window.location.search).get(key) || fallback
}

export function SubmitErrorReport() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [message, setMessage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const pageUrl = useMemo(
    () =>
      getInitialSearchValue(
        'url',
        typeof window === 'undefined' ? '' : window.location.href
      ),
    []
  )
  const errorStatus = useMemo(() => {
    const status = Number(getInitialSearchValue('status', '500'))
    return Number.isFinite(status) && status > 0 ? status : 500
  }, [])

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!message.trim()) {
      toast.error(t('Please describe what happened.'))
      return
    }

    setIsSubmitting(true)
    try {
      const result = await submitErrorReport({
        title,
        message,
        page_url: pageUrl,
        error_status: errorStatus,
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to submit error report.'))
        return
      }
      toast.success(t('Error report submitted.'))
      navigate({ to: '/' })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className='bg-background flex min-h-svh items-center justify-center p-4'>
      <Card className='w-full max-w-2xl rounded-lg'>
        <CardHeader>
          <div className='bg-destructive/10 text-destructive mb-2 flex size-10 items-center justify-center rounded-md'>
            <AlertTriangle className='size-5' />
          </div>
          <CardTitle>{t('Submit error report')}</CardTitle>
          <CardDescription>
            {t('Send the error details to the site administrators.')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className='space-y-4' onSubmit={handleSubmit}>
            <div className='grid gap-2'>
              <Label htmlFor='error-status'>{t('Error status')}</Label>
              <Input id='error-status' value={errorStatus} disabled />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='page-url'>{t('Error page')}</Label>
              <Input id='page-url' value={pageUrl} disabled />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='error-title'>{t('Title')}</Label>
              <Input
                id='error-title'
                value={title}
                maxLength={200}
                placeholder={t('Briefly summarize the problem')}
                onChange={(event) => setTitle(event.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='error-message'>{t('Error details')}</Label>
              <Textarea
                id='error-message'
                value={message}
                required
                maxLength={5000}
                rows={7}
                placeholder={t(
                  'Describe what you were doing before this error appeared.'
                )}
                onChange={(event) => setMessage(event.target.value)}
              />
            </div>
            <div className='flex flex-col-reverse gap-2 sm:flex-row sm:justify-end'>
              <Button
                type='button'
                variant='outline'
                onClick={() => navigate({ to: '/' })}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={isSubmitting}>
                <Send className='size-4' />
                {isSubmitting ? t('Submitting...') : t('Submit')}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
