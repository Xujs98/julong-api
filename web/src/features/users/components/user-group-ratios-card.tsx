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
  formatUserGroupRatio,
  getSortedUserGroupRatios,
  userGroupRatiosCardClassName,
} from './user-group-ratios-card-layout'

export function UserGroupRatiosCard(props: {
  groupRatios: Record<string, number>
}) {
  const { t } = useTranslation()
  const groupRatios = getSortedUserGroupRatios(props.groupRatios)

  if (groupRatios.length === 0) return null

  return (
    <section
      className={userGroupRatiosCardClassName}
      aria-label={t('Actual billing ratios')}
    >
      <div className='text-muted-foreground mb-2 text-xs font-medium'>
        {t('Actual billing ratios')}
      </div>
      <div className='flex flex-wrap gap-2'>
        {groupRatios.map((item) => (
          <div
            key={item.group}
            className='bg-background inline-flex h-7 items-center gap-2 rounded-md border px-2'
          >
            <GroupBadge group={item.group} size='sm' type='text' />
            <span className='bg-info inline-flex h-5 items-center rounded-full px-1.5 font-mono text-xs leading-none font-semibold text-white tabular-nums'>
              {formatUserGroupRatio(item.ratio)}x
            </span>
          </div>
        ))}
      </div>
    </section>
  )
}
