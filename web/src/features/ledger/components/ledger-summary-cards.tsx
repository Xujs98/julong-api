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
import {
  ChartNoAxesCombined,
  CircleDollarSign,
  Gauge,
  Landmark,
  ReceiptText,
  Users,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { formatQuota } from '@/lib/format'

import type { LedgerSummary } from '../types'

interface LedgerSummaryCardsProps {
  summary?: LedgerSummary
  loading: boolean
  canEditEstimateRatio: boolean
  onEditEstimateRatio: () => void
}

interface SummaryCardProps {
  title: string
  value: string
  icon: LucideIcon
  details: Array<{ label: string; value: string }>
  loading: boolean
  actionLabel?: string
  onAction?: () => void
}

function SummaryCard(props: SummaryCardProps) {
  const Icon = props.icon
  return (
    <Card
      className={
        props.onAction
          ? 'hover:bg-muted/30 rounded-lg shadow-xs transition-colors'
          : 'rounded-lg shadow-xs'
      }
    >
      <CardHeader className='pb-2'>
        {props.onAction ? (
          <button
            type='button'
            className='focus-visible:ring-ring flex w-full items-center justify-between gap-2 rounded-sm text-left outline-none focus-visible:ring-2'
            onClick={props.onAction}
            aria-label={props.actionLabel}
          >
            <span className='text-sm font-medium'>{props.title}</span>
            <Icon className='text-muted-foreground size-4' aria-hidden='true' />
          </button>
        ) : (
          <div className='flex items-center justify-between gap-2'>
            <CardTitle className='text-sm font-medium'>{props.title}</CardTitle>
            <Icon className='text-muted-foreground size-4' aria-hidden='true' />
          </div>
        )}
      </CardHeader>
      <CardContent>
        {props.loading ? (
          <div className='space-y-3'>
            <Skeleton className='h-7 w-2/3' />
            <Skeleton className='h-4 w-full' />
          </div>
        ) : (
          <>
            <div className='text-2xl font-semibold tabular-nums'>
              {props.value}
            </div>
            <dl className='text-muted-foreground mt-3 space-y-1 text-xs'>
              {props.details.map((detail) => (
                <div
                  key={detail.label}
                  className='flex items-center justify-between gap-3'
                >
                  <dt>{detail.label}</dt>
                  <dd className='text-foreground text-right font-medium tabular-nums'>
                    {detail.value}
                  </dd>
                </div>
              ))}
            </dl>
          </>
        )}
      </CardContent>
    </Card>
  )
}

function formatRatio(value: string | null | undefined): string {
  if (value == null) return '-'
  return `${Number(value).toFixed(4)}x`
}

export function LedgerSummaryCards(props: LedgerSummaryCardsProps) {
  const { t } = useTranslation()
  const summary = props.summary
  const cards: SummaryCardProps[] = [
    {
      title: t('Current total user quota'),
      value: formatQuota(summary?.user_quota.real ?? 0),
      icon: Users,
      details: [
        {
          label: t('Estimated consumption quota'),
          value: formatQuota(summary?.user_quota.estimated ?? 0),
        },
        {
          label: t('Estimate ratio'),
          value: formatRatio(summary?.estimate_ratio),
        },
        {
          label: t('Included users'),
          value: String(summary?.included_user_count ?? 0),
        },
      ],
      loading: props.loading,
      actionLabel: props.canEditEstimateRatio
        ? t('Configure estimate ratio')
        : undefined,
      onAction: props.canEditEstimateRatio
        ? props.onEditEstimateRatio
        : undefined,
    },
    {
      title: t('User consumption in selected period'),
      value: formatQuota(summary?.usage_quota.real ?? 0),
      icon: ChartNoAxesCombined,
      details: [
        {
          label: t('Estimated consumption quota'),
          value: formatQuota(summary?.usage_quota.estimated ?? 0),
        },
        {
          label: t('Real quota'),
          value: formatQuota(summary?.usage_quota.real ?? 0),
        },
      ],
      loading: props.loading,
    },
    {
      title: t('Daily operating cost'),
      value: formatBillingCurrencyFromUSD(
        Number(summary?.daily_operating_cost ?? 0)
      ),
      icon: CircleDollarSign,
      details: [
        {
          label: t('Ledger entries'),
          value: String(summary?.ledger_entry_count ?? 0),
        },
        { label: t('Cost formula'), value: t('Cost price × quantity') },
      ],
      loading: props.loading,
    },
    {
      title: t('Total operating cost'),
      value: formatBillingCurrencyFromUSD(
        Number(summary?.total_operating_cost ?? 0)
      ),
      icon: ReceiptText,
      details: [
        { label: t('Days'), value: String(summary?.days ?? 1) },
        { label: t('Cost formula'), value: t('Sum of all ledger costs') },
      ],
      loading: props.loading,
    },
    {
      title: t('Operational estimated quota'),
      value: formatQuota(summary?.operational_quota.real ?? 0),
      icon: Landmark,
      details: [
        {
          label: t('Real quota'),
          value: formatQuota(summary?.operational_quota.real ?? 0),
        },
        {
          label: t('Cost ratio'),
          value: formatRatio(summary?.operational_quota.cost_ratio),
        },
      ],
      loading: props.loading,
    },
    {
      title: t('Average cost ratios'),
      value: formatRatio(summary?.operational_quota.cost_ratio),
      icon: Gauge,
      details: [
        { label: 'Plus', value: formatRatio(summary?.cost_ratios.plus) },
        { label: 'Pro', value: formatRatio(summary?.cost_ratios.pro) },
        { label: 'K12', value: formatRatio(summary?.cost_ratios.k12) },
      ],
      loading: props.loading,
    },
  ]

  return (
    <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-3'>
      {cards.map((card) => (
        <SummaryCard key={card.title} {...card} />
      ))}
    </div>
  )
}
