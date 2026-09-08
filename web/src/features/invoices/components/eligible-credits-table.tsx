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
import { FileText, RotateCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestamp } from '@/lib/format'

import { getInvoiceSourceLabel } from '../labels'
import type { InvoiceCredit } from '../types'
import { InvoicePagination } from './invoice-pagination'

function formatInvoiceAmount(amount: number, unit: string) {
  return `${unit} ${amount.toFixed(2)}`
}

export function EligibleCreditsTable(props: {
  credits: InvoiceCredit[]
  total: number
  page: number
  pageSize: number
  unit: string
  minimumAmount: number
  enabled: boolean
  selected: Map<string, InvoiceCredit>
  applying: boolean
  onToggle: (credit: InvoiceCredit, checked: boolean) => void
  onTogglePage: (checked: boolean) => void
  onClear: () => void
  onApply: () => void
  onRefresh: () => void
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const now = Math.floor(Date.now() / 1000)
  const selectableCredits = props.credits.filter(
    (credit) => (credit.frozen_until ?? 0) <= now
  )
  const selectedAmount = [...props.selected.values()].reduce(
    (sum, credit) => sum + credit.amount,
    0
  )
  const currentPageSelected = selectableCredits.filter((credit) =>
    props.selected.has(credit.request_id)
  ).length
  const allPageSelected =
    selectableCredits.length > 0 &&
    currentPageSelected === selectableCredits.length
  const thresholdReached = selectedAmount >= props.minimumAmount

  return (
    <Card className='gap-0 py-0'>
      <CardHeader className='flex-row items-start justify-between gap-4 border-b py-4'>
        <div className='min-w-0'>
          <h3 className='text-base font-semibold'>{t('Eligible credits')}</h3>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Select one or more credits. The selected amount must reach {{amount}} {{unit}}.',
              { amount: props.minimumAmount.toFixed(2), unit: props.unit }
            )}
          </p>
        </div>
        <Button
          variant='outline'
          size='icon'
          aria-label={t('Refresh eligible credits')}
          onClick={props.onRefresh}
        >
          <RotateCw />
        </Button>
      </CardHeader>
      {props.selected.size > 0 && (
        <div className='bg-accent/40 flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3'>
          <div className='min-w-0'>
            <p className='font-medium'>
              {t('{{count}} selected', { count: props.selected.size })}
              <span className='text-muted-foreground mx-2'>|</span>
              {t('Total invoice amount: {{amount}}', {
                amount: formatInvoiceAmount(selectedAmount, props.unit),
              })}
            </p>
            {!thresholdReached && (
              <p className='text-warning text-sm'>
                {t('{{amount}} more is required to apply.', {
                  amount: formatInvoiceAmount(
                    props.minimumAmount - selectedAmount,
                    props.unit
                  ),
                })}
              </p>
            )}
          </div>
          <div className='flex items-center gap-2'>
            <Button variant='outline' onClick={props.onClear}>
              {t('Clear selection')}
            </Button>
            <Button
              disabled={!props.enabled || !thresholdReached || props.applying}
              onClick={props.onApply}
            >
              <FileText data-icon='inline-start' />
              {t('Invoice selected credits')}
            </Button>
          </div>
        </div>
      )}
      <CardContent className='p-0'>
        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className='w-12'>
                  <Checkbox
                    aria-label={t('Select current page')}
                    checked={allPageSelected}
                    onCheckedChange={(checked) =>
                      props.onTogglePage(checked === true)
                    }
                  />
                </TableHead>
                <TableHead>{t('Credit source')}</TableHead>
                <TableHead>{t('Eligible amount')}</TableHead>
                <TableHead>{t('Completed at')}</TableHead>
                <TableHead className='text-right'>{t('Status')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {props.credits.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className='h-28 text-center'>
                    <p className='text-muted-foreground'>
                      {t('No eligible credits')}
                    </p>
                  </TableCell>
                </TableRow>
              ) : (
                props.credits.map((credit) => (
                  <TableRow key={credit.request_id}>
                    <TableCell>
                      <Checkbox
                        aria-label={t('Select credit from {{time}}', {
                          time: formatTimestamp(credit.created_at),
                        })}
                        checked={props.selected.has(credit.request_id)}
                        disabled={(credit.frozen_until ?? 0) > now}
                        onCheckedChange={(checked) =>
                          props.onToggle(credit, checked === true)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      {getInvoiceSourceLabel(t, credit.source)}
                    </TableCell>
                    <TableCell className='font-medium'>
                      {formatInvoiceAmount(credit.amount, props.unit)}
                    </TableCell>
                    <TableCell>{formatTimestamp(credit.created_at)}</TableCell>
                    <TableCell className='text-right'>
                      <span className='bg-muted rounded-full px-2 py-1 text-xs font-medium'>
                        {(credit.frozen_until ?? 0) > now
                          ? t('Frozen until {{time}}', {
                              time: formatTimestamp(credit.frozen_until ?? 0),
                            })
                          : t('Eligible')}
                      </span>
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
