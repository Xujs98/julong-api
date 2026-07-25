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
import {
  ChevronLeft,
  ChevronRight,
  ChevronsUpDown,
  Loader2,
  X,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useDebounce } from '@/hooks/use-debounce'

import {
  resolveEmailCampaignUsers,
  searchEmailCampaignUsers,
  type EmailCampaignUserOption,
} from './email-campaigns-api'

const USER_PAGE_SIZE = 20
const MAX_VISIBLE_SELECTED_USERS = 8
const EMPTY_USERS: EmailCampaignUserOption[] = []

type EmailCampaignUserPickerProps = {
  selectedUserIds: number[]
  onChange: (userIds: number[]) => void
}

function userLabel(user: EmailCampaignUserOption) {
  return user.display_name || user.username || `#${user.id}`
}

export function EmailCampaignUserPicker(props: EmailCampaignUserPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const debouncedSearch = useDebounce(search.trim(), 300)
  const selectedSet = useMemo(
    () => new Set(props.selectedUserIds),
    [props.selectedUserIds]
  )

  const searchQuery = useQuery({
    queryKey: ['email-campaign-user-search', debouncedSearch, page],
    enabled: open,
    queryFn: async () => {
      const response = await searchEmailCampaignUsers(
        debouncedSearch,
        page,
        USER_PAGE_SIZE
      )
      if (!response.success) {
        throw new Error(response.message || t('Failed to search users'))
      }
      return (
        response.data || {
          page,
          page_size: USER_PAGE_SIZE,
          total: 0,
          items: [],
        }
      )
    },
    placeholderData: (previous) => previous,
  })

  const searchUsers = searchQuery.data?.items ?? EMPTY_USERS
  const selectedUsersQuery = useQuery({
    queryKey: ['email-campaign-selected-users', props.selectedUserIds],
    enabled: props.selectedUserIds.length > 0,
    queryFn: async () => {
      const response = await resolveEmailCampaignUsers(props.selectedUserIds)
      if (!response.success) {
        throw new Error(response.message || t('Failed to load users'))
      }
      return response.data || []
    },
  })

  const knownUsers = useMemo(() => {
    const users = new Map<number, EmailCampaignUserOption>()
    for (const user of selectedUsersQuery.data || []) users.set(user.id, user)
    for (const user of searchUsers) users.set(user.id, user)
    return users
  }, [searchUsers, selectedUsersQuery.data])
  const totalPages = Math.max(
    1,
    Math.ceil((searchQuery.data?.total || 0) / USER_PAGE_SIZE)
  )

  const toggleUser = (userID: number) => {
    if (selectedSet.has(userID)) {
      props.onChange(props.selectedUserIds.filter((id) => id !== userID))
      return
    }
    props.onChange([...props.selectedUserIds, userID])
  }

  const visibleSelectedIds = props.selectedUserIds.slice(
    0,
    MAX_VISIBLE_SELECTED_USERS
  )
  const hiddenSelectedCount =
    props.selectedUserIds.length - visibleSelectedIds.length

  return (
    <div className='space-y-2'>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              id='campaign-user-picker'
              type='button'
              variant='outline'
              role='combobox'
              aria-expanded={open}
              aria-label={t('Selected users')}
              className='h-10 w-full justify-between px-3 font-normal'
            />
          }
        >
          <span className='flex min-w-0 items-center gap-2'>
            <span className='truncate'>
              {props.selectedUserIds.length > 0
                ? t('Selected users')
                : t('Select')}
            </span>
            {props.selectedUserIds.length > 0 && (
              <Badge variant='secondary' className='rounded-sm px-1.5'>
                {props.selectedUserIds.length}
              </Badge>
            )}
          </span>
          <ChevronsUpDown className='size-4 shrink-0 opacity-50' />
        </PopoverTrigger>

        <PopoverContent
          align='start'
          className='w-[min(36rem,calc(100vw-2rem))] gap-0 overflow-hidden p-0'
          onWheel={(event) => event.stopPropagation()}
          onTouchMove={(event) => event.stopPropagation()}
        >
          <PopoverHeader className='border-b px-3 py-2.5'>
            <PopoverTitle className='flex items-center justify-between gap-3'>
              <span>{t('Selected users')}</span>
              <Badge variant='outline' className='rounded-sm'>
                {props.selectedUserIds.length}
              </Badge>
            </PopoverTitle>
          </PopoverHeader>
          <Command shouldFilter={false}>
            <CommandInput
              value={search}
              placeholder={`${t('User ID')} / ${t('Username')} / ${t('Email')}`}
              onValueChange={(value) => {
                setSearch(value)
                setPage(1)
              }}
            />
            <CommandList className='h-80 max-h-80'>
              {searchQuery.isFetching && searchUsers.length === 0 ? (
                <div className='text-muted-foreground flex h-24 items-center justify-center'>
                  <Loader2 className='size-4 animate-spin' />
                </div>
              ) : (
                <>
                  <CommandEmpty>
                    {t(
                      'No users available. Try adjusting your search or filters.'
                    )}
                  </CommandEmpty>
                  {searchUsers.map((user) => {
                    const checked = selectedSet.has(user.id)
                    return (
                      <CommandItem
                        key={user.id}
                        value={String(user.id)}
                        onSelect={() => toggleUser(user.id)}
                        className='items-start gap-3 px-3 py-2.5'
                      >
                        <Checkbox
                          checked={checked}
                          tabIndex={-1}
                          aria-hidden='true'
                          className='pointer-events-none mt-0.5'
                        />
                        <span className='min-w-0 flex-1'>
                          <span className='flex min-w-0 items-center gap-2'>
                            <span className='truncate font-medium'>
                              {userLabel(user)}
                            </span>
                            <span className='text-muted-foreground shrink-0 font-mono text-xs'>
                              #{user.id}
                            </span>
                          </span>
                          <span className='text-muted-foreground block truncate text-xs'>
                            @{user.username} | {user.email}
                          </span>
                        </span>
                      </CommandItem>
                    )
                  })}
                </>
              )}
            </CommandList>
          </Command>
          <div className='bg-muted/30 flex items-center justify-between border-t px-3 py-2'>
            <span className='text-muted-foreground text-xs'>
              {t('Total')} {searchQuery.data?.total || 0}
            </span>
            <div className='flex items-center gap-2'>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Previous')}
                disabled={page <= 1 || searchQuery.isFetching}
                onClick={() => setPage((value) => Math.max(1, value - 1))}
              >
                <ChevronLeft className='size-4' />
              </Button>
              <span className='text-muted-foreground min-w-14 text-center text-xs tabular-nums'>
                {page} / {totalPages}
              </span>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Next')}
                disabled={page >= totalPages || searchQuery.isFetching}
                onClick={() => setPage((value) => value + 1)}
              >
                <ChevronRight className='size-4' />
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>

      {props.selectedUserIds.length > 0 && (
        <div className='flex min-h-6 flex-wrap items-center gap-1.5'>
          {visibleSelectedIds.map((userID) => {
            const user = knownUsers.get(userID)
            const label = user ? userLabel(user) : `#${userID}`
            return (
              <Badge
                key={userID}
                variant='secondary'
                className='max-w-52 rounded-sm pr-0.5'
              >
                <span className='truncate'>{label}</span>
                <button
                  type='button'
                  className='hover:bg-muted-foreground/15 flex size-4 shrink-0 items-center justify-center rounded-sm'
                  aria-label={`${t('Remove')} ${label}`}
                  onClick={() => toggleUser(userID)}
                >
                  <X className='size-3' />
                </button>
              </Badge>
            )
          })}
          {hiddenSelectedCount > 0 && (
            <Badge variant='outline' className='rounded-sm'>
              +{hiddenSelectedCount}
            </Badge>
          )}
        </div>
      )}
    </div>
  )
}
