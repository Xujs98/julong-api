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
import type { ColumnDef } from '@tanstack/react-table'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { ActivityTimeCell } from '@/components/activity-time-cell'
import { BadgeCell } from '@/components/data-table'
import { GroupBadge } from '@/components/group-badge'
import { LongText } from '@/components/long-text'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { canOpenUserDetails } from '@/lib/admin-permissions'
import { getCurrencyDisplay } from '@/lib/currency'
import { formatQuota } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  USER_STATUS,
  USER_STATUSES,
  USER_ROLES,
  isUserDeleted,
} from '../constants'
import {
  getUserTagFilterValue,
  USER_TAG_ROW_MARKER_CLASS_NAME,
} from '../lib/user-tags'
import type { User, UserTag } from '../types'
import { DataTableRowActions } from './data-table-row-actions'
import { UserOnlineBadge } from './user-online-badge'
import { UserQuotaCell } from './user-quota-cell'
import { useUsers } from './users-provider'

export function useUsersColumns(tags: UserTag[] = []): ColumnDef<User>[] {
  const { t } = useTranslation()
  const { setOpen, setCurrentRow } = useUsers()
  const viewer = useAuthStore((state) => state.auth.user)
  const currencyConfig = useSystemConfigStore((state) => state.config.currency)
  const { meta: currency } = getCurrencyDisplay()
  const quotaUnit = currency.kind === 'tokens' ? t('Tokens') : currency.symbol
  return useMemo<ColumnDef<User>[]>(
    () => [
      {
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            indeterminate={table.getIsSomePageRowsSelected()}
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
            aria-label='Select all'
            className='translate-y-[2px]'
          />
        ),
        cell: ({ row }) => {
          const tag = row.original.tag
          return (
            <div className='relative flex h-9 items-center'>
              {tag && (
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <span
                        className={USER_TAG_ROW_MARKER_CLASS_NAME}
                        style={{ backgroundColor: tag.color }}
                        aria-label={t('Tag: {{name}}', { name: tag.name })}
                      />
                    }
                  />
                  <TooltipContent>{tag.name}</TooltipContent>
                </Tooltip>
              )}
              <Checkbox
                checked={row.getIsSelected()}
                onCheckedChange={(value) => row.toggleSelected(!!value)}
                aria-label='Select row'
                className='translate-y-[2px]'
              />
            </div>
          )
        },
        enableSorting: false,
        enableHiding: false,
        size: 40,
      },
      {
        accessorKey: 'id',
        header: t('ID'),
        cell: ({ row }) => (
          <TableId
            value={row.getValue('id') as number}
            className='w-[60px] text-sm'
          />
        ),
        size: 80,
        meta: { mobileOrder: 10 },
      },
      {
        id: 'tag_id',
        accessorFn: (user) => getUserTagFilterValue(user.tag_id),
        header: t('Tag'),
        filterFn: (row, id, value) => value.includes(String(row.getValue(id))),
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: 'username',
        header: t('Username'),
        cell: ({ row }) => {
          const user = row.original
          const username = row.getValue('username') as string
          const displayName = row.original.display_name
          const remark = row.original.remark

          return (
            <div className='flex min-w-[160px] flex-col gap-1'>
              <div className='flex items-center gap-2'>
                {canOpenUserDetails(viewer, user.role) ? (
                  <button
                    type='button'
                    className='text-primary hover:text-primary/80 max-w-[140px] cursor-pointer truncate text-left font-medium underline-offset-4 hover:underline'
                    onClick={() => {
                      setCurrentRow(user)
                      setOpen('user-detail')
                    }}
                  >
                    {username}
                  </button>
                ) : (
                  <span className='max-w-[140px] truncate font-medium'>
                    {username}
                  </span>
                )}
                <UserOnlineBadge user={user} />
                {user.risk_tag && (
                  <Badge
                    className='text-white'
                    style={{ backgroundColor: user.risk_tag.color }}
                  >
                    {t(user.risk_tag.name)}
                  </Badge>
                )}
                {remark && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <StatusBadge variant='success' copyable={false} />
                      }
                    >
                      <LongText className='max-w-[80px]'>{remark}</LongText>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className='text-xs'>{remark}</p>
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>
              {displayName && displayName !== username && (
                <LongText className='text-muted-foreground max-w-[180px] text-xs'>
                  {displayName}
                </LongText>
              )}
              <div className='flex flex-wrap gap-1'>
                {(user.shared_ip_user_count ?? 0) > 1 && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <StatusBadge
                          label={t('Shared IP: {{count}} users', {
                            count: user.shared_ip_user_count,
                          })}
                          variant='warning'
                          copyable={false}
                          size='sm'
                        />
                      }
                    />
                    <TooltipContent>{user.last_login_ip}</TooltipContent>
                  </Tooltip>
                )}
                {user.last_login_ip_blocked && (
                  <StatusBadge
                    label={t('IP blocked')}
                    variant='danger'
                    copyable={false}
                    size='sm'
                  />
                )}
              </div>
            </div>
          )
        },
        enableHiding: false,
        size: 220,
        meta: { mobileTitle: true },
      },
      {
        id: 'agent',
        header: t('Agent'),
        cell: ({ row }) => {
          const user = row.original
          return (
            <div className='flex flex-col gap-1'>
              {user.is_agent ? (
                <button
                  type='button'
                  className='text-success w-fit min-w-[2.5rem] cursor-pointer text-left text-sm font-medium whitespace-nowrap'
                  onClick={() => {
                    setCurrentRow(user)
                    setOpen('agent-detail')
                  }}
                >
                  {t('Agent')}
                </button>
              ) : (
                <span className='text-muted-foreground text-xs'>-</span>
              )}
              {user.agent_username && (
                <span className='text-muted-foreground text-xs'>
                  {t('Belongs to')}: {user.agent_username}
                </span>
              )}
            </div>
          )
        },
        enableSorting: false,
        size: 140,
      },
      {
        accessorKey: 'status',
        header: t('Status'),
        cell: ({ row }) => {
          const user = row.original
          const requestCount = user.request_count

          const statusConfig = isUserDeleted(user)
            ? USER_STATUSES[USER_STATUS.DELETED]
            : USER_STATUSES[user.status as keyof typeof USER_STATUSES]

          if (!statusConfig) {
            return null
          }

          return (
            <Tooltip>
              <TooltipTrigger render={<div className='-ml-1.5 cursor-help' />}>
                <StatusBadge
                  label={t(statusConfig.labelKey)}
                  variant={
                    isUserDeleted(user) ? 'neutral' : statusConfig.variant
                  }
                  copyable={false}
                  className='font-normal'
                />
              </TooltipTrigger>
              <TooltipContent>
                <p className='text-xs'>
                  {t('Requests:')} {requestCount.toLocaleString()}
                </p>
              </TooltipContent>
            </Tooltip>
          )
        },
        filterFn: (row, id, value) => {
          return value.includes(String(row.getValue(id)))
        },
        enableSorting: false,
        size: 120,
        meta: { mobileBadge: true },
      },
      {
        id: 'quota',
        accessorKey: 'quota',
        header: `${t('Available Balance')} (${quotaUnit})`,
        cell: ({ row }) => {
          const user = row.original
          return <UserQuotaCell remaining={user.quota} used={user.used_quota} />
        },
        size: 180,
        minSize: 160,
        meta: { mobileOrder: 40 },
      },
      {
        accessorKey: 'group',
        header: t('User Group'),
        cell: ({ row }) => {
          const group = row.getValue('group') as string
          return (
            <BadgeCell>
              <GroupBadge group={group} className='font-normal' />
            </BadgeCell>
          )
        },
        filterFn: (row, id, value) => {
          const group = String(
            row.getValue(id) || t('User Group')
          ).toLowerCase()
          const searchValue = String(value).toLowerCase()
          return group.includes(searchValue)
        },
        size: 140,
        meta: { mobileOrder: 30 },
      },
      {
        accessorKey: 'role',
        header: t('Role'),
        cell: ({ row }) => {
          const roleValue = row.getValue('role') as number
          const roleConfig = USER_ROLES[roleValue as keyof typeof USER_ROLES]

          if (!roleConfig) {
            return null
          }

          return <span className='text-sm'>{t(roleConfig.labelKey)}</span>
        },
        filterFn: (row, id, value) => {
          return value.includes(String(row.getValue(id)))
        },
        enableSorting: false,
        size: 120,
        meta: { mobileOrder: 20 },
      },
      {
        id: 'invite_info',
        header: t('Invite Info'),
        cell: ({ row }) => {
          const user = row.original
          const affCount = user.aff_count || 0
          const affHistoryQuota = user.aff_history_quota || 0
          const inviterId = user.inviter_id || 0

          if (affCount === 0 && affHistoryQuota === 0 && inviterId === 0) {
            return <span className='text-muted-foreground text-sm'>—</span>
          }

          return (
            <div
              data-table-text='secondary'
              className='min-w-0 space-y-1 text-xs font-normal'
            >
              {(affCount > 0 || affHistoryQuota !== 0) && (
                <LongText>
                  {t('Invited {{count}} users', { count: affCount })} ·{' '}
                  {t('Earnings')}:{' '}
                  <span className='tabular-nums'>
                    {formatQuota(affHistoryQuota)}
                  </span>
                </LongText>
              )}
              {inviterId > 0 && (
                <LongText className='text-muted-foreground'>
                  {t('Inviter ID: {{id}}', { id: inviterId })}
                </LongText>
              )}
            </div>
          )
        },
        size: 240,
        enableSorting: false,
        meta: { mobileHidden: true },
      },
      {
        accessorKey: 'created_at',
        header: t('Time'),
        cell: ({ row }) => (
          <ActivityTimeCell
            createdAt={row.original.created_at ?? 0}
            lastAt={row.original.last_login_at ?? 0}
            lastLabel={t('Last Login')}
            format='absolute'
          />
        ),
        size: 260,
        minSize: 240,
        meta: { mobileHidden: true },
      },
      {
        id: 'actions',
        header: () => t('Actions'),
        cell: ({ row }) => <DataTableRowActions row={row} tags={tags} />,
        meta: { pinned: 'right' as const },
      },
    ],
    // formatQuota reads the currency configuration from the store.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, quotaUnit, currencyConfig, tags, setOpen, setCurrentRow, viewer]
  )
}
