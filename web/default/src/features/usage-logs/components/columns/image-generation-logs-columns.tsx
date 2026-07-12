import type { ColumnDef } from '@tanstack/react-table'
import { Images } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { formatQuota, formatTimestampToDate } from '@/lib/format'

import type { ImageGenerationLog } from '../../types'
import { ImageGenerationPreviewDialog } from '../dialogs/image-generation-preview-dialog'
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
