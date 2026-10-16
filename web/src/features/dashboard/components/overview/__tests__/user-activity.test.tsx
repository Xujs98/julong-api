import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'

import { UserOnlineBadge } from '@/features/users/components/user-online-badge'
import { getUserStatusOptions } from '@/features/users/constants'
import { getUserQuotaSummaryParams } from '@/features/users/lib/user-quota-summary'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { UserActivityCards } from '../user-activity-cards'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  useAuthStore.getState().auth.reset()
})
function showCards(role: number) {
  useAuthStore.getState().auth.setUser({ id: 2, username: 'viewer', role })
  return render(
    <QueryClientProvider
      client={
        new QueryClient({
          defaultOptions: { queries: { retry: false, gcTime: 0 } },
        })
      }
    >
      <UserActivityCards />
    </QueryClientProvider>
  )
}
it.each([10, 100])(
  'shows real daily/online counts to an administrator (role %s)',
  async (role) => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: { daily_active_users: 23, online_users: 7 },
      },
    })
    showCards(role)
    expect(await screen.findByText('23')).toBeInTheDocument()
    expect(screen.getByText('7')).toBeInTheDocument()
    expect(
      screen.getByText('Unique active accounts today (UTC+8)')
    ).toBeInTheDocument()
    expect(screen.queryByText(/excluding Root/i)).not.toBeInTheDocument()
  }
)
it('does not fetch or expose site activity to ordinary users', () => {
  const get = vi.spyOn(api, 'get')
  const { container } = showCards(1)
  expect(container).toBeEmptyDOMElement()
  expect(get).not.toHaveBeenCalled()
})
it('does not disguise failed statistics as zero', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: false, message: 'unavailable' },
  })
  showCards(100)
  await waitFor(() => expect(screen.getAllByText('--')).toHaveLength(2))
  expect(screen.queryByText('0')).not.toBeInTheDocument()
})
it('shows green online and grey offline badges, and never a Root state', () => {
  const { container, rerender } = render(
    <UserOnlineBadge user={{ role: 1, is_online: true }} />
  )
  expect(screen.getByText('Online').closest('[data-slot]')).toHaveClass(
    'text-success'
  )
  rerender(<UserOnlineBadge user={{ role: 1, is_online: false }} />)
  expect(screen.getByText('Offline').closest('[data-slot]')).toHaveClass(
    'text-muted-foreground'
  )
  for (const state of [true, false, undefined]) {
    rerender(<UserOnlineBadge user={{ role: 100, is_online: state }} />)
    expect(container).toBeEmptyDOMElement()
  }
})
it('offers online/offline in the existing status filter and preserves the summary filter', () => {
  const options = getUserStatusOptions((key) => key)
  expect(options.map((o) => o.value)).toEqual(
    expect.arrayContaining(['online', 'offline'])
  )
  expect(getUserQuotaSummaryParams({ status: ['offline'] })).toMatchObject({
    status: 'offline',
  })
})

it('distinguishes API activity and simultaneous website/API activity', () => {
  const { rerender } = render(
    <UserOnlineBadge
      user={{ role: 1, is_online: true, online_source: 'api' }}
    />
  )
  expect(screen.getByText('Online · API')).toBeInTheDocument()
  rerender(
    <UserOnlineBadge
      user={{ role: 1, is_online: true, online_source: 'both' }}
    />
  )
  expect(screen.getByText('Online · Web + API')).toBeInTheDocument()
  expect(
    screen.getByText('Online · Web + API').closest('[data-slot]')
  ).toHaveClass('user-presence-glow')
  rerender(
    <UserOnlineBadge
      user={{ role: 1, is_online: false, online_source: 'both' }}
    />
  )
  expect(screen.getByText('Offline').closest('[data-slot]')).not.toHaveClass(
    'user-presence-glow'
  )
})
