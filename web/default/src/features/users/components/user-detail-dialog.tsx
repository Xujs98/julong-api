import { useQuery } from '@tanstack/react-query'
import type React from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  getAdminPlans,
  getUserSubscriptions,
} from '@/features/subscriptions/api'
import type { UserSubscription } from '@/features/subscriptions/types'
import { getAllLogs } from '@/features/usage-logs/api'
import type { UsageLog } from '@/features/usage-logs/data/schema'
import {
  formatNumber,
  formatQuota,
  formatTimestampToDate,
  formatTokens,
} from '@/lib/format'
import { cn } from '@/lib/utils'

import { getUserUsageSummary } from '../api'
import {
  USER_ROLES,
  USER_STATUS,
  USER_STATUSES,
  isUserDeleted,
} from '../constants'
import { useUsers } from './users-provider'

const RECENT_LOG_LIMIT = 20

function InfoItem(props: {
  label: string
  value: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'border-b p-3 last:border-b-0 sm:border-r sm:[&:nth-child(3n)]:border-r-0',
        props.className
      )}
    >
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='mt-1 text-sm font-medium break-all'>{props.value}</div>
    </div>
  )
}

function Metric(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='border-r px-3 py-3 even:border-r-0 sm:border-r sm:border-b-0 sm:px-4 sm:last:border-r-0 sm:even:border-r [&:nth-child(-n+2)]:border-b'>
      <div className='text-muted-foreground text-xs leading-4'>
        {props.label}
      </div>
      <div className='mt-1 text-base font-semibold tabular-nums sm:text-lg'>
        {props.value}
      </div>
    </div>
  )
}

function EmptyRow({ colSpan }: { colSpan: number }) {
  const { t } = useTranslation()
  return (
    <TableRow>
      <TableCell colSpan={colSpan} className='h-28 text-center'>
        <span className='text-muted-foreground text-sm'>
          {t('No usage logs found.')}
        </span>
      </TableCell>
    </TableRow>
  )
}

