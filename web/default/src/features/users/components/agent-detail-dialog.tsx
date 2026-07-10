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

import {
  Card,
  CardContent,
} from '@/components/ui/card'
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
import { StatusBadge } from '@/components/status-badge'
import { formatQuota, formatTimestampToDate } from '@/lib/format'

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
      <DialogContent className='sm:max-w-[980px]'>
        <DialogHeader className='gap-3'>
          <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
            <div className='space-y-2'>
              <DialogTitle>{t('Agent detail')}</DialogTitle>
              <DialogDescription className='flex flex-wrap items-center gap-2'>
                <span className='text-foreground text-lg font-semibold'>
                  {agent?.username || '-'}
                </span>
                {agent?.display_name && agent.display_name !== agent.username && (
                  <span>{agent.display_name}</span>
                )}
                <StatusBadge
                  label={t('Agent')}
                  variant='success'
                  copyable={false}
                  type='text'
                />
              </DialogDescription>
            </div>
            <div className='grid grid-cols-2 gap-2 sm:grid-cols-3'>
              <Metric label={t('Agent discount (%)')} value={`${agent?.agent_discount ?? 100}%`} />
              <Metric label={t('Redemption Codes')} value={redemptions.length} />
              <Metric label={t('Users')} value={users.length} />
            </div>
          </div>
        </DialogHeader>

        {isLoading ? (
          <div className='text-muted-foreground py-8 text-center text-sm'>
            {t('Loading...')}
          </div>
        ) : (
          <Tabs defaultValue='info' className='gap-4'>
            <TabsList className='w-full justify-start sm:w-fit'>
              <TabsTrigger value='info'>{t('Basic Information')}</TabsTrigger>
              <TabsTrigger value='redemptions'>
                {t('Redemption Codes')}
              </TabsTrigger>
              <TabsTrigger value='users'>{t('Users')}</TabsTrigger>
            </TabsList>

            <TabsContent value='info'>
              <div className='grid gap-3 sm:grid-cols-3'>
                <InfoItem label={t('ID')} value={agent?.id ?? '-'} />
                <InfoItem label={t('Username')} value={agent?.username ?? '-'} />
                <InfoItem label={t('Group')} value={agent?.group || '-'} />
                <InfoItem label={t('Wallet balance')} value={formatQuota(agent?.quota ?? 0)} />
                <InfoItem label={t('Used:')} value={formatQuota(agent?.used_quota ?? 0)} />
                <InfoItem label={t('Requests:')} value={(agent?.request_count ?? 0).toLocaleString()} />
                <div className='sm:col-span-3'>
                  <div className='text-muted-foreground text-xs'>{t('Agent top-up link')}</div>
                  <div className='mt-1 rounded-md border bg-muted/20 px-3 py-2 text-sm break-all'>
                    {agent?.agent_topup_link || '-'}
                  </div>
                </div>
              </div>
            </TabsContent>

            <TabsContent value='redemptions'>
              <DataPanel>
                <Table>
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
                          <TableCell className='font-medium'>{redemption.name}</TableCell>
                          <TableCell>{formatQuota(redemption.quota)}</TableCell>
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

            <TabsContent value='users'>
              <DataPanel>
                <Table>
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
                          <TableCell className='font-medium'>{user.username}</TableCell>
                          <TableCell>{formatQuota(user.quota)}</TableCell>
                          <TableCell>{formatQuota(user.used_quota)}</TableCell>
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
          </Tabs>
        )}
      </DialogContent>
    </Dialog>
  )
}

function Metric(props: { label: string; value: React.ReactNode }) {
  return (
    <Card size='sm' className='min-w-[110px] gap-1 rounded-lg py-2'>
      <CardContent className='px-3'>
        <div className='text-muted-foreground text-xs'>{props.label}</div>
        <div className='mt-1 text-base font-semibold tabular-nums'>
          {props.value}
        </div>
      </CardContent>
    </Card>
  )
}

function InfoItem(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='rounded-md border bg-muted/20 px-3 py-2'>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='mt-1 text-sm font-medium break-all'>
        {props.value}
      </div>
    </div>
  )
}

function DataPanel(props: { children: React.ReactNode }) {
  return (
    <div className='overflow-hidden rounded-md border'>
      <ScrollArea className='h-[360px]'>
        {props.children}
      </ScrollArea>
    </div>
  )
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
