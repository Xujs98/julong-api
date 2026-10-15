import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { SidebarProvider } from '@/components/ui/sidebar'
import { SearchProvider } from '@/context/search-provider'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { AppHeader } from '../app-header'

afterEach(() => useAuthStore.getState().auth.reset())
test('header places accessible customer support between search and notifications', async () => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'fixture', role: 1 })
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: url === '/api/status' ? { version: 'v1.0.0' } : [],
    },
  }))
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => (
        <SidebarProvider>
          <SearchProvider>
            <AppHeader
              showConfigDrawer={false}
              showProfileDropdown={false}
              showTopNav={false}
            />
          </SearchProvider>
        </SidebarProvider>
      ),
    }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  const support = await screen.findByRole('button', { name: 'Contact Support' })
  const search = screen.getByRole('button', { name: 'Search' })
  const notifications = screen.getByRole('button', { name: 'Notifications' })
  expect(
    search.compareDocumentPosition(support) & Node.DOCUMENT_POSITION_FOLLOWING
  ).toBeTruthy()
  expect(
    support.compareDocumentPosition(notifications) &
      Node.DOCUMENT_POSITION_FOLLOWING
  ).toBeTruthy()
  expect(support).toHaveAttribute('aria-haspopup', 'dialog')
})
