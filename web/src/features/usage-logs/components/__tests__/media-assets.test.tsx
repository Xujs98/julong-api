import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router'
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  within,
  waitFor,
} from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { MediaAssetsTable } from '../media-assets-table'
import { UsageLogsProvider, useUsageLogsContext } from '../usage-logs-provider'

const response = {
  success: true,
  data: {
    items: [
      {
        asset_id: 'asset_preview',
        user_id: 7,
        username: 'asset_alice',
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

function ScopeToggle() {
  const { setViewScope } = useUsageLogsContext()
  return (
    <button type='button' onClick={() => setViewScope('self')}>
      Only Mine
    </button>
  )
}

const clients: QueryClient[] = []

async function mount(initialSearch = '') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  const root = createRootRoute()
  const authenticated = createRoute({
    getParentRoute: () => root,
    id: '_authenticated',
  })
  const media = createRoute({
    getParentRoute: () => authenticated,
    path: '/usage-logs/$section',
    validateSearch: (search: Record<string, unknown>) => ({
      startTime:
        typeof search.startTime === 'number' ? search.startTime : undefined,
      endTime: typeof search.endTime === 'number' ? search.endTime : undefined,
      username:
        typeof search.username === 'string' ? search.username : undefined,
    }),
    component: () => (
      <UsageLogsProvider>
        <ScopeToggle />
        <MediaAssetsTable />
      </UsageLogsProvider>
    ),
  })
  const router = createRouter({
    routeTree: root.addChildren([authenticated.addChildren([media])]),
    history: createMemoryHistory({
      initialEntries: [['/usage-logs/assets', initialSearch].join('')],
    }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return router
}

afterEach(() => {
  cleanup()
  for (const client of clients.splice(0)) client.clear()
  vi.restoreAllMocks()
  useAuthStore.setState(useAuthStore.getInitialState(), true)
})

test('regular users can preview their media without delete controls', async () => {
  useAuthStore.getState().auth.setUser({ id: 7, username: 'user', role: 1 })
  vi.spyOn(api, 'get').mockResolvedValue({ data: response })

  await mount()

  expect(await screen.findByText(/Base64 media/)).toBeVisible()
  expect(
    screen.queryByRole('columnheader', { name: 'User' })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('textbox', { name: 'Filter by username' })
  ).not.toBeInTheDocument()
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
  expect(dialog).toHaveClass('overflow-x-hidden', 'sm:max-w-3xl')
  expect(within(dialog).getByRole('img')).toHaveClass(
    'max-h-[55dvh]',
    'max-w-full',
    'object-contain'
  )
  expect(
    within(dialog).getByRole('heading', { name: 'Media asset details' })
  ).toBeVisible()
  expect(
    within(dialog).getByRole('button', { name: 'Copy to clipboard' })
  ).toBeVisible()
  fireEvent.click(within(dialog).getByRole('button', { name: 'Close' }))
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  )
})

test('admins have a delete action and must confirm it', async () => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'admin',
    role: 10,
    permissions: { admin_permissions: { task_logs: { assets_read: true } } },
  })
  vi.spyOn(api, 'get').mockResolvedValue({ data: response })

  await mount()

  fireEvent.click(await screen.findByRole('button', { name: /Delete/ }))
  const dialog = await screen.findByRole('alertdialog')
  expect(
    within(dialog).getByText(
      'This will permanently remove the selected media asset.'
    )
  ).toBeVisible()
})

test('admin user column is separate, search applies on demand, dates use seconds and reset clears username', async () => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'admin',
    role: 10,
    permissions: { admin_permissions: { task_logs: { assets_read: true } } },
  })
  const get = vi.spyOn(api, 'get').mockResolvedValue({ data: response })
  const router = await mount('?startTime=100000&endTime=200000')
  await waitFor(() =>
    expect(screen.getByRole('columnheader', { name: 'User' })).toBeVisible()
  )
  const media = screen.getByRole('columnheader', { name: 'Media' })
  const row = screen.getByRole('button', { name: /asset_alice/ }).closest('tr')
  const headerRow = media.closest('tr')
  expect(row).not.toBeNull()
  expect(headerRow).not.toBeNull()
  if (!row || !headerRow) throw new Error('Expected table rows')
  const headers = within(headerRow).getAllByRole('columnheader')
  expect(
    within(row).getAllByRole('cell')[headers.indexOf(media)]
  ).not.toHaveTextContent('asset_alice')
  expect(get).toHaveBeenLastCalledWith('/api/media-assets', {
    params: expect.objectContaining({
      start_time: 100,
      end_time: 200,
      view_scope: 'all',
    }),
  })

  const input = screen.getByRole('textbox', { name: 'Filter by username' })
  fireEvent.change(input, { target: { value: 'alice' } })
  expect(get).toHaveBeenCalledTimes(1)
  fireEvent.click(screen.getByRole('button', { name: 'Search' }))
  await waitFor(() =>
    expect(get).toHaveBeenLastCalledWith('/api/media-assets', {
      params: expect.objectContaining({ username: 'alice' }),
    })
  )
  expect(router.state.location.search).toMatchObject({
    startTime: 100000,
    endTime: 200000,
    username: 'alice',
  })

  fireEvent.click(await screen.findByRole('button', { name: 'Reset' }))
  await waitFor(() =>
    expect(
      screen.getByRole('textbox', { name: 'Filter by username' })
    ).toHaveValue('')
  )
  await waitFor(() =>
    expect(get).toHaveBeenLastCalledWith('/api/media-assets', {
      params: expect.objectContaining({ username: undefined }),
    })
  )
})

