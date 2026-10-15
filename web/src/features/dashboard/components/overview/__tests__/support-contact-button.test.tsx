import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { SupportContactButton } from '../support-contact-button'

function renderButton() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <SupportContactButton compact />
    </QueryClientProvider>
  )
}
beforeEach(() => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'fixture', role: 1 })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: [
        { id: 'qq', type: 'qq', label: 'QQ support', value: '12345' },
        {
          id: 'g',
          type: 'wechat_group',
          label: 'Community',
          value: 'Group contact',
          qr_code_url: '/qr.png',
        },
      ],
    },
  })
})
afterEach(() => useAuthStore.getState().auth.reset())

test('compact button loads contacts on keyboard activation and shows group QR and copy controls', async () => {
  renderButton()
  expect(api.get).not.toHaveBeenCalled()
  const user = userEvent.setup()
  const button = screen.getByRole('button', { name: 'Contact Support' })
  expect(button).toHaveAttribute('aria-expanded', 'false')
  button.focus()
  await user.keyboard('{Enter}')
  const dialog = await screen.findByRole('dialog', { name: 'Contact Support' })
  expect(await within(dialog).findByText('Community')).toBeInTheDocument()
  expect(within(dialog).getByText('Group contact')).toBeInTheDocument()
  expect(
    within(dialog).getByRole('img', { name: 'Group QR code' })
  ).toHaveAttribute('src', '/qr.png')
  expect(
    within(dialog).getAllByRole('button', { name: 'Copy to clipboard' })
  ).toHaveLength(2)
  expect(within(dialog).getByRole('link')).toHaveAttribute(
    'href',
    'tencent://message/?uin=12345&Site=&Menu=yes'
  )
  await user.click(within(dialog).getByRole('button', { name: 'View QR code' }))
  expect(
    await screen.findByRole('dialog', { name: 'Community' })
  ).toBeInTheDocument()
})

test('empty response shows empty state but network failure shows retry instead', async () => {
  vi.mocked(api.get).mockResolvedValue({ data: { success: true, data: [] } })
  renderButton()
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: 'Contact Support' }))
  expect(
    await screen.findByText('No support contacts configured')
  ).toBeInTheDocument()
  await user.keyboard('{Escape}')
  vi.mocked(api.get).mockRejectedValue(new Error('offline'))
  await user.click(screen.getByRole('button', { name: 'Contact Support' }))
  expect(
    await screen.findByText('Failed to load support contacts')
  ).toBeInTheDocument()
  expect(
    screen.queryByText('No support contacts configured')
  ).not.toBeInTheDocument()
})

test('logged out header does not show a control backed by a login-only endpoint', () => {
  useAuthStore.getState().auth.reset()
  renderButton()
  expect(
    screen.queryByRole('button', { name: 'Contact Support' })
  ).not.toBeInTheDocument()
  expect(api.get).not.toHaveBeenCalled()
})
