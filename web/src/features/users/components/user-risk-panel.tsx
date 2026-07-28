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
import {
  Activity,
  CircleAlert,
  Network,
  RotateCcw,
  ShieldAlert,
  type LucideIcon,
  Unplug,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { formatNumber, formatQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { getUserRiskReport, updateUserRiskDetection } from '../api'
import type {
  UserRiskLevel,
  UserRiskReport,
  UserRiskSignalCode,
} from '../types'

const windowOptions = [
  { value: '1', labelKey: 'Last 24 hours' },
  { value: '7', labelKey: 'Last 7 days' },
  { value: '30', labelKey: 'Last 30 days' },
] as const

const signalMeta: Record<
  UserRiskSignalCode,
  { labelKey: string; icon: LucideIcon }
> = {
  sensitive_word_attempts: {
    labelKey: 'Sensitive-word attempts',
    icon: ShieldAlert,
  },
  failed_request_rate: {
    labelKey: 'High request failure rate',
    icon: CircleAlert,
  },
  client_abort: {
    labelKey: 'Frequent client disconnects',
    icon: Unplug,
  },
  abnormal_stream: {
    labelKey: 'Abnormal stream terminations',
    icon: Activity,
  },
  failed_refunds: {
    labelKey: 'Repeated failure refunds',
    icon: RotateCcw,
  },
  refund_after_output: {
    labelKey: 'Failed refunds after response output',
    icon: ShieldAlert,
  },
  multiple_ips: {
    labelKey: 'Multiple active IP addresses',
    icon: Network,
  },
}

const levelMeta: Record<
  UserRiskLevel,
  { labelKey: string; variant: StatusVariant; progressClass: string }
> = {
  low: {
    labelKey: 'Low risk',
    variant: 'success',
    progressClass: '[&_[data-slot=progress-indicator]]:bg-success',
  },
  medium: {
    labelKey: 'Medium risk',
    variant: 'warning',
    progressClass: '[&_[data-slot=progress-indicator]]:bg-warning',
  },
  high: {
    labelKey: 'High risk',
    variant: 'danger',
    progressClass: '[&_[data-slot=progress-indicator]]:bg-destructive',
  },
}

function RiskMetric(props: { label: string; value: ReactNode }) {
  return (
    <div className='border-r border-b p-3 last:border-r-0 sm:border-b-0'>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='mt-1 text-base font-semibold tabular-nums'>
        {props.value}
      </div>
    </div>
  )
}

function RiskPanelSkeleton() {
  return (
    <div className='space-y-3'>
      <Skeleton className='h-28 w-full' />
      <Skeleton className='h-20 w-full' />
      <Skeleton className='h-32 w-full' />
    </div>
  )
}

function RiskReportContent(props: { report: UserRiskReport }) {
  const { t } = useTranslation()
  const level = levelMeta[props.report.level]
  const nonClientStreamErrors = Math.max(
    0,
    props.report.summary.abnormal_stream_count -
      props.report.summary.client_abort_count
  )

  return (
    <div className='space-y-4'>
      <div className='rounded-lg border p-4'>
        <div className='flex items-start justify-between gap-4'>
          <div>
            <div className='text-muted-foreground text-xs'>
              {t('Risk score')}
            </div>
            <div className='mt-1 text-2xl font-semibold tracking-normal tabular-nums'>
              {props.report.score}
              <span className='text-muted-foreground text-sm font-normal'>
                /100
              </span>
            </div>
          </div>
          <StatusBadge
            label={t(level.labelKey)}
            variant={level.variant}
            copyable={false}
          />
        </div>
        <Progress
          value={props.report.score}
          className={cn('mt-3 gap-0', level.progressClass)}
        />
      </div>

      <div className='grid grid-cols-2 overflow-hidden rounded-lg border sm:grid-cols-3'>
        <RiskMetric
          label={t('Total requests')}
          value={formatNumber(props.report.summary.total_requests)}
        />
        <RiskMetric
          label={t('Request errors')}
          value={`${formatNumber(props.report.summary.error_count)} (${Math.round(props.report.summary.error_rate * 100)}%)`}
        />
        <RiskMetric
          label={t('Refunded quota')}
          value={formatQuota(props.report.summary.refund_quota)}
        />
        <RiskMetric
          label={t('Failed refunds')}
          value={formatNumber(props.report.summary.failed_refund_count)}
        />
        <RiskMetric
          label={t('Refunds after output')}
          value={formatNumber(props.report.summary.refund_after_output_count)}
        />
        <RiskMetric
          label={t('Abnormal streams')}
          value={formatNumber(
            props.report.summary.client_abort_count + nonClientStreamErrors
          )}
        />
        <RiskMetric
          label={t('Unique IPs')}
          value={formatNumber(props.report.summary.unique_ip_count)}
        />
      </div>

      <section>
        <h3 className='mb-2 text-sm font-medium'>{t('Risk signals')}</h3>
        {props.report.signals.length === 0 ? (
          <div className='text-muted-foreground flex h-24 items-center justify-center rounded-lg border text-sm'>
            {t('No risk signals detected in this period.')}
          </div>
        ) : (
          <div className='divide-y overflow-hidden rounded-lg border'>
            {props.report.signals.map((signal) => {
              const meta = signalMeta[signal.code]
              const severity = levelMeta[signal.severity]
              const Icon = meta.icon
              return (
                <div
                  key={signal.code}
                  className='flex items-start gap-3 p-3 sm:items-center'
                >
                  <div
                    className={cn(
                      'bg-muted flex size-8 shrink-0 items-center justify-center rounded-md',
                      signal.severity === 'high' &&
                        'bg-destructive/10 text-destructive',
                      signal.severity === 'medium' &&
                        'bg-warning/10 text-warning'
                    )}
                  >
                    <Icon className='size-4' aria-hidden='true' />
                  </div>
                  <div className='min-w-0 flex-1'>
                    <div className='text-sm font-medium'>
                      {t(meta.labelKey)}
                    </div>
                    <div className='text-muted-foreground mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 text-xs'>
                      <span>
                        {t('{{count}} occurrences', { count: signal.count })}
                      </span>
                      <span>
                        {t('Last detected: {{time}}', {
                          time: formatTimestampToDate(signal.last_seen),
                        })}
                      </span>
                    </div>
                  </div>
                  <StatusBadge
                    label={`+${signal.score}`}
                    variant={severity.variant}
                    copyable={false}
                  />
                </div>
              )
            })}
          </div>
        )}
      </section>
    </div>
  )
}

export function UserRiskPanel(props: { userId?: number }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [windowDays, setWindowDays] = useState<1 | 7 | 30>(7)
  const query = useQuery({
    queryKey: ['admin-user-risk-report', props.userId, windowDays],
    enabled: Boolean(props.userId),
    queryFn: async () => {
      if (!props.userId) return null
      const result = await getUserRiskReport(props.userId, windowDays)
      if (!result.success || !result.data) {
        throw new Error(result.message || 'Load failed')
      }
      return result.data
    },
  })
  const updateMutation = useMutation({
    mutationFn: async (enabled: boolean) => {
      if (!props.userId) throw new Error('Missing user')
      const result = await updateUserRiskDetection(props.userId, enabled)
      if (!result.success) throw new Error(result.message || 'Update failed')
      return result.data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['admin-user-risk-report', props.userId],
      })
    },
  })

  const report = query.data
  const switchChecked = Boolean(report?.global_enabled || report?.user_enabled)

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <h2 className='text-sm font-medium'>{t('Risk detection')}</h2>
          <p className='text-muted-foreground mt-0.5 text-xs'>
            {report?.global_enabled
              ? t('Enabled globally for all users')
              : t('Enable risk detection for this user')}
          </p>
        </div>
        <div className='flex items-center gap-3'>
          <Switch
            checked={switchChecked}
            disabled={
              !props.userId ||
              query.isLoading ||
              updateMutation.isPending ||
              report?.global_enabled
            }
            onCheckedChange={(checked) => updateMutation.mutate(checked)}
            aria-label={t('Enable risk detection for this user')}
          />
          <Select
            value={String(windowDays)}
            onValueChange={(value) => {
              const parsed = Number(value)
              if (parsed === 1 || parsed === 7 || parsed === 30) {
                setWindowDays(parsed)
              }
            }}
          >
            <SelectTrigger className='w-40'>
              <SelectValue>
                {t(
                  windowOptions.find(
                    (option) => option.value === String(windowDays)
                  )?.labelKey || 'Last 7 days'
                )}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {windowOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {t(option.labelKey)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {query.isLoading && <RiskPanelSkeleton />}
      {query.isError && (
        <div className='flex h-28 flex-col items-center justify-center gap-2 rounded-lg border'>
          <span className='text-destructive text-sm'>
            {t('Failed to load risk report')}
          </span>
          <Button size='sm' variant='outline' onClick={() => query.refetch()}>
            {t('Retry')}
          </Button>
        </div>
      )}
      {updateMutation.isError && (
        <div className='text-destructive text-sm'>
          {t('Failed to update risk detection')}
        </div>
      )}
      {report && !report.enabled && (
        <div className='text-muted-foreground flex h-28 items-center justify-center rounded-lg border px-4 text-center text-sm'>
          {t(
            'Risk detection is disabled for this user. Enable the user switch or turn on the global switch in System Settings.'
          )}
        </div>
      )}
      {report?.enabled && <RiskReportContent report={report} />}
    </div>
  )
}
