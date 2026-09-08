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
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

export function InvoicePagination(props: {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const pageCount = Math.max(1, Math.ceil(props.total / props.pageSize))
  const start = props.total === 0 ? 0 : (props.page - 1) * props.pageSize + 1
  const end = Math.min(props.page * props.pageSize, props.total)

  return (
    <div className='flex min-h-12 flex-wrap items-center justify-between gap-3 border-t px-4 py-2.5'>
      <p className='text-muted-foreground text-sm'>
        {t('Showing {{start}} to {{end}} of {{total}} results', {
          start,
          end,
          total: props.total,
        })}
      </p>
      <div className='flex items-center gap-1'>
        <Button
          variant='outline'
          size='icon-sm'
          aria-label={t('Previous page')}
          disabled={props.page <= 1}
          onClick={() => props.onPageChange(props.page - 1)}
        >
          <ChevronLeft />
        </Button>
        <span className='border-primary text-primary flex h-8 min-w-10 items-center justify-center border px-2 text-sm'>
          {props.page}
        </span>
        <Button
          variant='outline'
          size='icon-sm'
          aria-label={t('Next page')}
          disabled={props.page >= pageCount}
          onClick={() => props.onPageChange(props.page + 1)}
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  )
}
