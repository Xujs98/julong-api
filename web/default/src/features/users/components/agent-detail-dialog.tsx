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
import { formatQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { getAgentDetail } from '../api'
import { useUsers } from './users-provider'

export function AgentDetailDialog() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow } = useUsers()
  const isOpen = open === 'agent-detail'
  const agentId = currentRow?.id

  const { data, isLoading } = useQuery({
    queryKey: ['agent-detail', agentId],
    queryFn: async () => {
      if (!agentId) return null
      const result = await getAgentDetail(agentId)
      return result.success ? result.data : null
    },
    enabled: isOpen && Boolean(agentId),
  })

  const agent = data?.agent ?? currentRow
  const redemptions = data?.redemptions ?? []
  const users = data?.users ?? []

  return (
    <Dialog open={isOpen} onOpenChange={(value) => !value && setOpen(null)}>
      <DialogContent className='max-h-[calc(100dvh-2rem)] overflow-hidden p-0 sm:max-w-[900px]'>
        <DialogHeader className='gap-0 border-b px-4 pt-4 pr-12 pb-0 sm:px-6 sm:pt-5 sm:pr-14'>
          <DialogTitle>{t('Agent detail')}</DialogTitle>
          <DialogDescription className='sr-only'>
            {agent?.username || '-'}
          </DialogDescription>
          <div className='mt-4 flex items-center gap-3 pb-4'>
            <Avatar size='lg' className='size-12'>
              <AvatarFallback className='text-base font-semibold'>
                {getInitials(agent?.display_name || agent?.username)}
              </AvatarFallback>
            </Avatar>
            <div className='min-w-0 flex-1'>
              <div className='flex flex-wrap items-center gap-2'>
                <span className='truncate text-base font-semibold'>
                  {agent?.username || '-'}
                </span>
                <StatusBadge
                  label={t('Agent')}
                  variant='success'
                  copyable={false}
                  type='text'
                />
              </div>
              <div className='text-muted-foreground mt-0.5 truncate text-sm'>
                {agent?.display_name || agent?.group || '-'}
              </div>
            </div>
          </div>
          <div className='grid grid-cols-3 border-t'>
            <Metric
              label={t('Agent discount (%)')}
              value={`${agent?.agent_discount ?? 100}%`}
            />
            <Metric label={t('Redemption Codes')} value={redemptions.length} />
            <Metric label={t('Users')} value={users.length} />
          </div>
        </DialogHeader>

        {isLoading ? (
          <div className='grid gap-3 p-4 sm:grid-cols-2 sm:p-6'>
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className='h-14 w-full' />
            ))}
          </div>
        ) : (
          <Tabs defaultValue='info' className='min-h-0 gap-0'>
            <TabsList variant='line' className='mx-4 mt-2 w-fit sm:mx-6'>
              <TabsTrigger value='info'>{t('Basic Information')}</TabsTrigger>
              <TabsTrigger value='redemptions'>
                {t('Redemption Codes')}
              </TabsTrigger>
              <TabsTrigger value='users'>{t('Users')}</TabsTrigger>
            </TabsList>

            <ScrollArea className='max-h-[calc(100dvh-17rem)]'>
              <TabsContent value='info' className='p-4 sm:p-6'>
                <div className='grid overflow-hidden rounded-lg border sm:grid-cols-2'>
                  <InfoItem label={t('ID')} value={agent?.id ?? '-'} />
                  <InfoItem
                    label={t('Username')}
                    value={agent?.username ?? '-'}
                  />
                  <InfoItem label={t('Group')} value={agent?.group || '-'} />
                  <InfoItem
                    label={t('Wallet balance')}
                    value={formatQuota(agent?.quota ?? 0)}
                    emphasized
                  />
                  <InfoItem
                    label={t('Used:')}
                    value={formatQuota(agent?.used_quota ?? 0)}
                  />
                  <InfoItem
                    label={t('Requests:')}
                    value={(agent?.request_count ?? 0).toLocaleString()}
                  />
                  <InfoItem
                    className='sm:col-span-2'
                    label={t('Agent top-up link')}
                    value={agent?.agent_topup_link || '-'}
                  />
                </div>
              </TabsContent>

              <TabsContent value='redemptions' className='p-4 sm:p-6'>
                <DataPanel>
                  <Table className='min-w-[680px]'>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('ID')}</TableHead>
                        <TableHead>{t('Name')}</TableHead>
                        <TableHead>{t('Quota')}</TableHead>
                        <TableHead>{t('Created')}</TableHead>
                        <TableHead>{t('Redeemed By')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {redemptions.length === 0 ? (
                        <EmptyRow colSpan={5} />
                      ) : (
                        redemptions.map((redemption) => (
                          <TableRow key={redemption.id}>
                            <TableCell>{redemption.id}</TableCell>
                            <TableCell className='font-medium'>
                              {redemption.name}
                            </TableCell>
                            <TableCell>
                              {formatQuota(redemption.quota)}
                            </TableCell>
                            <TableCell>
                              {formatTimestampToDate(redemption.created_time)}
                            </TableCell>
                            <TableCell>
                              {redemption.used_user_id > 0
                                ? redemption.used_user_id
                                : '-'}
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </DataPanel>
              </TabsContent>

              <TabsContent value='users' className='p-4 sm:p-6'>
                <DataPanel>
                  <Table className='min-w-[620px]'>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('ID')}</TableHead>
                        <TableHead>{t('Username')}</TableHead>
                        <TableHead>{t('Quota')}</TableHead>
                        <TableHead>{t('Used:')}</TableHead>
                        <TableHead>{t('Requests:')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {users.length === 0 ? (
                        <EmptyRow colSpan={5} />
                      ) : (
                        users.map((user) => (
                          <TableRow key={user.id}>
                            <TableCell>{user.id}</TableCell>
                            <TableCell className='font-medium'>
                              {user.username}
                            </TableCell>
                            <TableCell>{formatQuota(user.quota)}</TableCell>
                            <TableCell>
                              {formatQuota(user.used_quota)}
                            </TableCell>
                            <TableCell>
                              {user.request_count.toLocaleString()}
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </DataPanel>
              </TabsContent>
            </ScrollArea>
          </Tabs>
        )}
      </DialogContent>
    </Dialog>
  )
}

function Metric(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='border-r px-2 py-3 last:border-r-0 sm:px-4'>
      <div className='text-muted-foreground text-xs leading-4'>
        {props.label}
      </div>
      <div className='mt-1 text-lg font-semibold tabular-nums'>
        {props.value}
      </div>
    </div>
  )
}

function InfoItem(props: {
  label: string
  value: React.ReactNode
  className?: string
  emphasized?: boolean
}) {
  return (
    <div
      className={cn(
        'border-b p-3 last:border-b-0 sm:border-r sm:[&:nth-child(even)]:border-r-0',
        props.className
      )}
    >
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div
        className={cn(
          'mt-1 break-all',
          props.emphasized
            ? 'text-base font-semibold tabular-nums'
            : 'text-sm font-medium'
        )}
      >
        {props.value}
      </div>
    </div>
  )
}

function DataPanel(props: { children: React.ReactNode }) {
  return (
    <div className='overflow-x-auto rounded-lg border'>{props.children}</div>
  )
}

function getInitials(value?: string) {
  if (!value) return '-'
  return value.trim().slice(0, 2).toUpperCase()
}

function EmptyRow(props: { colSpan: number }) {
  const { t } = useTranslation()
  return (
    <TableRow>
      <TableCell
        colSpan={props.colSpan}
        className='text-muted-foreground py-8 text-center'
      >
        {t('No data')}
      </TableCell>
    </TableRow>
  )
}
