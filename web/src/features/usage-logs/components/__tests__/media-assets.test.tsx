import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { MediaAssetsTable } from '../media-assets-table'

const response = {
  success: true,
  data: {
    items: [
      {
        asset_id: 'asset_preview',
        user_id: 7,
        source_type: 'base64' as const,
        media_type: 'image' as const,
        mime_type: 'image/png',
        url: 'https://api.example.test/api/media-assets/asset_preview/content',
        size: 1024,
        created_at: 1_790_000_000,
        expires_at: 1_790_086_400,
      },
    ],
    total: 1,
  },
}

function mount() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MediaAssetsTable />
    </QueryClientProvider>
  )
}

afterEach(() => {
  vi.restoreAllMocks()
  useAuthStore.setState(useAuthStore.getInitialState(), true)
})

test('regular users can preview their media without delete controls', async () => {
  useAuthStore.getState().auth.setUser({ id: 7, username: 'user', role: 1 })
  vi.spyOn(api, 'get').mockResolvedValue({ data: response })

  mount()

  expect(await screen.findByText(/Base64 media/)).toBeVisible()
  expect(screen.getByRole('button', { name: /Preview/ })).toBeVisible()
  expect(
    screen.queryByRole('button', { name: /Delete/ })
  ).not.toBeInTheDocument()

  fireEvent.click(screen.getByRole('button', { name: /Preview/ }))
  const dialog = await screen.findByRole('dialog')
  expect(within(dialog).getByRole('img')).toHaveAttribute(
    'src',
    response.data.items[0].url
  )
  expect(within(dialog).getByText(response.data.items[0].url)).toBeVisible()
})

test('admins have a delete action and must confirm it', async () => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
  vi.spyOn(api, 'get').mockResolvedValue({ data: response })

  mount()

  fireEvent.click(await screen.findByRole('button', { name: /Delete/ }))
  const dialog = await screen.findByRole('alertdialog')
  expect(
    within(dialog).getByText(
      'This will permanently remove the selected media asset.'
    )
  ).toBeVisible()
})
