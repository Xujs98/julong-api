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
import { Megaphone } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { getUserAnnouncements } from '@/features/announcements/api'
import type { Announcement } from '@/features/announcements/types'
import { formatDateTimeObject } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

function hashAnnouncements(announcements: Announcement[]) {
  const input = JSON.stringify(announcements)
  let hash = 0
  for (let index = 0; index < input.length; index += 1) {
    hash = (hash << 5) - hash + input.charCodeAt(index)
    hash |= 0
  }
  return hash.toString(36)
}

export function AnnouncementPopup() {
  const { t } = useTranslation()
  const { data } = useQuery({
    queryKey: ['user-announcements'],
    queryFn: getUserAnnouncements,
    staleTime: 60 * 1000,
  })
  const userId = useAuthStore((state) => state.auth.user?.id)
  const [open, setOpen] = useState(false)
  const announcements = useMemo(
    () =>
      (data?.data || [])
        .filter((item) => item.notificationMode === 'popup')
        .slice(0, 20),
    [data?.data]
  )
  const signature = useMemo(
    () => hashAnnouncements(announcements),
    [announcements]
  )
  const storageKey = `announcement-popup:${userId || 'anonymous'}`

  useEffect(() => {
    if (announcements.length === 0) {
      setOpen(false)
      return
    }
    setOpen(window.localStorage.getItem(storageKey) !== signature)
  }, [announcements.length, signature, storageKey])

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      window.localStorage.setItem(storageKey, signature)
    }
    setOpen(nextOpen)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={handleOpenChange}
      title={
        <span className='flex items-center gap-2'>
          <Megaphone className='size-5' />
          {t('System Announcements')}
        </span>
      }
      contentClassName='sm:max-w-lg'
      contentHeight='min(60vh, 32rem)'
      footer={
        <Button type='button' onClick={() => handleOpenChange(false)}>
          {t('Close')}
        </Button>
      }
    >
      <div className='flex flex-col'>
        {announcements.map((announcement, index) => (
          <div
            key={
              announcement.id ??
              `announcement-${hashAnnouncements([announcement])}`
            }
          >
            <article className='space-y-2 py-3'>
              <h3 className='font-semibold'>{announcement.title}</h3>
              <RichContent breaks content={announcement.content || ''} />
              {announcement.startTime ? (
                <time className='text-muted-foreground block text-xs'>
                  {formatDateTimeObject(new Date(announcement.startTime))}
                </time>
              ) : null}
            </article>
            {index < announcements.length - 1 ? <Separator /> : null}
          </div>
        ))}
      </div>
    </Dialog>
  )
}
