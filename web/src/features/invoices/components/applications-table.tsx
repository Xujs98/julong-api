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
import { Download, Eye, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestamp } from '@/lib/format'

import { getInvoiceStatusLabel } from '../labels'
import type { InvoiceApplication, InvoiceStatus } from '../types'
import { InvoicePagination } from './invoice-pagination'

const statusVariants: Record<InvoiceStatus, string> = {
  pending: 'bg-warning/10 text-warning',
  processing: 'bg-info/10 text-info',
  completed: 'bg-success/10 text-success',
  rejected: 'bg-destructive/10 text-destructive',
  non_reusable: 'bg-muted text-muted-foreground',
  cancelled: 'bg-muted text-muted-foreground',
}

export function ApplicationsTable(props: {
  title: string
  description: string
  applications: InvoiceApplication[]
  total: number
  page: number
  pageSize: number
  admin?: boolean
  onPageChange: (page: number) => void
  onManage?: (application: InvoiceApplication) => void
  onCancel?: (application: InvoiceApplication) => void
  onDownload?: (application: InvoiceApplication) => void
}) {
  const { t } = useTranslation()
  const columnCount = props.admin ? 7 : 6

  return (
    <Card className='gap-0 py-0'>
      <CardHeader className='border-b py-4'>
        <h3 className='text-base font-semibold'>{props.title}</h3>
        <p className='text-muted-foreground text-sm'>{props.description}</p>
      </CardHeader>
      <CardContent className='p-0'>
        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Application')}</TableHead>
                {props.admin && <TableHead>{t('User')}</TableHead>}
                <TableHead>{t('Invoice amount')}</TableHead>
                <TableHead>{t('Email')}</TableHead>
                <TableHead>{t('Applied at')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead className='text-right'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {props.applications.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={columnCount}
                    className='text-muted-foreground h-28 text-center'
                  >
                    {t('No invoice applications')}
                  </TableCell>
                </TableRow>
              ) : (
                props.applications.map((application) => (
                  <TableRow key={application.id}>
                    <TableCell>#{application.id}</TableCell>
                    {props.admin && (
                      <TableCell>
                        {application.username || `#${application.user_id}`}
                      </TableCell>
                    )}
                    <TableCell className='font-medium'>
                      {application.unit} {application.amount.toFixed(2)}
                    </TableCell>
                    <TableCell>{application.email}</TableCell>
                    <TableCell>
                      {formatTimestamp(application.created_at)}
                    </TableCell>
                    <TableCell>
                      <span
                        className={`${statusVariants[application.status]} rounded-full px-2 py-1 text-xs font-medium`}
                      >
                        {getInvoiceStatusLabel(t, application.status)}
                      </span>
                    </TableCell>
                    <TableCell className='text-right'>
                      <div className='flex justify-end gap-2'>
                        {props.admin && (
                          <Button
                            variant='outline'
                            size='icon-sm'
                            aria-label={t('View invoice application {{id}}', {
                              id: application.id,
                            })}
                            onClick={() => props.onManage?.(application)}
                          >
                            <Eye />
                          </Button>
                        )}
                        {!props.admin && application.status === 'pending' && (
                          <Button
                            variant='outline'
                            size='icon-sm'
                            aria-label={t('Cancel invoice application {{id}}', {
                              id: application.id,
                            })}
                            onClick={() => props.onCancel?.(application)}
                          >
                            <X />
                          </Button>
                        )}
                        {application.status === 'completed' &&
                          application.has_attachment && (
                            <Button
                              variant='outline'
                              size='icon-sm'
                              aria-label={t(
                                'Download invoice application {{id}}',
                                { id: application.id }
                              )}
                              onClick={() => props.onDownload?.(application)}
                            >
                              <Download />
                            </Button>
                          )}
                        {!props.admin &&
                          application.status !== 'pending' &&
                          !(
                            application.status === 'completed' &&
                            application.has_attachment
                          ) && <span className='text-muted-foreground'>-</span>}
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
        <InvoicePagination
          page={props.page}
          pageSize={props.pageSize}
          total={props.total}
          onPageChange={props.onPageChange}
        />
      </CardContent>
    </Card>
  )
}
