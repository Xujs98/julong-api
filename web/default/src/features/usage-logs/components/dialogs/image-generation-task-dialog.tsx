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
import { Download, FileJson, RefreshCw } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'

import { getImageGenerationTask } from '../../image-generation-task-api'
import type { ImageGenerationLog } from '../../types'

interface Props {
  log: ImageGenerationLog
  open: boolean
  onOpenChange: (open: boolean) => void
  pollingIntervalMs: number
}

const statusVariants: Record<ImageGenerationLog['status'], StatusVariant> = {
  pending: 'warning',
  processing: 'info',
  success: 'success',
  failed: 'danger',
}

function statusLabel(
  status: ImageGenerationLog['status'],
  t: (key: string) => string
) {
  if (status === 'pending') return t('Pending')
  if (status === 'processing') return t('In Progress')
  if (status === 'success') return t('Success')
  return t('Failed')
}

export function ImageGenerationTaskDialog({
  log,
  open,
  onOpenChange,
  pollingIntervalMs,
}: Props) {
  const { t } = useTranslation()
  const taskQuery = useQuery({
    queryKey: ['image-generation-task', log.id],
    queryFn: () => getImageGenerationTask(log.id),
    enabled: open && Boolean(log.task_id),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'pending' || status === 'processing'
        ? pollingIntervalMs
        : false
    },
  })
  const formattedJson = useMemo(
    () => (taskQuery.data ? JSON.stringify(taskQuery.data, null, 2) : ''),
    [taskQuery.data]
  )

  const downloadJson = () => {
    if (!formattedJson) return
    const url = URL.createObjectURL(
      new Blob([formattedJson], { type: 'application/json;charset=utf-8' })
    )
    const link = document.createElement('a')
    link.href = url
    link.download = `${log.task_id}.json`
    link.click()
    URL.revokeObjectURL(url)
  }

  let content = (
    <ScrollArea className='bg-muted/30 mt-3 max-h-[65vh] rounded-md border'>
      <pre className='min-w-0 p-4 font-mono text-xs leading-5 break-words whitespace-pre-wrap'>
        {formattedJson || t('No data')}
      </pre>
    </ScrollArea>
  )
  if (taskQuery.isLoading) {
    content = (
      <div className='space-y-2 py-3'>
        <Skeleton className='h-4 w-full' />
        <Skeleton className='h-4 w-5/6' />
        <Skeleton className='h-32 w-full' />
      </div>
    )
  } else if (taskQuery.isError) {
    content = (
      <div className='text-destructive flex min-h-36 items-center justify-center gap-2 text-sm'>
        <FileJson className='size-4' />
        {t('Failed to load image generation task')}
      </div>
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Task ID')}
      description={log.task_id}
      contentClassName='sm:max-w-3xl'
      contentHeight='auto'
    >
      <div className='flex min-h-9 items-center justify-between gap-3 border-b pb-3'>
        {taskQuery.data ? (
          <StatusBadge
            label={statusLabel(taskQuery.data.status, t)}
            variant={statusVariants[taskQuery.data.status]}
            copyable={false}
          />
        ) : (
          <Skeleton className='h-5 w-20' />
        )}
        <div className='flex items-center gap-1'>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            title={t('Refresh')}
            aria-label={t('Refresh')}
            disabled={taskQuery.isFetching}
            onClick={() => void taskQuery.refetch()}
          >
            <RefreshCw
              className={`size-4 ${taskQuery.isFetching ? 'animate-spin' : ''}`}
            />
          </Button>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            title={t('Download JSON')}
            aria-label={t('Download JSON')}
            disabled={!formattedJson}
            onClick={downloadJson}
          >
            <Download className='size-4' />
          </Button>
        </div>
      </div>

      {content}
    </Dialog>
  )
}
