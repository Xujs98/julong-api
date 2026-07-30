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
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import dayjs from '@/lib/dayjs'
import { formatQuota, formatTimestamp } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import {
  createLedgerEntry,
  deleteLedgerEntries,
  getLedgerEntries,
  getLedgerSummary,
  updateLedgerSettings,
  updateLedgerEntry,
} from './api'
import { LedgerDeleteDialog } from './components/ledger-delete-dialog'
import { LedgerEntryDialog } from './components/ledger-entry-dialog'
import { LedgerEstimateRatioDialog } from './components/ledger-estimate-ratio-dialog'
import { LedgerExportMenu } from './components/ledger-export-menu'
import { LedgerSummaryCards } from './components/ledger-summary-cards'
import type { LedgerEntry, LedgerListParams, LedgerMutation } from './types'

const PAGE_SIZE = 20

export function Ledger() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.auth.user)
  const [page, setPage] = useState(1)
  const [range, setRange] = useState(() => ({
    start: dayjs().startOf('day').toDate(),
    end: dayjs().endOf('day').toDate(),
  }))
  const [selectedIds, setSelectedIds] = useState<Set<number>>(() => new Set())
  const [formOpen, setFormOpen] = useState(false)
  const [editingEntry, setEditingEntry] = useState<LedgerEntry>()
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [estimateRatioOpen, setEstimateRatioOpen] = useState(false)

  const canWrite = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.LEDGER,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const canDelete = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.LEDGER,
    ADMIN_PERMISSION_ACTIONS.DELETE
  )
  const params = useMemo<LedgerListParams>(
    () => ({
      p: page,
      page_size: PAGE_SIZE,
      start_timestamp: Math.floor(range.start.getTime() / 1000),
      end_timestamp: Math.floor(range.end.getTime() / 1000),
    }),
    [page, range.end, range.start]
  )
  const summaryParams = useMemo(
    () => ({
      start_timestamp: params.start_timestamp,
      end_timestamp: params.end_timestamp,
    }),
    [params.end_timestamp, params.start_timestamp]
  )

  const entriesQuery = useQuery({
    queryKey: ['ledger', 'entries', params],
    queryFn: async () => {
      const response = await getLedgerEntries(params)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load ledger'))
      }
      return response.data
    },
    placeholderData: (previous) => previous,
  })
  const summaryQuery = useQuery({
    queryKey: ['ledger', 'summary', summaryParams],
    queryFn: async () => {
      const response = await getLedgerSummary(summaryParams)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load ledger summary'))
      }
      return response.data
    },
  })
  const saveMutation = useMutation({
    mutationFn: async (data: LedgerMutation) => {
      const response = editingEntry
        ? await updateLedgerEntry(editingEntry.id, data)
        : await createLedgerEntry(data)
      if (!response.success) {
        throw new Error(response.message || t('Failed to save ledger entry'))
      }
      return response
    },
    onSuccess: async () => {
      toast.success(t('Ledger entry saved'))
      setFormOpen(false)
      setEditingEntry(undefined)
      await queryClient.invalidateQueries({ queryKey: ['ledger'] })
    },
  })
  const deleteMutation = useMutation({
    mutationFn: async (ids: number[]) => {
      const response = await deleteLedgerEntries(ids)
      if (!response.success) {
        throw new Error(
          response.message || t('Failed to delete ledger entries')
        )
      }
      return response.data ?? 0
    },
    onSuccess: async (deleted) => {
      toast.success(t('Deleted {{count}} ledger entries', { count: deleted }))
      setDeleteOpen(false)
      setSelectedIds(new Set())
      await queryClient.invalidateQueries({ queryKey: ['ledger'] })
    },
  })
  const estimateRatioMutation = useMutation({
    mutationFn: async (estimateRatio: number) => {
      const response = await updateLedgerSettings(estimateRatio)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to save estimate ratio'))
      }
      return response.data
    },
    onSuccess: async () => {
      toast.success(t('Estimate ratio saved'))
      setEstimateRatioOpen(false)
      await queryClient.invalidateQueries({
        queryKey: ['ledger', 'summary'],
      })
    },
  })

  const entries = entriesQuery.data?.items ?? []
  const total = entriesQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const currentIds = entries.map((entry) => entry.id)
  const selectedCurrentCount = currentIds.filter((id) =>
    selectedIds.has(id)
  ).length
  const allCurrentSelected =
    entries.length > 0 && selectedCurrentCount === entries.length
  const someCurrentSelected = selectedCurrentCount > 0 && !allCurrentSelected

  const changeRange = (next: { start?: Date; end?: Date }) => {
    if (!next.start || !next.end) return
    setRange({ start: next.start, end: next.end })
    setPage(1)
    setSelectedIds(new Set())
  }
  const toggleAllCurrent = (checked: boolean) => {
    setSelectedIds((current) => {
      const next = new Set(current)
      currentIds.forEach((id) => {
        if (checked) next.add(id)
        else next.delete(id)
      })
      return next
    })
  }
  const toggleRow = (id: number, checked: boolean) => {
    setSelectedIds((current) => {
      const next = new Set(current)
      if (checked) next.add(id)
      else next.delete(id)
      return next
    })
  }
  const startCreate = () => {
    setEditingEntry(undefined)
    setFormOpen(true)
  }
  const startEdit = (entry: LedgerEntry) => {
    setEditingEntry(entry)
    setFormOpen(true)
  }
  const startSingleDelete = (id: number) => {
    setSelectedIds(new Set([id]))
    setDeleteOpen(true)
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Ledger')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <LedgerExportMenu params={params} />
          {canWrite && (
            <Button size='sm' onClick={startCreate}>
              <Plus data-icon='inline-start' />
              {t('Add ledger entry')}
            </Button>
          )}
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-5 pb-6'>
            <section aria-labelledby='ledger-date-filter'>
              <div className='flex flex-col gap-2 border-b pb-4 sm:flex-row sm:items-center sm:justify-between'>
                <div>
                  <h3 id='ledger-date-filter' className='text-sm font-semibold'>
                    {t('Date query')}
                  </h3>
                  <p className='text-muted-foreground mt-0.5 text-xs'>
                    {t('Filter ledger records and summaries by ledger date.')}
                  </p>
                </div>
                <CompactDateTimeRangePicker
                  start={range.start}
                  end={range.end}
                  onChange={changeRange}
                  className='sm:w-[360px]'
                />
              </div>
            </section>

            <section aria-labelledby='ledger-summary'>
              <h3 id='ledger-summary' className='mb-3 text-sm font-semibold'>
                {t('Summary')}
              </h3>
              <LedgerSummaryCards
                summary={summaryQuery.data}
                loading={summaryQuery.isLoading}
                canEditEstimateRatio={canWrite}
                onEditEstimateRatio={() => setEstimateRatioOpen(true)}
              />
            </section>

            <section aria-labelledby='ledger-list'>
              <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                <div>
                  <h3 id='ledger-list' className='text-sm font-semibold'>
                    {t('Ledger entries')}
                  </h3>
                  <p className='text-muted-foreground mt-0.5 text-xs'>
                    {t('{{count}} records', { count: total })}
                  </p>
                </div>
                {canDelete && selectedIds.size > 0 && (
                  <Button
                    size='sm'
                    variant='destructive'
                    onClick={() => setDeleteOpen(true)}
                  >
                    <Trash2 data-icon='inline-start' />
                    {t('Delete selected ({{count}})', {
                      count: selectedIds.size,
                    })}
                  </Button>
                )}
              </div>

              <div className='overflow-hidden rounded-lg border'>
                {entriesQuery.isLoading && !entriesQuery.data && (
                  <div className='space-y-3 p-4'>
                    {Array.from({ length: 5 }, (_, index) => (
                      <Skeleton key={index} className='h-11 w-full' />
                    ))}
                  </div>
                )}
                {!entriesQuery.isLoading && entries.length === 0 && (
                  <Empty className='py-14'>
                    <EmptyHeader>
                      <EmptyTitle>{t('No ledger entries')}</EmptyTitle>
                      <EmptyDescription>
                        {t('No ledger entries match the selected date range.')}
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                )}
                {entries.length > 0 && (
                  <Table className='min-w-[980px]'>
                    <TableHeader>
                      <TableRow>
                        {canDelete && (
                          <TableHead className='w-10'>
                            <Checkbox
                              checked={allCurrentSelected}
                              indeterminate={someCurrentSelected}
                              aria-checked={
                                someCurrentSelected
                                  ? 'mixed'
                                  : allCurrentSelected
                              }
                              onCheckedChange={(checked) =>
                                toggleAllCurrent(checked === true)
                              }
                              aria-label={t('Select current page')}
                            />
                          </TableHead>
                        )}
                        <TableHead>{t('Ledger date and time')}</TableHead>
                        <TableHead>{t('Platform')}</TableHead>
                        <TableHead>{t('Account information')}</TableHead>
                        <TableHead>{t('Email')}</TableHead>
                        <TableHead>{t('Type')}</TableHead>
                        <TableHead className='text-right'>
                          {t('Quota')}
                        </TableHead>
                        <TableHead className='text-right'>
                          {t('Cost price')}
                        </TableHead>
                        <TableHead className='text-right'>
                          {t('Quantity')}
                        </TableHead>
                        {(canWrite || canDelete) && (
                          <TableHead className='w-24 text-right'>
                            {t('Actions')}
                          </TableHead>
                        )}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {entries.map((entry) => (
                        <TableRow
                          key={entry.id}
                          data-state={
                            selectedIds.has(entry.id) ? 'selected' : undefined
                          }
                        >
                          {canDelete && (
                            <TableCell>
                              <Checkbox
                                checked={selectedIds.has(entry.id)}
                                onCheckedChange={(checked) =>
                                  toggleRow(entry.id, checked === true)
                                }
                                aria-label={t('Select ledger entry {{id}}', {
                                  id: entry.id,
                                })}
                              />
                            </TableCell>
                          )}
                          <TableCell>
                            {formatTimestamp(entry.occurred_at)}
                          </TableCell>
                          <TableCell className='font-medium'>
                            {entry.platform}
                          </TableCell>
                          <TableCell className='max-w-56 truncate'>
                            {entry.account}
                          </TableCell>
                          <TableCell className='max-w-56 truncate'>
                            {entry.email || '-'}
                          </TableCell>
                          <TableCell className='uppercase'>
                            {entry.type}
                          </TableCell>
                          <TableCell className='text-right'>
                            {formatQuota(entry.quota)}
                          </TableCell>
                          <TableCell className='text-right'>
                            {formatBillingCurrencyFromUSD(
                              Number(entry.cost_price)
                            )}
                          </TableCell>
                          <TableCell className='text-right'>
                            {entry.quantity}
                          </TableCell>
                          {(canWrite || canDelete) && (
                            <TableCell>
                              <div className='flex justify-end gap-1'>
                                {canWrite && (
                                  <Tooltip>
                                    <TooltipTrigger
                                      render={
                                        <Button
                                          type='button'
                                          size='icon'
                                          variant='ghost'
                                          onClick={() => startEdit(entry)}
                                          aria-label={t('Edit ledger entry')}
                                        />
                                      }
                                    >
                                      <Pencil />
                                    </TooltipTrigger>
                                    <TooltipContent>{t('Edit')}</TooltipContent>
                                  </Tooltip>
                                )}
                                {canDelete && (
                                  <Tooltip>
                                    <TooltipTrigger
                                      render={
                                        <Button
                                          type='button'
                                          size='icon'
                                          variant='ghost'
                                          onClick={() =>
                                            startSingleDelete(entry.id)
                                          }
                                          aria-label={t('Delete ledger entry')}
                                        />
                                      }
                                    >
                                      <Trash2 />
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      {t('Delete')}
                                    </TooltipContent>
                                  </Tooltip>
                                )}
                              </div>
                            </TableCell>
                          )}
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </div>

              <div className='mt-3 flex items-center justify-between gap-3'>
                <span className='text-muted-foreground text-xs'>
                  {t('Page {{page}} of {{total}}', { page, total: totalPages })}
                </span>
                <div className='flex gap-2'>
                  <Button
                    size='sm'
                    variant='outline'
                    disabled={page <= 1}
                    onClick={() =>
                      setPage((current) => Math.max(1, current - 1))
                    }
                  >
                    {t('Previous')}
                  </Button>
                  <Button
                    size='sm'
                    variant='outline'
                    disabled={page >= totalPages}
                    onClick={() =>
                      setPage((current) => Math.min(totalPages, current + 1))
                    }
                  >
                    {t('Next')}
                  </Button>
                </div>
              </div>
            </section>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <LedgerEntryDialog
        open={formOpen}
        entry={editingEntry}
        submitting={saveMutation.isPending}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditingEntry(undefined)
        }}
        onSubmit={async (data) => {
          try {
            await saveMutation.mutateAsync(data)
          } catch (error) {
            toast.error(
              error instanceof Error
                ? error.message
                : t('Failed to save ledger entry')
            )
          }
        }}
      />
      <LedgerDeleteDialog
        open={deleteOpen}
        count={selectedIds.size}
        deleting={deleteMutation.isPending}
        onOpenChange={setDeleteOpen}
        onConfirm={async () => {
          try {
            await deleteMutation.mutateAsync([...selectedIds])
          } catch (error) {
            toast.error(
              error instanceof Error
                ? error.message
                : t('Failed to delete ledger entries')
            )
          }
        }}
      />
      <LedgerEstimateRatioDialog
        open={estimateRatioOpen}
        estimateRatio={Number(summaryQuery.data?.estimate_ratio ?? 1)}
        submitting={estimateRatioMutation.isPending}
        onOpenChange={setEstimateRatioOpen}
        onSubmit={async (estimateRatio) => {
          try {
            await estimateRatioMutation.mutateAsync(estimateRatio)
          } catch (error) {
            toast.error(
              error instanceof Error
                ? error.message
                : t('Failed to save estimate ratio')
            )
          }
        }}
      />
    </>
  )
}
