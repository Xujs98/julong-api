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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Clipboard,
  Download,
  Eye,
  Loader2,
  MessageSquareText,
  ShieldAlert,
  Trash2,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatTimestampToDate } from '@/lib/format'

import {
  deleteUserRequestContentLogs,
  getUserRequestContentLog,
  getUserRequestContentLogs,
  updateUserRequestContentLogging,
} from '../api'
import { extractRequestConversation } from '../lib/request-content'
import type {
  UserRequestContentLog,
  UserRequestContentLogDetail,
  UserRequestContentStatus,
} from '../types'

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function statusVariant(status: UserRequestContentStatus): StatusVariant {
  if (status === 'success') return 'success'
  if (status === 'error') return 'danger'
  return 'warning'
}

function statusLabel(status: UserRequestContentStatus): string {
  if (status === 'success') return 'Completed'
  if (status === 'error') return 'Error'
  return 'In progress'
}

function downloadJSON(content: unknown, requestId: string) {
  const json = JSON.stringify(content, null, 2)
  const url = URL.createObjectURL(
    new Blob([json], { type: 'application/json;charset=utf-8' })
  )
  const link = document.createElement('a')
  link.href = url
  link.download = `request-content-${requestId}.json`
  link.click()
  URL.revokeObjectURL(url)
}

function RequestContentDetailDialog(props: {
  userId: number
  log: UserRequestContentLog | null
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: [
      'admin-user-request-content-detail',
      props.userId,
      props.log?.id,
    ],
    enabled: Boolean(props.log),
    queryFn: async () => {
      if (!props.log) throw new Error('Missing request log')
      const result = await getUserRequestContentLog(props.userId, props.log.id)
      if (!result.success || !result.data) {
        throw new Error(result.message || 'Load failed')
      }
      return result.data
    },
  })
  const detail = query.data as UserRequestContentLogDetail | undefined
  const conversation = useMemo(
    () => extractRequestConversation(detail?.content),
    [detail?.content]
  )
  const formattedJSON = useMemo(
    () => (detail ? JSON.stringify(detail.content, null, 2) : ''),
    [detail]
  )

  const copyJSON = async () => {
    if (!formattedJSON) return
    try {
      await navigator.clipboard.writeText(formattedJSON)
      toast.success(t('Copied'))
    } catch {
      toast.error(t('Operation failed'))
    }
  }

  let body = (
    <div className='space-y-3'>
      <Skeleton className='h-24 w-full' />
      <Skeleton className='h-52 w-full' />
    </div>
  )
  if (query.isError) {
    body = (
      <div className='flex h-48 flex-col items-center justify-center gap-3 text-center'>
        <span className='text-destructive text-sm'>
          {t('Failed to load request content')}
        </span>
        <Button variant='outline' size='sm' onClick={() => query.refetch()}>
          {t('Retry')}
        </Button>
      </div>
    )
  } else if (detail) {
    body = (
      <Tabs defaultValue='conversation'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <TabsList>
            <TabsTrigger value='conversation'>{t('Conversation')}</TabsTrigger>
            <TabsTrigger value='json'>{t('JSON data')}</TabsTrigger>
          </TabsList>
          <div className='flex gap-2'>
            <Button
              type='button'
              variant='outline'
              size='icon-sm'
              title={t('Copy JSON')}
              aria-label={t('Copy JSON')}
              onClick={copyJSON}
            >
              <Clipboard />
            </Button>
            <Button
              type='button'
              variant='outline'
              size='icon-sm'
              title={t('Download JSON')}
              aria-label={t('Download JSON')}
              onClick={() =>
                downloadJSON(detail.content, detail.log.request_id)
              }
            >
              <Download />
            </Button>
          </div>
        </div>
        {detail.log.error_message && (
          <div className='border-destructive/40 bg-destructive/5 text-destructive mt-3 rounded-md border p-3 text-xs break-words'>
            {detail.log.error_message}
          </div>
        )}
        <TabsContent value='conversation' className='mt-3'>
          <ScrollArea className='h-[56vh] pr-3'>
            {conversation.length === 0 ? (
              <div className='text-muted-foreground flex h-40 items-center justify-center text-sm'>
                {t(
                  'No structured conversation found. View the JSON data instead.'
                )}
              </div>
            ) : (
              <div className='space-y-3'>
                {conversation.map((item) => (
                  <div
                    key={item.id}
                    className='overflow-hidden rounded-md border'
                  >
                    <div className='bg-muted/50 border-b px-3 py-2 text-xs font-medium uppercase'>
                      {item.role}
                    </div>
                    <pre className='max-h-80 overflow-auto p-3 font-sans text-sm leading-6 break-words whitespace-pre-wrap'>
                      {item.text}
                    </pre>
                  </div>
                ))}
              </div>
            )}
          </ScrollArea>
        </TabsContent>
        <TabsContent value='json' className='mt-3'>
          <ScrollArea className='bg-muted/20 h-[56vh] rounded-md border'>
            <pre className='p-4 font-mono text-xs leading-5 break-words whitespace-pre-wrap'>
              {formattedJSON}
            </pre>
          </ScrollArea>
        </TabsContent>
      </Tabs>
    )
  }

  return (
    <Dialog
      open={Boolean(props.log)}
      onOpenChange={props.onOpenChange}
      title={t('Request context and prompt')}
      description={props.log?.request_id}
      contentClassName='sm:max-w-5xl'
    >
      {body}
    </Dialog>
  )
}

