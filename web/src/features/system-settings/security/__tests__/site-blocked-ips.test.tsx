import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { SiteBlockedIPs } from '../site-blocked-ips'

function renderList() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <SiteBlockedIPs />
    </QueryClientProvider>
  )
}
afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})
beforeEach(() => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        items: [
          {
            id: 1,
            ip: '198.51.100.44',
            user_id: 42,
            reason: 'disabled user request',
          },
        ],
        total: 1,
      },
    },
  })
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
})
test('automatic blocks are visible and unblocking requires confirmation then refreshes the list', async () => {
  renderList()
  const user = userEvent.setup()
  expect(await screen.findByText('198.51.100.44')).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Unblock' }))
  expect(api.put).not.toHaveBeenCalled()
  const dialog = await screen.findByRole('alertdialog')
  await user.click(within(dialog).getByRole('button', { name: 'Unblock' }))
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith('/api/option/site_blocked_ips', {
      ips: ['198.51.100.44'],
      blocked: false,
    })
  )
})
test('loading and API failure show recoverable states without pretending the blacklist is empty', async () => {
  vi.mocked(api.get).mockRejectedValue(new Error('offline'))
  renderList()
  expect(screen.getByText('Loading...')).toBeInTheDocument()
  expect(
    await screen.findByRole('button', { name: 'Retry' })
  ).toBeInTheDocument()
  expect(screen.queryByText('No Data')).not.toBeInTheDocument()
})
test('manual block submits an IP and a failed update keeps its confirmation open', async () => {
  vi.mocked(api.put).mockResolvedValue({
    data: { success: false, message: 'rejected' },
  })
  renderList()
  const user = userEvent.setup()
  await screen.findByText('198.51.100.44')
  await user.type(
    screen.getByRole('textbox', { name: 'IP Address' }),
    '203.0.113.7'
  )
  await user.click(screen.getByRole('button', { name: 'Block IP' }))
  const dialog = await screen.findByRole('alertdialog')
  await user.click(within(dialog).getByRole('button', { name: 'Block IP' }))
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith('/api/option/site_blocked_ips', {
      ips: ['203.0.113.7'],
      blocked: true,
    })
  )
  expect(screen.getByRole('alertdialog')).toBeInTheDocument()
})

test('empty list supports searching and clearing the IP filter', async () => {
  vi.mocked(api.get).mockResolvedValue({
    data: { success: true, data: { items: [], total: 0 } },
  })
  renderList()
  const user = userEvent.setup()
  await screen.findByText('No Data')
  expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  const input = screen.getByRole('textbox', { name: 'IP Address' })
  await user.type(input, '203.0.113.20')
  await user.click(screen.getByRole('button', { name: 'Search' }))
  await waitFor(() =>
    expect(api.get).toHaveBeenLastCalledWith('/api/option/site_blocked_ips', {
      params: { p: 1, page_size: 20, ip: '203.0.113.20' },
    })
  )
  await user.click(screen.getByRole('button', { name: 'Refresh' }))
  await waitFor(() =>
    expect(api.get).toHaveBeenLastCalledWith('/api/option/site_blocked_ips', {
      params: { p: 1, page_size: 20, ip: '' },
    })
  )
  expect(input).toHaveValue('')
})
