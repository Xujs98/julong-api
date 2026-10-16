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
import { useNavigate, getRouteApi } from '@tanstack/react-router'
import { Eye, Image as ImageIcon, Music, Trash2, Video } from 'lucide-react'
import { useState, useCallback, type ReactNode } from 'react'
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
import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { TableCell, TableRow } from '@/components/ui/table'
import { api } from '@/lib/api'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'
import { formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { getDefaultTimeRange } from '../lib/utils'
import { CompactDateTimeRangePicker } from './compact-date-time-range-picker'
import {
  LogsFilterField,
  LogsFilterInput,
  LogsFilterToolbar,
} from './logs-filter-toolbar'
import { useLogsViewScope, useUsageLogsContext } from './usage-logs-provider'

const route = getRouteApi('/_authenticated/usage-logs/$section')

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
  const search = route.useSearch()
  const { isAdminView, visibilityKey } = useLogsViewScope('assets')
  return (
    <MediaAssetsTableContent
      key={JSON.stringify([
        search.startTime,
        search.endTime,
        search.username,
        isAdminView,
        visibilityKey,
      ])}
    />
  )
}

function MediaAssetsTableContent() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const searchParams = route.useSearch()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const role = useAuthStore((state) => state.auth.user?.role ?? ROLE.USER)
  const { isAdminView: isAdmin, visibilityKey } = useLogsViewScope('assets')
  const { sensitiveVisible, setSelectedUserId, setUserInfoDialogOpen } =
    useUsageLogsContext()
  const canDelete = role >= ROLE.ADMIN
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 50 })
  const [selected, setSelected] = useState<MediaAsset | null>(null)
  const [pendingDelete, setPendingDelete] = useState<MediaAsset | null>(null)

  const [defaultRange] = useState(getDefaultTimeRange)
  const startTime = searchParams.startTime ?? defaultRange.start.getTime()
  const endTime = searchParams.endTime ?? defaultRange.end.getTime()
  const username = isAdmin ? searchParams.username || '' : ''

  const [filters, setFilters] = useState<{
    startTime?: Date
    endTime?: Date
    username?: string
  }>(() => ({
    startTime: new Date(startTime),
    endTime: new Date(endTime),
    username,
  }))

  const handleFilterChange = useCallback(
    (field: keyof typeof filters, value: Date | string | undefined) => {
      setFilters((prev) => ({ ...prev, [field]: value }))
    },
    []
  )

  const handleApply = useCallback(() => {
    void navigate({
      to: '/usage-logs/$section',
      params: { section: 'assets' },
      search: {
        startTime: filters.startTime?.getTime(),
        endTime: filters.endTime?.getTime(),
        username: isAdmin ? filters.username?.trim() || undefined : undefined,
      },
    })
    setPagination((previous) => ({ ...previous, pageIndex: 0 }))
  }, [filters, isAdmin, navigate])

  const handleReset = useCallback(() => {
    const { start, end } = getDefaultTimeRange()
    setFilters({ startTime: start, endTime: end, username: '' })
    void navigate({
      to: '/usage-logs/$section',
      params: { section: 'assets' },
      search: {
        startTime: start.getTime(),
        endTime: end.getTime(),
      },
    })
    setPagination((previous) => ({ ...previous, pageIndex: 0 }))
  }, [navigate])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') handleApply()
    },
    [handleApply]
  )

  const assetsQuery = useQuery({
    queryKey: [
      'media-assets',
      userId,
      visibilityKey,
      isAdmin,
      pagination,
      startTime,
      endTime,
      username,
    ],
    queryFn: async () => {
      const response = await api.get<MediaAssetListResponse>(
        '/api/media-assets',
        {
          params: {
            page: pagination.pageIndex + 1,
            page_size: pagination.pageSize,
            start_time: Math.floor(startTime / 1000),
            end_time: Math.floor(endTime / 1000),
            username: username || undefined,
            view_scope: isAdmin ? 'all' : 'self',
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
    ...(isAdmin
      ? [
          {
            id: 'user',
            header: t('User'),
            cell: (asset: MediaAsset) => {
              const displayName = asset.username || String(asset.user_id || '?')
              return (
                <button
                  type='button'
                  className='flex items-center gap-1.5 text-left'
                  onClick={(e) => {
                    e.stopPropagation()
                    setSelectedUserId(asset.user_id)
                    setUserInfoDialogOpen(true)
                  }}
                >
                  <Avatar className='ring-border/60 size-6 ring-1 max-sm:hidden'>
                    <AvatarFallback
                      className={cn(
                        'text-[11px] font-semibold',
                        !sensitiveVisible && 'bg-muted text-muted-foreground'
                      )}
                      style={
                        sensitiveVisible
                          ? getUserAvatarStyle(displayName)
                          : undefined
                      }
                    >
                      {sensitiveVisible
                        ? getUserAvatarFallback(displayName)
                        : '•'}
                    </AvatarFallback>
                  </Avatar>
                  <span className='text-muted-foreground truncate text-sm hover:underline'>
                    {sensitiveVisible ? displayName : '••••'}
                  </span>
                </button>
              )
            },
            cellClassName: 'px-4 py-3',
          },
        ]
      : []),
    {
      id: 'media',
      header: t('Media'),
      cell: (asset) => (
        <span
          className={`inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium ${mediaStyles[asset.media_type]}`}
        >
          <MediaIcon type={asset.media_type} />
          {asset.source_type === 'base64'
            ? t('Base64 media')
            : t('Public')} ·{' '}
          {t(mediaLabelKeys[asset.media_type])}
        </span>
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

  const { table } = useDataTable({
    data: assets,
    columns: columns.map((column) => ({
      id: column.id,
      header: String(column.header),
      accessorFn: (asset: MediaAsset) => asset.asset_id,
      enableHiding: column.id !== 'details' && column.id !== 'actions',
    })),
    totalCount: assetsQuery.data?.total ?? 0,
    manualPagination: true,
    pagination,
    onPaginationChange: setPagination,
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
  })
  const visibleColumns = columns.filter((column) =>
    table.getColumn(column.id)?.getIsVisible()
  )

  let tableContent: ReactNode
  if (assetsQuery.isError) {
    tableContent = (
      <ErrorState
        title={t('Failed to load media assets')}
        description={assetsQuery.error.message}
        onRetry={() => void assetsQuery.refetch()}
      />
    )
  } else if (assetsQuery.isPending) {
    tableContent = <LoadingState />
  } else {
    tableContent = (
      <StaticDataTable
        className='min-h-0 flex-1 overflow-auto'
        tableClassName='min-w-[820px] text-sm'
        columns={visibleColumns}
        data={assets}
        getRowKey={(asset) => asset.asset_id}
        emptyContent={t('No media assets')}
        headerRowClassName='bg-muted/50'
        renderRow={(asset, index) => (
          <TableRow key={asset.asset_id}>
            {visibleColumns.map((column) => (
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
    )
  }

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <LogsFilterToolbar
        table={table}
        primaryFilters={
          <>
            <LogsFilterField wide>
              <CompactDateTimeRangePicker
                start={filters.startTime}
                end={filters.endTime}
                onChange={({ start, end }) => {
                  handleFilterChange('startTime', start)
                  handleFilterChange('endTime', end)
                }}
              />
            </LogsFilterField>
            {isAdmin ? (
              <LogsFilterField>
                <LogsFilterInput
                  aria-label={t('Filter by username')}
                  placeholder={t('Filter by username')}
                  value={filters.username || ''}
                  onChange={(e) =>
                    handleFilterChange('username', e.target.value)
                  }
                  onKeyDown={handleKeyDown}
                />
              </LogsFilterField>
            ) : null}
          </>
        }
        mobilePinnedFilters={
          <LogsFilterField wide>
            <CompactDateTimeRangePicker
              start={filters.startTime}
              end={filters.endTime}
              onChange={({ start, end }) => {
                handleFilterChange('startTime', start)
                handleFilterChange('endTime', end)
              }}
            />
          </LogsFilterField>
        }
        mobileFilters={
          isAdmin ? (
            <LogsFilterField>
              <LogsFilterInput
                aria-label={t('Filter by username')}
                placeholder={t('Filter by username')}
                value={filters.username || ''}
                onChange={(e) => handleFilterChange('username', e.target.value)}
                onKeyDown={handleKeyDown}
              />
            </LogsFilterField>
          ) : null
        }
        mobileFilterCount={filters.username ? 1 : 0}
        hasActiveFilters={
          !!filters.username ||
          filters.startTime?.getTime() !== defaultRange.start.getTime() ||
          filters.endTime?.getTime() !== defaultRange.end.getTime()
        }
        onSearch={handleApply}
        searchLoading={assetsQuery.isFetching}
        onReset={handleReset}
      />
      {tableContent}
      {(assetsQuery.data?.total ?? 0) > pagination.pageSize ? (
        <DataTablePagination table={table} compact />
      ) : null}
      <Dialog
        open={selected !== null}
        onOpenChange={(open) => !open && setSelected(null)}
        title={t('Media asset details')}
        contentClassName='sm:max-w-3xl'
        bodyClassName='space-y-4'
        footer={
          selected ? (
            <div className='flex w-full min-w-0 items-center gap-2'>
              <code className='bg-muted min-w-0 flex-1 truncate rounded p-2 text-xs'>
                {selected.url}
              </code>
              <CopyButton value={selected.url} />
            </div>
          ) : null
        }
      >
        {selected ? (
          <div className='bg-muted/30 flex min-w-0 items-center justify-center overflow-hidden rounded border p-2'>
            {selected.media_type === 'image' ? (
              <img
                src={selected.url}
                alt={t('Media asset preview')}
                className='max-h-[55dvh] max-w-full object-contain'
              />
            ) : null}
            {selected.media_type === 'video' ? (
              <video
                src={selected.url}
                controls
                className='max-h-[55dvh] max-w-full object-contain'
              />
            ) : null}
            {selected.media_type === 'audio' ? (
              <audio src={selected.url} controls className='w-full min-w-0' />
            ) : null}
          </div>
        ) : null}
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
