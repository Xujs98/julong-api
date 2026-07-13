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
import { Megaphone } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { useStatus } from '@/hooks/use-status'
import { formatDateTimeObject } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

type Announcement = {
  id?: number | string
  content?: string
  extra?: string
  publishDate?: string | Date
}

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
  const { status } = useStatus()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const [open, setOpen] = useState(false)
  const popupEnabled = status?.announcements_popup_enabled === true
  const announcements = useMemo(
    () => ((status?.announcements || []) as Announcement[]).slice(0, 20),
    [status?.announcements]
  )
  const signature = useMemo(
    () => hashAnnouncements(announcements),
    [announcements]
  )
  const storageKey = `announcement-popup:${userId || 'anonymous'}`

  useEffect(() => {
    if (!popupEnabled || announcements.length === 0) {
      setOpen(false)
      return
    }
    setOpen(window.localStorage.getItem(storageKey) !== signature)
  }, [announcements.length, popupEnabled, signature, storageKey])

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
              <RichContent breaks content={announcement.content || ''} />
              {announcement.extra ? (
                <div className='text-muted-foreground text-sm'>
                  <RichContent breaks content={announcement.extra} />
                </div>
              ) : null}
              {announcement.publishDate ? (
                <time className='text-muted-foreground block text-xs'>
                  {formatDateTimeObject(new Date(announcement.publishDate))}
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
