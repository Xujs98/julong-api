import { useQuery } from '@tanstack/react-query'
import { Bug, ChevronLeft, ChevronRight, Eye } from 'lucide-react'
import type React from 'react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestampToDate } from '@/lib/format'

import { getErrorReport, getErrorReports } from './api'
import type { ErrorReport } from './types'

const PAGE_SIZE = 20

function getReportTitle(report: ErrorReport, fallback: string) {
  return report.title || report.message.slice(0, 60) || fallback
}

function DetailRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className='grid gap-1'>
      <div className='text-muted-foreground text-xs'>{label}</div>
      <div className='break-words text-sm'>{children || '-'}</div>
    </div>
  )
}

export function ErrorReports() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<number | null>(null)

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['error-reports', page],
    queryFn: async () => {
      const result = await getErrorReports({ p: page, page_size: PAGE_SIZE })
      if (!result.success) {
        throw new Error(result.message || t('Failed to load error reports.'))
      }
      return result.data || { items: [], total: 0, page, page_size: PAGE_SIZE }
    },
    placeholderData: (previousData) => previousData,
  })

  const detailQuery = useQuery({
    queryKey: ['error-report', selectedId],
    enabled: selectedId !== null,
    queryFn: async () => {
      const result = await getErrorReport(selectedId!)
      if (!result.success || !result.data) {
        throw new Error(result.message || t('Failed to load error report.'))
      }
      return result.data
    },
  })

  const reports = data?.items || []
  const total = data?.total || 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const selectedReport = detailQuery.data

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Error Reports')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='border-border bg-card overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Time')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Title')}</TableHead>
                  <TableHead>{t('User')}</TableHead>
                  <TableHead>{t('Error page')}</TableHead>
                  <TableHead className='w-24 text-right'>
                    {t('Actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow>
                    <TableCell colSpan={6} className='h-24 text-center'>
                      {t('Loading...')}
                    </TableCell>
                  </TableRow>
                ) : reports.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className='h-40 text-center'>
                      <div className='text-muted-foreground flex flex-col items-center gap-2'>
                        <Bug className='size-8' />
                        <span>{t('No error reports found.')}</span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  reports.map((report) => (
                    <TableRow key={report.id}>
                      <TableCell className='whitespace-nowrap'>
                        {formatTimestampToDate(report.created_at)}
                      </TableCell>
                      <TableCell>
                        <Badge variant='outline'>{report.error_status}</Badge>
                      </TableCell>
                      <TableCell className='max-w-[260px] truncate'>
                        {getReportTitle(report, t('Untitled report'))}
                      </TableCell>
                      <TableCell className='whitespace-nowrap'>
                        {report.username || report.user_id || t('Anonymous')}
                      </TableCell>
                      <TableCell className='max-w-[320px] truncate'>
                        {report.page_url || '-'}
                      </TableCell>
                      <TableCell className='text-right'>
                        <Button
                          size='sm'
                          variant='outline'
                          onClick={() => setSelectedId(report.id)}
                        >
                          <Eye className='size-4' />
                          {t('View')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
            <div className='bg-muted/30 flex flex-col gap-3 border-t p-3 sm:flex-row sm:items-center sm:justify-between'>
              <div className='text-muted-foreground text-sm'>
                {t('Total')} {total}
                {isFetching ? ` · ${t('Refreshing...')}` : ''}
              </div>
              <div className='flex items-center justify-end gap-2'>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page <= 1}
                  onClick={() => setPage((value) => Math.max(1, value - 1))}
                >
                  <ChevronLeft className='size-4' />
                  {t('Previous')}
                </Button>
                <span className='text-muted-foreground min-w-16 text-center text-sm'>
                  {page} / {totalPages}
                </span>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page >= totalPages}
                  onClick={() =>
                    setPage((value) => Math.min(totalPages, value + 1))
                  }
                >
                  {t('Next')}
                  <ChevronRight className='size-4' />
                </Button>
              </div>
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Dialog
        open={selectedId !== null}
        onOpenChange={(open) => !open && setSelectedId(null)}
      >
        <DialogContent className='max-h-[85vh] overflow-y-auto sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>{t('Error report details')}</DialogTitle>
            <DialogDescription>
              {t('Submitted error information from the 500 error page.')}
            </DialogDescription>
          </DialogHeader>
          {!selectedReport ? (
            <div className='text-muted-foreground py-8 text-center'>
              {t('Loading...')}
            </div>
          ) : (
            <div className='grid gap-4'>
              <div className='grid gap-4 sm:grid-cols-2'>
                <DetailRow label={t('Time')}>
                  {formatTimestampToDate(selectedReport.created_at)}
                </DetailRow>
                <DetailRow label={t('Status')}>
                  {selectedReport.error_status}
                </DetailRow>
                <DetailRow label={t('User')}>
                  {selectedReport.username ||
                    selectedReport.user_id ||
                    t('Anonymous')}
                </DetailRow>
                <DetailRow label={t('IP Address')}>
                  {selectedReport.ip}
                </DetailRow>
              </div>
              <DetailRow label={t('Title')}>
                {selectedReport.title || t('Untitled report')}
              </DetailRow>
              <DetailRow label={t('Error page')}>
                {selectedReport.page_url}
              </DetailRow>
              <DetailRow label={t('Error details')}>
                <pre className='bg-muted max-h-60 overflow-auto rounded-md p-3 whitespace-pre-wrap'>
                  {selectedReport.message}
                </pre>
              </DetailRow>
              <DetailRow label={t('Browser')}>
                {selectedReport.user_agent}
              </DetailRow>
              {selectedReport.stack ? (
                <DetailRow label={t('Stack trace')}>
                  <pre className='bg-muted max-h-60 overflow-auto rounded-md p-3 whitespace-pre-wrap'>
                    {selectedReport.stack}
                  </pre>
                </DetailRow>
              ) : null}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
