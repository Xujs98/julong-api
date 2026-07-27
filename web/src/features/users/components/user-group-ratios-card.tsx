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

import { GroupBadge } from '@/components/group-badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatQuota, formatTokens } from '@/lib/format'

import type { UserGroupUsage } from '../types'
import {
  formatUserGroupRatio,
  getSortedUserGroupUsage,
  userGroupRatiosCardClassName,
} from './user-group-ratios-card-layout'

export function UserGroupRatiosCard(props: {
  groupUsage: Record<string, UserGroupUsage>
}) {
  const { t } = useTranslation()
  const groupUsage = getSortedUserGroupUsage(props.groupUsage)

  if (groupUsage.length === 0) return null

  return (
    <section
      className={userGroupRatiosCardClassName}
      aria-label={t('Actual billing ratios')}
    >
      <div className='text-muted-foreground mb-2 text-xs font-medium'>
        {t('Actual billing ratios')}
      </div>
      <div className='flex flex-wrap gap-2'>
        {groupUsage.map((item) => (
          <Tooltip key={item.group}>
            <TooltipTrigger
              render={
                <div className='bg-background inline-flex h-7 cursor-help items-center gap-2 rounded-md border px-2' />
              }
            >
              <GroupBadge group={item.group} size='sm' type='text' />
              <span className='bg-info inline-flex h-5 items-center rounded-full px-1.5 font-mono text-xs leading-none font-semibold text-white tabular-nums'>
                {formatUserGroupRatio(item.ratio)}x
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <div className='space-y-1 text-xs'>
                <div>
                  {t('Used:')} {formatQuota(item.quota)}
                </div>
                <div>
                  {t('Tokens')}: {formatTokens(item.tokenUsed)}
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </section>
  )
}