test('view menu controls visible columns and self scope requests only owned media', async () => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'admin',
    role: 10,
    permissions: { admin_permissions: { task_logs: { assets_read: true } } },
  })
  const get = vi.spyOn(api, 'get').mockResolvedValue({ data: response })
  await mount()
  await waitFor(() =>
    expect(screen.getByRole('columnheader', { name: 'User' })).toBeVisible()
  )
  fireEvent.click(screen.getByRole('button', { name: 'View' }))
  fireEvent.click(await screen.findByRole('menuitemcheckbox', { name: 'User' }))
  expect(
    screen.queryByRole('columnheader', { name: 'User' })
  ).not.toBeInTheDocument()
  fireEvent.keyDown(screen.getByRole('menuitemcheckbox', { name: 'User' }), {
    key: 'Escape',
  })
  fireEvent.click(screen.getByRole('button', { name: 'Only Mine' }))
  await waitFor(() =>
    expect(get).toHaveBeenLastCalledWith('/api/media-assets', {
      params: expect.objectContaining({
        view_scope: 'self',
        username: undefined,
      }),
    })
  )
  expect(
    screen.queryByRole('textbox', { name: 'Filter by username' })
  ).not.toBeInTheDocument()
})

test('tightening Root privacy closes an open preview and drops cached rows before refetch completes', async () => {
  const viewer = {
    id: 1,
    username: 'admin',
    role: 10,
    permissions: { admin_permissions: { task_logs: { assets_read: true } } },
  }
  useAuthStore.getState().auth.setUser(viewer)
  const get = vi.spyOn(api, 'get').mockResolvedValue({ data: response })
  await mount()
  fireEvent.click(await screen.findByRole('button', { name: /Preview/ }))
  expect(await screen.findByRole('dialog')).toBeVisible()
  get.mockImplementation(() => new Promise(() => {}))
  act(() =>
    useAuthStore.getState().auth.setUser({
      ...viewer,
      permissions: {
        admin_permissions: {
          task_logs: { assets_read: true },
          privacy: { hide_root: true },
        },
      },
    })
  )
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  )
  expect(screen.queryByText('asset_alice')).not.toBeInTheDocument()
})
