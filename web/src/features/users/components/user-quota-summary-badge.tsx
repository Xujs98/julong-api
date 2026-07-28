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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { formatNumber, formatQuota } from '@/lib/format'

import type { UserQuotaSummary } from '../types'

type UserQuotaSummaryBadgeProps = {
  summary?: UserQuotaSummary
  isFetching: boolean
}

export function UserQuotaSummaryBadge(props: UserQuotaSummaryBadgeProps) {
  const { t } = useTranslation()

  return (
    <Badge
      variant='secondary'
      className='h-9 w-full justify-start gap-2 px-3 sm:w-[280px]'
      aria-busy={props.isFetching}
      aria-live='polite'
    >
      <span className='min-w-0 flex-1 truncate'>
        {t('Remaining quota')}:{' '}
        <strong>
          {props.summary ? formatQuota(props.summary.total_quota) : '--'}
        </strong>
      </span>
      <span className='border-border shrink-0 border-l pl-2'>
        {props.summary ? formatNumber(props.summary.user_count) : '--'}{' '}
        {t('Users')}
      </span>
    </Badge>
  )
}
