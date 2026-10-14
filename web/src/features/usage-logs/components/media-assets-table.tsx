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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Eye, Image as ImageIcon, Music, Trash2, Video } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { CopyButton } from '@/components/copy-button'
import {
  DataTablePagination,
  StaticDataTable,
  useDataTable,
  type StaticDataTableColumn,
} from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { TableCell, TableRow } from '@/components/ui/table'
import { api } from '@/lib/api'
import { formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

type MediaAsset = {
  asset_id: string
  user_id: number
  username?: string
  source_type: 'base64' | 'public'
  media_type: 'image' | 'video' | 'audio'
  mime_type?: string
  url: string
  size: number
  created_at: number
  expires_at: number
}

type MediaAssetListResponse = {
  success: boolean
  message?: string
  data?: {
    items: MediaAsset[]
    total: number
  }
}

const mediaStyles = {
  image: 'text-sky-700 bg-sky-100 dark:text-sky-300 dark:bg-sky-950/40',
  video:
    'text-violet-700 bg-violet-100 dark:text-violet-300 dark:bg-violet-950/40',
  audio: 'text-amber-700 bg-amber-100 dark:text-amber-300 dark:bg-amber-950/40',
} as const

const mediaLabelKeys = {
  image: 'Image',
  video: 'Video',
  audio: 'Audio',
} as const

function MediaIcon(props: { type: MediaAsset['media_type'] }) {
  if (props.type === 'image') {
    return <ImageIcon className='size-4' aria-hidden='true' />
  }
  if (props.type === 'video') {
    return <Video className='size-4' aria-hidden='true' />
  }
  return <Music className='size-4' aria-hidden='true' />
}

export function MediaAssetsTable() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const role = useAuthStore((state) => state.auth.user?.role ?? ROLE.USER)
  const canDelete = role >= ROLE.ADMIN
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 50 })
  const [selected, setSelected] = useState<MediaAsset | null>(null)
  const [pendingDelete, setPendingDelete] = useState<MediaAsset | null>(null)
  const assetsQuery = useQuery({
    queryKey: ['media-assets', pagination],
    queryFn: async () => {
      const response = await api.get<MediaAssetListResponse>(
        '/api/media-assets',
        {
          params: {
            page: pagination.pageIndex + 1,
            page_size: pagination.pageSize,
          },
        }
      )
      if (!response.data.success || !response.data.data) {
        throw new Error(response.data.message || 'Failed to load media assets')
      }
      return response.data.data
    },
    retry: false,
  })
  const assets = assetsQuery.data?.items ?? []
  const { table } = useDataTable({
    data: assets,
    columns: [],
    totalCount: assetsQuery.data?.total ?? 0,
    manualPagination: true,
    pagination,
    onPaginationChange: setPagination,
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
  })
  const deleteMutation = useMutation({
    mutationFn: async (assetId: string) =>
      api.delete(`/api/media-assets/${assetId}`),
    onSuccess: () => {
      toast.success(t('Media asset deleted'))
      setPendingDelete(null)
      void queryClient.invalidateQueries({ queryKey: ['media-assets'] })
    },
    onError: (error) =>
      handleServerError(error, t('Failed to delete media asset')),
  })

  const columns: StaticDataTableColumn<MediaAsset>[] = [
    {
      id: 'media',
      header: t('Media'),
      cell: (asset) => (
        <div className='space-y-1'>
          <span
            className={`inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium ${mediaStyles[asset.media_type]}`}
          >
            <MediaIcon type={asset.media_type} />
            {asset.source_type === 'base64'
              ? t('Base64 media')
              : t('Public')} ·{' '}
            {t(mediaLabelKeys[asset.media_type])}
          </span>
          {asset.username ? (
            <div className='text-muted-foreground text-xs'>
              {asset.username}
            </div>
          ) : null}
        </div>
      ),
      cellClassName: 'px-4 py-3',
    },
    {
      id: 'submitted',
      header: t('Submitted at'),
      cell: (asset) => formatTimestampToDate(asset.created_at, 'seconds'),
      cellClassName: 'px-4 py-3 text-xs',
    },
    {
      id: 'expires',
      header: t('Expires at'),
      cell: (asset) =>
        asset.expires_at
          ? formatTimestampToDate(asset.expires_at, 'seconds')
          : t('Never'),
      cellClassName: 'px-4 py-3 text-xs',
    },
    {
      id: 'details',
      header: t('Details'),
      cell: (asset) => (
        <Button size='sm' variant='outline' onClick={() => setSelected(asset)}>
          <Eye className='size-4' aria-hidden='true' />
          {t('Preview')}
        </Button>
      ),
      cellClassName: 'px-4 py-3',
    },
    ...(canDelete
      ? [
          {
            id: 'actions',
            header: t('Actions'),
            cell: (asset: MediaAsset) => (
              <Button
                size='sm'
                variant='ghost'
                onClick={() => setPendingDelete(asset)}
                disabled={deleteMutation.isPending}
              >
                <Trash2 className='size-4' aria-hidden='true' />
                {t('Delete')}
              </Button>
            ),
            cellClassName: 'px-4 py-3',
          },
        ]
      : []),
  ]

  if (assetsQuery.isError) {
    return (
      <ErrorState
        title={t('Failed to load media assets')}
        description={assetsQuery.error.message}
        onRetry={() => void assetsQuery.refetch()}
      />
    )
  }
  if (assetsQuery.isPending) {
    return <LoadingState />
  }

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <StaticDataTable
        className='min-h-0 flex-1 overflow-auto'
        tableClassName='min-w-[820px] text-sm'
        columns={columns}
        data={assets}
        getRowKey={(asset) => asset.asset_id}
        emptyContent={t('No media assets')}
        headerRowClassName='bg-muted/50'
        renderRow={(asset, index) => (
          <TableRow key={asset.asset_id}>
            {columns.map((column) => (
              <TableCell
                key={column.id}
                className={
                  typeof column.cellClassName === 'function'
                    ? column.cellClassName(asset, index)
                    : column.cellClassName
                }
              >
                {column.cell?.(asset, index)}
              </TableCell>
            ))}
          </TableRow>
        )}
      />
      {assetsQuery.data.total > pagination.pageSize ? (
        <DataTablePagination table={table} compact />
      ) : null}
      <Dialog
        open={selected !== null}
        onOpenChange={(open) => !open && setSelected(null)}
      >
        <DialogContent className='max-w-3xl'>
          <DialogHeader>
            <DialogTitle>{t('Media asset details')}</DialogTitle>
          </DialogHeader>
          {selected ? (
            <div className='space-y-4'>
              <div className='bg-muted/30 flex min-h-48 items-center justify-center rounded border p-2'>
                {selected.media_type === 'image' ? (
                  <img
                    src={selected.url}
                    alt={t('Media asset preview')}
                    className='max-h-[60vh] max-w-full object-contain'
                  />
                ) : null}
                {selected.media_type === 'video' ? (
                  <video
                    src={selected.url}
                    controls
                    className='max-h-[60vh] max-w-full'
                  />
                ) : null}
                {selected.media_type === 'audio' ? (
                  <audio src={selected.url} controls />
                ) : null}
              </div>
              <div className='flex items-center gap-2'>
                <code className='bg-muted min-w-0 flex-1 truncate rounded p-2 text-xs'>
                  {selected.url}
                </code>
                <CopyButton value={selected.url} />
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => !open && setPendingDelete(null)}
        title={t('Delete media asset')}
        desc={t('This will permanently remove the selected media asset.')}
        destructive
        confirmText={t('Delete')}
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (pendingDelete) deleteMutation.mutate(pendingDelete.asset_id)
        }}
      />
    </div>
  )
}
