import { useQuery } from '@tanstack/react-query'
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
import { VChart } from '@visactor/react-vchart'
import {
  BarChart3,
  Coins,
  Hash,
  Layers,
  RotateCcw,
  UsersRound,
  type LucideIcon,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import { useTheme } from '@/context/theme-provider'
import { getGroupQuotaDates } from '@/features/dashboard/api'
import { buildQueryParams, getDefaultDays } from '@/features/dashboard/lib'
import type {
  DashboardFilters,
  GroupAnalyticsMetric,
} from '@/features/dashboard/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'
import { useThemeRadiusPx } from '@/lib/theme-radius'
import { computeTimeRange } from '@/lib/time'
import { VCHART_OPTION } from '@/lib/vchart'

interface GroupAnalysisProps {
  filters: DashboardFilters
}

interface MetricOption {
  value: GroupAnalyticsMetric
  labelKey: string
  icon: LucideIcon
  tone: IconBadgeTone
}

const METRIC_OPTIONS: MetricOption[] = [
  { value: 'quota', labelKey: 'Quota', icon: Coins, tone: 'success' },
  { value: 'count', labelKey: 'Requests', icon: Hash, tone: 'info' },
  { value: 'token_used', labelKey: 'Tokens', icon: Layers, tone: 'chart-4' },
  {
    value: 'user_count',
    labelKey: 'Users',
    icon: UsersRound,
    tone: 'warning',
  },
]

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

export function GroupAnalysis(props: GroupAnalysisProps) {
  const { t, i18n } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { customization } = useThemeCustomization()
  const chartRadius = useThemeRadiusPx(
    '--radius-md',
    `${customization.preset}:${customization.radius}`
  )
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [metric, setMetric] = useState<GroupAnalyticsMetric>('quota')
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  const queryParams = useMemo(() => {
    const timeRange = computeTimeRange(
      getDefaultDays(props.filters.time_granularity),
      props.filters.start_timestamp,
      props.filters.end_timestamp
    )
    return buildQueryParams(timeRange, props.filters)
  }, [props.filters])

  const query = useQuery({
    queryKey: ['dashboard-group-analytics', queryParams],
    queryFn: () => getGroupQuotaDates(queryParams),
  })
  const analytics = query.data?.data
  const rows = useMemo(() => analytics?.items ?? [], [analytics?.items])

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (module) => module.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }
    void updateTheme()
  }, [resolvedTheme])

  const formatMetric = useCallback(
    (value: number, compact = false) => {
      if (metric === 'quota') return formatQuota(value)
      return compact
        ? formatCompactNumber(value, locale)
        : formatNumber(value, locale)
    },
    [locale, metric]
  )

  const selectedMetric =
    METRIC_OPTIONS.find((option) => option.value === metric) ??
    METRIC_OPTIONS[0]
  const chartRows = useMemo(
    () =>
      [...rows]
        .sort((a, b) => b[metric] - a[metric])
        .map((row) => ({
          group: row.group || t('Unknown'),
          value: row[metric],
        })),
    [rows, metric, t]
  )
  const chartSpec = useMemo(
    () => ({
      type: 'bar',
      direction: 'horizontal',
      data: [{ id: 'groupData', values: chartRows }],
      xField: 'value',
      yField: 'group',
      seriesField: 'group',
      bar: { style: { cornerRadius: chartRadius } },
      legends: { visible: false },
      axes: [
        {
          orient: 'bottom',
          label: {
            formatMethod: (value: number) => formatMetric(Number(value), true),
          },
        },
        {
          orient: 'left',
          label: { autoLimit: true, autoHide: false },
        },
      ],
      tooltip: {
        mark: {
          title: { value: (datum: { group: string }) => datum.group },
          content: [
            {
              key: t(selectedMetric.labelKey),
              value: (datum: { value: number }) => formatMetric(datum.value),
            },
          ],
        },
      },
    }),
    [chartRadius, chartRows, formatMetric, selectedMetric.labelKey, t]
  )

  if (query.isError) {
    return (
      <div className='flex min-h-80 flex-col items-center justify-center gap-3 rounded-lg border'>
        <span className='text-destructive text-sm'>
          {t('Failed to load group analytics')}
        </span>
        <Button variant='outline' size='sm' onClick={() => query.refetch()}>
          <RotateCcw />
          {t('Retry')}
        </Button>
      </div>
    )
  }

  const statItems = [
    {
      key: 'count',
      label: t('Total Count'),
      value: formatNumber(analytics?.totals.count ?? 0, locale),
      icon: Hash,
      tone: 'info' as const,
    },
    {
      key: 'quota',
      label: t('Total Quota'),
      value: formatQuota(analytics?.totals.quota ?? 0),
      icon: Coins,
      tone: 'success' as const,
    },
    {
      key: 'tokens',
      label: t('Total Tokens'),
      value: formatCompactNumber(analytics?.totals.token_used ?? 0, locale),
      icon: Layers,
      tone: 'chart-4' as const,
    },
    {
      key: 'users',
      label: t('Total Users'),
      value: formatNumber(analytics?.totals.user_count ?? 0, locale),
      icon: UsersRound,
      tone: 'warning' as const,
    },
  ]

  let chartContent = themeReady ? (
    <VChart
      key={`${metric}-${chartRows.length}-${resolvedTheme}-${customization.preset}`}
      spec={{
        ...chartSpec,
        theme: resolvedTheme === 'dark' ? 'dark' : 'light',
        background: 'transparent',
      }}
      option={VCHART_OPTION}
    />
  ) : null
  if (chartRows.length === 0) {
    chartContent = (
      <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
        {t('No group usage data')}
      </div>
    )
  }
  if (query.isLoading) {
    chartContent = <Skeleton className='h-full w-full' />
  }

  let tableRows = rows.map((row) => (
    <TableRow key={row.group || '__unknown'}>
      <TableCell className='font-medium'>{row.group || t('Unknown')}</TableCell>
      <TableCell className='text-right font-mono tabular-nums'>
        {formatQuota(row.quota)}
      </TableCell>
      <TableCell className='text-right font-mono tabular-nums'>
        {formatNumber(row.count, locale)}
      </TableCell>
      <TableCell className='text-right font-mono tabular-nums'>
        {formatNumber(row.token_used, locale)}
      </TableCell>
      <TableCell className='text-right font-mono tabular-nums'>
        {formatNumber(row.user_count, locale)}
      </TableCell>
    </TableRow>
  ))
  if (rows.length === 0) {
    tableRows = [
      <TableRow key='empty'>
        <TableCell
          colSpan={5}
          className='text-muted-foreground h-24 text-center'
        >
          {t('No group usage data')}
        </TableCell>
      </TableRow>,
    ]
  }
  if (query.isLoading) {
    tableRows = [
      <TableRow key='loading'>
        <TableCell colSpan={5} className='h-24'>
          <Skeleton className='mx-auto h-5 w-40' />
        </TableCell>
      </TableRow>,
    ]
  }

  return (
    <div className='space-y-3 sm:space-y-4'>
      <div className='overflow-hidden rounded-lg border'>
        <div className='divide-border/60 grid grid-cols-2 divide-x sm:grid-cols-4'>
          {statItems.map((item) => {
            const Icon = item.icon
            return (
              <div key={item.key} className='min-w-0 px-3 py-3 sm:px-5 sm:py-4'>
                <div className='flex min-w-0 items-center gap-2'>
                  <IconBadge tone={item.tone} size='stat'>
                    <Icon />
                  </IconBadge>
                  <span className='text-muted-foreground truncate text-xs font-medium uppercase'>
                    {item.label}
                  </span>
                </div>
                {query.isLoading ? (
                  <Skeleton className='mt-2 h-7 w-20' />
                ) : (
                  <div className='mt-2 truncate font-mono text-xl font-bold tabular-nums sm:text-2xl'>
                    {item.value}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      <div className='overflow-hidden rounded-lg border'>
        <div className='flex flex-col gap-2 border-b px-3 py-2 sm:flex-row sm:items-center sm:justify-between sm:px-5 sm:py-3'>
          <div className='flex items-center gap-2'>
            <IconBadge tone='chart-2' size='sm'>
              <BarChart3 />
            </IconBadge>
            <span className='text-sm font-semibold'>
              {t('Group usage comparison')}
            </span>
            <span className='text-muted-foreground text-xs'>
              {t('Total:')} {formatNumber(analytics?.totals.group_count ?? 0)}
            </span>
          </div>
          <Tabs
            value={metric}
            onValueChange={(value) => setMetric(value as GroupAnalyticsMetric)}
          >
            <TabsList
              className='max-w-full overflow-x-auto'
              aria-label={t('Group metric')}
            >
              {METRIC_OPTIONS.map((option) => {
                const Icon = option.icon
                return (
                  <TabsTrigger
                    key={option.value}
                    value={option.value}
                    className='gap-1.5 px-2.5 text-xs'
                  >
                    <Icon data-icon='inline-start' aria-hidden='true' />
                    {t(option.labelKey)}
                  </TabsTrigger>
                )
              })}
            </TabsList>
          </Tabs>
        </div>
        <div className='h-[340px] p-2 sm:h-96'>{chartContent}</div>
      </div>

      <div className='overflow-x-auto rounded-lg border'>
        <Table className='min-w-[680px]'>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Group')}</TableHead>
              <TableHead className='text-right'>{t('Quota')}</TableHead>
              <TableHead className='text-right'>{t('Requests')}</TableHead>
              <TableHead className='text-right'>{t('Tokens')}</TableHead>
              <TableHead className='text-right'>{t('Users')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>{tableRows}</TableBody>
        </Table>
      </div>
    </div>
  )
}
