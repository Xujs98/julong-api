import { useQuery } from '@tanstack/react-query'
import type React from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
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
  formatNumber,
  formatQuota,
  formatTimestampToDate,
  formatTokens,
} from '@/lib/format'
import { getAllLogs } from '@/features/usage-logs/api'
import type { UsageLog } from '@/features/usage-logs/data/schema'

import { USER_ROLES, USER_STATUS, USER_STATUSES, isUserDeleted } from '../constants'
import { getUserUsageSummary } from '../api'
import { useUsers } from './users-provider'

const RECENT_LOG_LIMIT = 20

function InfoItem(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='rounded-md border bg-muted/20 px-3 py-2'>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='mt-1 text-sm font-medium break-all'>{props.value}</div>
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
      <DialogContent className='max-h-[88vh] overflow-hidden sm:max-w-[980px]'>
        <DialogHeader>
          <DialogTitle>{t('User detail')}</DialogTitle>
          <DialogDescription className='flex flex-wrap items-center gap-2'>
            <span className='text-foreground text-lg font-semibold'>
              {user?.username || '-'}
            </span>
            {user?.display_name && user.display_name !== user.username && (
              <span>{user.display_name}</span>
            )}
            {statusConfig && (
              <StatusBadge
                label={t(statusConfig.labelKey)}
                variant={statusConfig.variant}
                copyable={false}
                type='text'
              />
            )}
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue='info' className='min-h-0 gap-4'>
          <TabsList className='w-full justify-start sm:w-fit'>
            <TabsTrigger value='info'>{t('Basic Information')}</TabsTrigger>
            <TabsTrigger value='logs'>{t('Usage Logs')}</TabsTrigger>
          </TabsList>

          <ScrollArea className='max-h-[68vh] pr-3'>
            <TabsContent value='info' className='mt-0'>
              <div className='grid gap-3 sm:grid-cols-3'>
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
                  label={t('Wallet balance')}
                  value={formatQuota(user?.quota ?? 0)}
                />
                <InfoItem
                  label={t('Used:')}
                  value={formatQuota(user?.used_quota ?? 0)}
                />
                <InfoItem
                  label={t('Total token consumption')}
                  value={
                    summaryQuery.isLoading
                      ? t('Loading...')
                      : formatTokens(totalTokens)
                  }
                />
                <InfoItem
                  label={t('Requests:')}
                  value={formatNumber(user?.request_count ?? 0)}
                />
                <InfoItem
                  label={t('Created At')}
                  value={formatTimestampToDate(user?.created_at)}
                />
                <InfoItem
                  label={t('Last Login')}
                  value={formatTimestampToDate(user?.last_login_at)}
                />
                <div className='sm:col-span-3'>
                  <InfoItem label={t('Remark')} value={user?.remark || '-'} />
                </div>
              </div>
            </TabsContent>

            <TabsContent value='logs' className='mt-0'>
              <div className='overflow-hidden rounded-md border'>
                <Table>
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
