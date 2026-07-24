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
import { Bell, Check, Clock3 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { getUserAnnouncements } from '@/features/announcements/api'
import type { Announcement } from '@/features/announcements/types'
import { formatDateTimeObject } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

function hashAnnouncement(announcement: Announcement) {
  const input = JSON.stringify(announcement)
  let hash = 0
  for (let index = 0; index < input.length; index += 1) {
    hash = (hash << 5) - hash + input.charCodeAt(index)
    hash |= 0
  }
  return `${announcement.id}:${hash.toString(36)}`
}

function readStoredKeys(storageKey: string) {
  try {
    const value = JSON.parse(window.localStorage.getItem(storageKey) || '[]')
    return Array.isArray(value)
      ? value.filter((item) => typeof item === 'string')
      : []
  } catch {
    return []
  }
}

export function AnnouncementPopup() {
  const { t } = useTranslation()
  const { data } = useQuery({
    queryKey: ['user-announcements'],
    queryFn: getUserAnnouncements,
    staleTime: 60 * 1000,
  })
  const userId = useAuthStore((state) => state.auth.user?.id)
  const storageKey = `announcement-popup-read:${userId || 'anonymous'}`
  const [readKeys, setReadKeys] = useState<string[]>([])
  const [open, setOpen] = useState(false)
  const popupAnnouncements = useMemo(
    () =>
      (data?.data || [])
        .filter((item) => item.notificationMode === 'popup')
        .slice(0, 20),
    [data?.data]
  )
  const current = popupAnnouncements.find(
    (announcement) => !readKeys.includes(hashAnnouncement(announcement))
  )

  useEffect(() => {
    setReadKeys(readStoredKeys(storageKey))
  }, [storageKey])

  useEffect(() => {
    if (current) setOpen(true)
  }, [current])

  const markCurrentRead = () => {
    if (!current) return
    const nextKeys = [...new Set([...readKeys, hashAnnouncement(current)])]
    window.localStorage.setItem(storageKey, JSON.stringify(nextKeys))
    setReadKeys(nextKeys)
    setOpen(false)
  }

  if (!current) return null

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className='gap-0 overflow-hidden rounded-lg p-0 sm:max-w-2xl'>
        <DialogHeader className='border-b border-amber-200/70 bg-amber-50/70 p-6 text-left sm:p-8 dark:border-amber-900/60 dark:bg-amber-950/20'>
          <div className='mb-5 flex items-center gap-3'>
            <div className='flex size-11 items-center justify-center rounded-lg bg-orange-500 text-white shadow-sm'>
              <Bell className='size-5' />
            </div>
            <span className='inline-flex h-8 items-center gap-2 rounded-md bg-orange-500 px-3 text-sm font-medium text-white'>
              <span className='size-2 rounded-full bg-white ring-4 ring-white/30' />
              {t('Unread')}
            </span>
          </div>
          <DialogTitle className='text-2xl leading-tight font-semibold sm:text-3xl'>
            {current.title}
          </DialogTitle>
          <DialogDescription className='mt-3 flex items-center gap-2 text-sm sm:text-base'>
            <Clock3 className='size-4 shrink-0' />
            <span>{t('Just now')}</span>
            {current.startTime ? (
              <>
                <span aria-hidden='true'>·</span>
                <time>{formatDateTimeObject(new Date(current.startTime))}</time>
              </>
            ) : null}
          </DialogDescription>
        </DialogHeader>

        <div className='min-h-40 p-6 sm:p-8'>
          <div className='border-l-4 border-amber-500 py-1 pl-5 text-base leading-7 sm:text-lg'>
            <RichContent breaks content={current.content || ''} />
          </div>
        </div>

        <DialogFooter className='bg-muted/30 !m-0 !rounded-none border-t p-5 sm:p-6'>
          <Button
            type='button'
            onClick={markCurrentRead}
            className='min-w-40 bg-orange-500 text-white hover:bg-orange-600'
          >
            <Check className='size-4' />
            {t('Mark as read')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
