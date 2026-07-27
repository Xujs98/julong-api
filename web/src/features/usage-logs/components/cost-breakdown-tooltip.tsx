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
import type { ReactElement } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatLogQuota } from '@/lib/format'

import type { BillingRevenueItem } from '../lib/billing-revenue'

interface CostBreakdownTooltipProps {
  trigger: ReactElement
  subscriptionQuota: number | null
  revenueItems: BillingRevenueItem[]
}

export function CostBreakdownTooltip(props: CostBreakdownTooltipProps) {
  const { t } = useTranslation()

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger render={props.trigger} />
        <TooltipContent>
          <div className='flex min-w-52 flex-col gap-1.5'>
            {props.subscriptionQuota != null && (
              <div className='flex items-center justify-between gap-4'>
                <span>{t('Deducted by subscription')}</span>
                <span className='font-mono tabular-nums'>
                  {formatLogQuota(props.subscriptionQuota)}
                </span>
              </div>
            )}
            {props.revenueItems.map((item) => (
              <div
                key={item.key}
                className='flex items-center justify-between gap-4'
              >
                <span>{t(item.labelKey)}</span>
                <span className='font-mono tabular-nums'>
                  {formatLogQuota(item.quota)}
                </span>
              </div>
            ))}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