export function UserRequestContentPanel(props: { userId?: number }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedLog, setSelectedLog] = useState<UserRequestContentLog | null>(
    null
  )
  const [updating, setUpdating] = useState(false)
  const [clearOpen, setClearOpen] = useState(false)
  const [clearing, setClearing] = useState(false)
  const query = useQuery({
    queryKey: ['admin-user-request-content', props.userId],
    enabled: Boolean(props.userId),
    queryFn: async () => {
      if (!props.userId) throw new Error('Missing user')
      const result = await getUserRequestContentLogs(props.userId)
      if (!result.success || !result.data) {
        throw new Error(result.message || 'Load failed')
      }
      return result.data
    },
  })

  const updateEnabled = async (enabled: boolean) => {
    if (!props.userId) return
    setUpdating(true)
    try {
      const result = await updateUserRequestContentLogging(
        props.userId,
        enabled
      )
      if (!result.success) {
        toast.error(result.message || t('Operation failed'))
        return
      }
      await query.refetch()
      toast.success(
        enabled
          ? t('Request content recording enabled')
          : t('Request content recording disabled')
      )
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setUpdating(false)
    }
  }

  const clearLogs = async () => {
    if (!props.userId) return
    setClearing(true)
    try {
      const result = await deleteUserRequestContentLogs(props.userId)
      if (!result.success) {
        toast.error(result.message || t('Operation failed'))
        return
      }
      setClearOpen(false)
      setSelectedLog(null)
      await queryClient.invalidateQueries({
        queryKey: ['admin-user-request-content', props.userId],
      })
      toast.success(t('Request content records cleared'))
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setClearing(false)
    }
  }

  if (query.isLoading) {
    return (
      <div className='space-y-3'>
        <Skeleton className='h-20 w-full' />
        <Skeleton className='h-16 w-full' />
        <Skeleton className='h-16 w-full' />
      </div>
    )
  }

  if (query.isError || !query.data) {
    return (
      <div className='flex h-40 flex-col items-center justify-center gap-3 rounded-md border'>
        <span className='text-destructive text-sm'>
          {t('Failed to load request content records')}
        </span>
        <Button variant='outline' size='sm' onClick={() => query.refetch()}>
          {t('Retry')}
        </Button>
      </div>
    )
  }

  return (
    <>
      <div className='space-y-4'>
        <div className='flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-center sm:justify-between'>
          <div className='flex min-w-0 items-start gap-3'>
            <div className='bg-muted mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md'>
              <ShieldAlert className='size-4' />
            </div>
            <div className='min-w-0'>
              <div className='text-sm font-medium'>
                {t('Record request context and prompts')}
              </div>
              <div className='text-muted-foreground mt-1 text-xs leading-5'>
                {t(
                  'Only new requests after enabling are recorded. Embedded Base64 media and known credential fields are removed.'
                )}
              </div>
            </div>
          </div>
          <Switch
            checked={query.data.enabled}
            disabled={updating}
            aria-label={t('Record request context and prompts')}
            onCheckedChange={updateEnabled}
          />
        </div>

        <div className='flex items-center justify-between gap-3'>
          <div className='text-muted-foreground text-xs'>
            {t('Up to {{count}} recent requests are retained per user.', {
              count: query.data.max_items,
            })}
          </div>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={query.data.items.length === 0}
            onClick={() => setClearOpen(true)}
          >
            <Trash2 />
            {t('Clear records')}
          </Button>
        </div>

        {query.data.items.length === 0 ? (
          <div className='text-muted-foreground flex h-40 flex-col items-center justify-center gap-2 rounded-md border text-sm'>
            <MessageSquareText className='size-5' />
            <span>{t('No request content records')}</span>
          </div>
        ) : (
          <div className='overflow-hidden rounded-md border'>
            {query.data.items.map((log) => (
              <div
                key={log.id}
                className='grid gap-3 border-b p-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center'
              >
                <div className='min-w-0'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <span className='text-sm font-medium break-all'>
                      {log.model_name || '-'}
                    </span>
                    <StatusBadge
                      label={t(statusLabel(log.status))}
                      variant={statusVariant(log.status)}
                      copyable={false}
                      size='sm'
                    />
                    {log.truncated && (
                      <StatusBadge
                        label={t('Truncated')}
                        variant='warning'
                        copyable={false}
                        size='sm'
                      />
                    )}
                  </div>
                  <div className='text-muted-foreground mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                    <span>{formatTimestampToDate(log.created_at)}</span>
                    <code>{log.request_path}</code>
                    <span>{formatBytes(log.captured_size)}</span>
                  </div>
                  <div className='text-muted-foreground mt-1 truncate font-mono text-[11px]'>
                    {log.request_id}
                  </div>
                </div>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => setSelectedLog(log)}
                >
                  <Eye />
                  {t('View')}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {props.userId && (
        <RequestContentDetailDialog
          userId={props.userId}
          log={selectedLog}
          onOpenChange={(open) => !open && setSelectedLog(null)}
        />
      )}
      <ConfirmDialog
        open={clearOpen}
        onOpenChange={setClearOpen}
        title={t('Clear request content records?')}
        desc={t(
          'All captured contexts and prompts for this user will be permanently deleted.'
        )}
        confirmText={
          clearing ? <Loader2 className='animate-spin' /> : t('Clear records')
        }
        destructive
        isLoading={clearing}
        handleConfirm={clearLogs}
      />
    </>
  )
}
