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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { InvoiceSettingsSection } from '../invoice-settings-section'

function Fixture() {
  const [container, setContainer] = useState<HTMLDivElement | null>(null)
  return (
    <>
      <div ref={setContainer} />
      <SettingsPageProvider actionsContainer={container}>
        <InvoiceSettingsSection
          defaultValues={{
            enabled: true,
            minimumAmount: 300,
            unit: 'USD',
            processingDays: '1-3',
            rejectionFreezeHours: 72,
            allowedSources: [
              'online_recharge',
              'redemption',
              'admin_adjustment',
            ],
          }}
        />
      </SettingsPageProvider>
    </>
  )
}

async function renderSettings() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  const router = createRouter({
    routeTree: createRootRoute({ component: Fixture }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole('checkbox', { name: 'Redemption cards' })
  return invalidate
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
})

test('saves eligible source selection as a JSON array and refreshes invoices', async () => {
  const user = userEvent.setup()
  const invalidate = await renderSettings()
  const redemption = screen.getByRole('checkbox', { name: 'Redemption cards' })
  expect(redemption).toBeChecked()
  await user.click(redemption)
  await user.click(
    screen.getByRole('button', { name: 'Save invoice settings' })
  )
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith(
      '/api/option/',
      {
        key: 'invoice_setting.allowed_sources',
        value: '["admin_adjustment","online_recharge"]',
      },
      { params: undefined }
    )
  )
  expect(api.put).toHaveBeenCalledTimes(1)
  expect(invalidate).toHaveBeenCalledWith({ queryKey: ['invoice'] })
})

test('allows an empty list to disable all invoice sources', async () => {
  const user = userEvent.setup()
  await renderSettings()
  for (const checkbox of screen.getAllByRole('checkbox')) {
    await user.click(checkbox)
  }
  await user.click(
    screen.getByRole('button', { name: 'Save invoice settings' })
  )
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith(
      '/api/option/',
      { key: 'invoice_setting.allowed_sources', value: '[]' },
      { params: undefined }
    )
  )
})