export function UserDetailDialog() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow } = useUsers()
  const isOpen = open === 'user-detail'
  const user = currentRow

  const summaryQuery = useQuery({
    queryKey: ['admin-user-usage-summary', user?.id],
    enabled: isOpen && Boolean(user?.id),
    queryFn: async () => {
      const result = await getUserUsageSummary(user!.id)
      return result.success ? result.data : null
    },
  })

  const logsQuery = useQuery({
    queryKey: ['admin-user-detail-logs', user?.username],
    enabled: isOpen && Boolean(user?.username),
    queryFn: async () => {
      const result = await getAllLogs({
        p: 1,
        page_size: RECENT_LOG_LIMIT,
        type: 2,
        username: user!.username,
      })
      return result.success ? (result.data?.items as UsageLog[]) || [] : []
    },
  })

  const subscriptionsQuery = useQuery({
    queryKey: ['admin-user-detail-subscriptions', user?.id],
    enabled: isOpen && Boolean(user?.id),
    queryFn: async () => {
      const [subscriptionsResult, plansResult] = await Promise.all([
        getUserSubscriptions(user!.id),
        getAdminPlans(),
      ])
      const now = Date.now() / 1000
      const subscriptions = (subscriptionsResult.data || [])
        .map((record) => record.subscription)
        .filter(
          (subscription) =>
            subscription.status === 'active' && subscription.end_time > now
        )
      const planTitles = new Map(
        (plansResult.data || []).map((record) => [
          record.plan.id,
          record.plan.title,
        ])
      )

      return { subscriptions, planTitles }
    },
  })

  const statusConfig = user
    ? isUserDeleted(user)
      ? USER_STATUSES[USER_STATUS.DELETED]
      : USER_STATUSES[user.status as keyof typeof USER_STATUSES]
    : null
  const roleConfig = user
    ? USER_ROLES[user.role as keyof typeof USER_ROLES]
    : null
  const totalTokens = summaryQuery.data?.total_tokens ?? 0
  const logs = logsQuery.data || []

  return (
    <Dialog open={isOpen} onOpenChange={(value) => !value && setOpen(null)}>
      <DialogContent className='max-h-[calc(100dvh-2rem)] overflow-hidden p-0 sm:max-w-[900px]'>
        <DialogHeader className='gap-0 border-b px-4 pt-4 pr-12 pb-0 sm:px-6 sm:pt-5 sm:pr-14'>
          <DialogTitle>{t('User detail')}</DialogTitle>
          <DialogDescription className='sr-only'>
            {user?.username || '-'}
          </DialogDescription>
          <div className='mt-4 flex items-center gap-3 pb-4'>
            <Avatar size='lg' className='size-12'>
              <AvatarFallback className='text-base font-semibold'>
                {getInitials(user?.display_name || user?.username)}
              </AvatarFallback>
            </Avatar>
            <div className='min-w-0 flex-1'>
              <div className='flex flex-wrap items-center gap-2'>
                <span className='truncate text-base font-semibold'>
                  {user?.username || '-'}
                </span>
                {statusConfig && (
                  <StatusBadge
                    label={t(statusConfig.labelKey)}
                    variant={statusConfig.variant}
                    copyable={false}
                    type='text'
                  />
                )}
              </div>
              <div className='text-muted-foreground mt-0.5 truncate text-sm'>
                {user?.display_name || user?.email || '-'}
              </div>
            </div>
          </div>
          <div className='grid grid-cols-2 border-t sm:grid-cols-4'>
            <Metric
              label={t('Wallet balance')}
              value={formatQuota(user?.quota ?? 0)}
            />
            <Metric
              label={t('Used:')}
              value={formatQuota(user?.used_quota ?? 0)}
            />
            <Metric
              label={t('Total token consumption')}
              value={
                summaryQuery.isLoading ? (
                  <Skeleton className='h-5 w-16' />
                ) : (
                  formatTokens(totalTokens)
                )
              }
            />
            <Metric
              label={t('Requests:')}
              value={formatNumber(user?.request_count ?? 0)}
            />
          </div>
        </DialogHeader>

        <Tabs defaultValue='info' className='min-h-0 gap-0'>
          <TabsList variant='line' className='mx-4 mt-2 w-fit sm:mx-6'>
            <TabsTrigger value='info'>{t('Basic Information')}</TabsTrigger>
            <TabsTrigger value='logs'>{t('Usage Logs')}</TabsTrigger>
          </TabsList>

          <ScrollArea className='max-h-[calc(100dvh-19rem)]'>
            <TabsContent value='info' className='p-4 sm:p-6'>
              <div className='grid overflow-hidden rounded-lg border sm:grid-cols-3'>
                <InfoItem label={t('ID')} value={user?.id ?? '-'} />
                <InfoItem label={t('Username')} value={user?.username ?? '-'} />
                <InfoItem
                  label={t('Display Name')}
                  value={user?.display_name || '-'}
                />
                <InfoItem label={t('Email')} value={user?.email || '-'} />
                <InfoItem
                  label={t('Role')}
                  value={roleConfig ? t(roleConfig.labelKey) : '-'}
                />
                <InfoItem label={t('Group')} value={user?.group || '-'} />
                <InfoItem
                  label={t('Created At')}
                  value={formatTimestampToDate(user?.created_at)}
                />
                <InfoItem
                  label={t('Last Login')}
                  value={formatTimestampToDate(user?.last_login_at)}
                />
                <InfoItem
                  label={t('Subscriptions')}
                  value={
                    <SubscriptionInfo
                      subscriptions={
                        subscriptionsQuery.data?.subscriptions || []
                      }
                      planTitles={subscriptionsQuery.data?.planTitles}
                      isLoading={subscriptionsQuery.isLoading}
                    />
                  }
                />
                <InfoItem
                  className='sm:col-span-3 sm:border-r-0'
                  label={t('Remark')}
                  value={user?.remark || '-'}
                />
              </div>
            </TabsContent>

            <TabsContent value='logs' className='p-4 sm:p-6'>
              <div className='overflow-x-auto rounded-lg border'>
                <Table className='min-w-[780px]'>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Time')}</TableHead>
                      <TableHead>{t('Model')}</TableHead>
                      <TableHead>{t('Token')}</TableHead>
                      <TableHead>{t('Tokens')}</TableHead>
                      <TableHead>{t('Cost')}</TableHead>
                      <TableHead>{t('Channel')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {logsQuery.isLoading ? (
                      <TableRow>
                        <TableCell colSpan={6} className='h-28 text-center'>
                          {t('Loading...')}
                        </TableCell>
                      </TableRow>
                    ) : logs.length === 0 ? (
                      <EmptyRow colSpan={6} />
                    ) : (
                      logs.map((log) => {
                        const tokenCount =
                          (log.prompt_tokens || 0) +
                          (log.completion_tokens || 0)
                        return (
                          <TableRow key={log.id}>
                            <TableCell className='whitespace-nowrap'>
                              {formatTimestampToDate(log.created_at)}
                            </TableCell>
                            <TableCell>{log.model_name || '-'}</TableCell>
                            <TableCell>{log.token_name || '-'}</TableCell>
                            <TableCell>{formatTokens(tokenCount)}</TableCell>
                            <TableCell>{formatQuota(log.quota || 0)}</TableCell>
                            <TableCell>
                              {log.channel_name || log.channel || '-'}
                            </TableCell>
                          </TableRow>
                        )
                      })
                    )}
                  </TableBody>
                </Table>
              </div>
            </TabsContent>
          </ScrollArea>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}

function getInitials(value?: string) {
  if (!value) return '-'
  return value.trim().slice(0, 2).toUpperCase()
}

function SubscriptionInfo(props: {
  subscriptions: UserSubscription[]
  planTitles?: Map<number, string>
  isLoading: boolean
}) {
  const { t } = useTranslation()

  if (props.isLoading) {
    return <Skeleton className='h-9 w-full max-w-44' />
  }

  if (props.subscriptions.length === 0) {
    return (
      <span className='text-muted-foreground font-normal'>
        {t('No subscription records')}
      </span>
    )
  }

  const subscription = props.subscriptions[0]
  const remaining = Math.max(
    0,
    subscription.amount_total - subscription.amount_used
  )

  return (
    <div className='flex min-w-0 flex-col gap-1.5'>
      <div className='flex min-w-0 items-center gap-2'>
        <span className='truncate'>
          {props.planTitles?.get(subscription.plan_id) ||
            `#${subscription.plan_id}`}
        </span>
        <StatusBadge
          label={t('Active')}
          variant='success'
          copyable={false}
          type='text'
        />
      </div>
      <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs font-normal'>
        <span>
          {t('Remaining')}:{' '}
          {subscription.amount_total > 0
            ? formatQuota(remaining)
            : t('Unlimited')}
        </span>
        <span>
          {t('Expires')}: {formatTimestampToDate(subscription.end_time)}
        </span>
        {props.subscriptions.length > 1 && (
          <span>
            {t('Subscriptions')}: {props.subscriptions.length}
          </span>
        )}
      </div>
    </div>
  )
}
