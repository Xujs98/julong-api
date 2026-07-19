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
import type { ColumnDef } from '@tanstack/react-table'
import { FileJson, Images } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { formatQuota, formatTimestampToDate } from '@/lib/format'

import type { ImageGenerationLog } from '../../types'
import { ImageGenerationPreviewDialog } from '../dialogs/image-generation-preview-dialog'
import { ImageGenerationTaskDialog } from '../dialogs/image-generation-task-dialog'
import { PromptDialog } from '../dialogs/prompt-dialog'
import { createChannelColumn } from './column-helpers'

export function useImageGenerationLogsColumns(
  isAdmin: boolean
): ColumnDef<ImageGenerationLog>[] {
  const { t } = useTranslation()
  const columns: ColumnDef<ImageGenerationLog>[] = [
    {
      accessorKey: 'created_at',
      header: t('Time'),
      cell: ({ row }) => (
        <span className='font-mono text-xs whitespace-nowrap tabular-nums'>
          {formatTimestampToDate(row.original.created_at)}
        </span>
      ),
    },
  ]

  if (isAdmin) {
    columns.push(
      {
        accessorKey: 'username',
        header: t('User'),
        cell: ({ row }) => row.original.username || '-',
      },
      createChannelColumn<ImageGenerationLog>({ headerLabel: t('Channel') })
    )
  }

  columns.push(
    {
      accessorKey: 'task_id',
      header: t('Task ID'),
      cell: function TaskIdCell({ row }) {
        const [open, setOpen] = useState(false)
        const log = row.original
        if (!log.task_id) {
          return <span className='text-muted-foreground text-xs'>-</span>
        }
        return (
          <>
            <button
              type='button'
              className='text-foreground hover:bg-muted inline-flex max-w-[180px] items-center gap-1.5 rounded-md px-1.5 py-1 font-mono text-xs'
              title={t('JSON data')}
              onClick={() => setOpen(true)}
            >
              <FileJson className='size-3.5 shrink-0' />
              <span className='truncate'>{log.task_id}</span>
            </button>
            <ImageGenerationTaskDialog
              log={log}
              open={open}
              onOpenChange={setOpen}
            />
          </>
        )
      },
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      cell: ({ row }) => {
        const status = row.original.status || 'success'
        const config = {
          pending: { label: t('Pending'), variant: 'warning' as const },
          processing: { label: t('In Progress'), variant: 'info' as const },
          success: { label: t('Success'), variant: 'success' as const },
          failed: { label: t('Failed'), variant: 'danger' as const },
        }[status]
        return (
          <StatusBadge
            label={config.label}
            variant={config.variant}
            copyable={false}
            size='sm'
          />
        )
      },
    },
    {
      accessorKey: 'model_name',
      header: t('Model'),
      cell: ({ row }) => (
        <StatusBadge
          label={row.original.model_name || '-'}
          variant='neutral'
          size='sm'
          copyable={false}
        />
      ),
    },
    {
      accessorKey: 'prompt',
      header: t('Prompt'),
      cell: function PromptCell({ row }) {
        const [open, setOpen] = useState(false)
        return (
          <>
            <button
              type='button'
              className='text-muted-foreground max-w-[260px] truncate text-left text-xs hover:underline'
              onClick={() => setOpen(true)}
            >
              {row.original.prompt || '-'}
            </button>
            <PromptDialog
              prompt={row.original.prompt}
              open={open}
              onOpenChange={setOpen}
            />
          </>
        )
      },
    },
    {
      id: 'parameters',
      header: t('Parameters'),
      cell: ({ row }) => (
        <div className='text-muted-foreground flex flex-col text-xs'>
          <span>{row.original.size || '-'}</span>
          <span>{row.original.quality || '-'}</span>
        </div>
      ),
    },
    {
      accessorKey: 'image_count',
      header: t('Image'),
      cell: function ImagesCell({ row }) {
        const [open, setOpen] = useState(false)
        const log = row.original
        if (log.status !== 'success' || log.image_count <= 0) {
          return <span className='text-muted-foreground text-xs'>-</span>
        }
        return (
          <>
            <button
              type='button'
              className='hover:bg-muted inline-flex min-h-9 items-center gap-1.5 rounded-md px-2 text-xs'
              onClick={() => setOpen(true)}
            >
              <Images className='size-4' />
              {t('View')} ({log.image_count})
            </button>
            <ImageGenerationPreviewDialog
              log={log}
              open={open}
              onOpenChange={setOpen}
            />
          </>
        )
      },
    },
    {
      accessorKey: 'quota',
      header: t('Cost'),
      cell: ({ row }) => formatQuota(row.original.quota || 0),
    },
    {
      accessorKey: 'use_time',
      header: t('Duration'),
      cell: ({ row }) => {
        const seconds = Number(row.original.use_time || 0)
        return (
          <span className='font-mono text-xs tabular-nums'>
            {seconds > 0 ? `${seconds}s` : '<1s'}
          </span>
        )
      },
    }
  )

  return columns
}
