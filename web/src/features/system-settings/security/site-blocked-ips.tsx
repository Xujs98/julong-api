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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

type BlockedIP = { id: number; ip: string; user_id: number; reason: string }
type BlockPage = { items: BlockedIP[]; total: number }
const queryKey = ['site-blocked-ips'] as const
const pageSize = 20

export function SiteBlockedIPs() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [page, setPage] = useState(1)
  const [input, setInput] = useState('')
  const [search, setSearch] = useState('')
  const [pending, setPending] = useState<{
    ip: string
    blocked: boolean
  } | null>(null)
  const list = useQuery({
    queryKey: [...queryKey, page, search],
    queryFn: async () => {
      const response = await api.get('/api/option/site_blocked_ips', {
        params: { p: page, page_size: pageSize, ip: search },
      })
      return requireServerSuccess(
        response.data as { success: boolean; data: BlockPage }
      ).data
    },
  })
  const update = useMutation({
    mutationFn: async (value: { ip: string; blocked: boolean }) => {
      const response = await api.put('/api/option/site_blocked_ips', {
        ips: [value.ip],
        blocked: value.blocked,
      })
      requireServerSuccess(response.data)
    },
    onSuccess: async () => {
      setPending(null)
      await client.invalidateQueries({ queryKey })
    },
  })
  const actionLabel = pending?.blocked ? t('Block IP') : t('Unblock')
  return (
    <div className='flex flex-col gap-4'>
      <div>
        <h3 className='font-semibold'>{t('Website access IP blacklist')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Blocked IPs cannot access any page or API. Shared IPs affect every user on that address. This list is separate from outbound SSRF rules.'
          )}
        </p>
      </div>
      <form
        className='flex flex-wrap items-end gap-2'
        onSubmit={(event) => {
          event.preventDefault()
          setPage(1)
          setSearch(input.trim())
        }}
      >
        <div className='min-w-0 flex-1 space-y-2'>
          <Label htmlFor='site-block-ip'>{t('IP Address')}</Label>
          <Input
            id='site-block-ip'
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder='203.0.113.1'
          />
        </div>
        <Button type='submit' variant='outline'>
          {t('Search')}
        </Button>
        <Button
          type='button'
          variant='outline'
          onClick={() => {
            setInput('')
            setSearch('')
            setPage(1)
            void list.refetch()
          }}
        >
          {t('Refresh')}
        </Button>
        <Button
          type='button'
          variant='destructive'
          disabled={!input.trim() || update.isPending}
          onClick={() => setPending({ ip: input.trim(), blocked: true })}
        >
          {t('Block IP')}
        </Button>
      </form>
      {list.isPending && <LoadingState />}
      {list.isError && (
        <ErrorState
          onRetry={() => {
            void list.refetch()
          }}
        />
      )}
      {list.isSuccess && list.data.items.length === 0 && <EmptyState />}
      {list.isSuccess && list.data.items.length > 0 && (
        <ul
          className='max-h-80 overflow-y-auto rounded-md border'
          aria-label={t('Website access IP blacklist')}
        >
          {list.data.items.map((item) => (
            <li
              key={item.id}
              className='flex flex-wrap items-center justify-between gap-3 border-b p-3 last:border-b-0'
            >
              <div className='min-w-0 break-all'>
                <p className='font-mono text-sm'>{item.ip}</p>
                <p className='text-muted-foreground text-xs'>
                  {t('User ID')}: {item.user_id || '—'} ·{' '}
                  {item.reason === 'manually blocked'
                    ? t('Manual block')
                    : t('Disabled account')}
                </p>
              </div>
              <Button
                type='button'
                size='sm'
                variant='outline'
                disabled={update.isPending}
                onClick={() => setPending({ ip: item.ip, blocked: false })}
              >
                {t('Unblock')}
              </Button>
            </li>
          ))}
        </ul>
      )}
      <div className='flex justify-end gap-2'>
        <Button
          type='button'
          variant='outline'
          disabled={page === 1 || list.isFetching}
          onClick={() => setPage((current) => current - 1)}
        >
          {t('Previous')}
        </Button>
        <Button
          type='button'
          variant='outline'
          disabled={
            !list.data || page * pageSize >= list.data.total || list.isFetching
          }
          onClick={() => setPage((current) => current + 1)}
        >
          {t('Next')}
        </Button>
      </div>
      <ConfirmDialog
        open={pending !== null}
        onOpenChange={(open) => {
          if (!open && !update.isPending) setPending(null)
        }}
        title={actionLabel}
        desc={pending?.ip ?? ''}
        confirmText={actionLabel}
        destructive={pending?.blocked}
        isLoading={update.isPending}
        handleConfirm={() => {
          if (pending) update.mutate(pending)
        }}
      />
    </div>
  )
}
