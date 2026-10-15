import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { SupportContactsSection } from '../support-contacts-section'

const contact = {
  id: 'group',
  type: 'qq_group',
  label: 'Help group',
  value: '123456',
  qr_code_url: '',
}
function renderSettings() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <SupportContactsSection />
    </QueryClientProvider>
  )
}
beforeEach(() => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: [{ ...contact }] },
  })
  vi.spyOn(api, 'put').mockResolvedValue({
    data: {
      success: true,
      data: [
        {
          ...contact,
          qr_code_url: '/api/support-contacts/qrcode/qr-saved.png',
        },
      ],
    },
  })
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    value: vi.fn(() => 'blob:qr-preview'),
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    value: vi.fn(),
  })
})

test('group QR can be uploaded, previewed and saved with name and contact information', async () => {
  const user = userEvent.setup()
  renderSettings()
  const field = await screen.findByLabelText('Group QR code')
  const file = new File(['png'], 'qr.png', { type: 'image/png' })
  await user.upload(field, file)
  expect(screen.getByRole('img', { name: 'Group QR code' })).toHaveAttribute(
    'src',
    'blob:qr-preview'
  )
  await user.click(screen.getByRole('button', { name: 'View QR code' }))
  const preview = await screen.findByRole('dialog', { name: 'Group QR code' })
  expect(within(preview).getByRole('img')).toHaveAttribute(
    'src',
    'blob:qr-preview'
  )
  await user.keyboard('{Escape}')
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  )
  await user.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(api.put).toHaveBeenCalled())
  const body = vi.mocked(api.put).mock.calls[0][1] as FormData
  expect(body.get('qrcode_group')).toBe(file)
  expect(JSON.parse(body.get('contacts') as string)).toEqual([contact])
  await waitFor(() =>
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/api/support-contacts/qrcode/qr-saved.png'
    )
  )
})

test('deleting saved QR keeps display name and contact value and saves an empty URL', async () => {
  vi.mocked(api.get).mockResolvedValue({
    data: { success: true, data: [{ ...contact, qr_code_url: '/qr.png' }] },
  })
  renderSettings()
  const user = userEvent.setup()
  await user.click(
    await screen.findByRole('button', { name: 'Delete QR code' })
  )
  expect(screen.queryByRole('img')).not.toBeInTheDocument()
  expect(screen.getByLabelText('Display name')).toHaveValue('Help group')
  expect(screen.getByLabelText('Contact information')).toHaveValue('123456')
  await user.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith('/api/support-contacts', {
      contacts: [contact],
    })
  )
})

test('both group types offer QR upload while switching to phone removes the draft image', async () => {
  renderSettings()
  const user = userEvent.setup()
  await user.upload(
    await screen.findByLabelText('Group QR code'),
    new File(['png'], 'a.png', { type: 'image/png' })
  )
  await user.click(screen.getByRole('combobox', { name: 'Contact type' }))
  await user.click(await screen.findByRole('option', { name: 'WeChat group' }))
  expect(screen.getByLabelText('Group QR code')).toBeInTheDocument()
  await user.click(screen.getByRole('combobox', { name: 'Contact type' }))
  await user.click(await screen.findByRole('option', { name: 'Phone' }))
  expect(screen.queryByLabelText('Group QR code')).not.toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith('/api/support-contacts', {
      contacts: [{ ...contact, type: 'phone' }],
    })
  )
})

test('failed load cannot save an empty replacement and exposes retry', async () => {
  vi.mocked(api.get).mockRejectedValue(new Error('offline'))
  renderSettings()
  expect(
    await screen.findByText('Failed to load support contacts')
  ).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Add contact' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
})

test('invalid file is not previewed and failed save preserves the selected file for retry', async () => {
  vi.mocked(api.put).mockResolvedValue({
    data: { success: false, message: 'Save failed' },
  })
  renderSettings()
  const field = await screen.findByLabelText('Group QR code')
  const user = userEvent.setup({ applyAccept: false })
  await user.upload(
    field,
    new File(['svg'], 'x.svg', { type: 'image/svg+xml' })
  )
  expect(screen.queryByRole('img')).not.toBeInTheDocument()
  await user.upload(field, new File(['png'], 'x.png', { type: 'image/png' }))
  await user.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
  )
  expect(screen.getByRole('img')).toHaveAttribute('src', 'blob:qr-preview')
})
