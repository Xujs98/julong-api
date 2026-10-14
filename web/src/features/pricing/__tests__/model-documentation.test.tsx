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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { ModelDetailsApi } from '../components/model-details-api'
import type { ModelDocumentation, PricingModel } from '../types'

const model: PricingModel = {
  id: 1,
  model_name: 'video-demo',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['default'],
  supported_endpoint_types: ['openai'],
}
const fence = String.fromCharCode(96).repeat(3)
const document: ModelDocumentation = {
  id: 2,
  name: 'Video API guide',
  format: 'markdown',
  original_filename: 'video.md',
  content: [
    '# Video API',
    '',
    'Use **Bearer** authentication.',
    '',
    '- Submit a task',
    '- Poll for completion',
    '',
    '| Field | Value |',
    '| --- | --- |',
    '| resolution | 720p |',
    '',
    `${fence}json`,
    '{',
    '  "model": "video-demo"',
    '}',
    fence,
    '',
    '[Docs](https://api.example.com/docs)',
  ].join('\n'),
  created_time: 1,
  updated_time: 1,
}
const endpointMap = { openai: { path: '/v1/chat/completions' } }
const clients: QueryClient[] = []

function renderApi(initialModel: PricingModel = model) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  const view = render(
    <QueryClientProvider client={client}>
      <ModelDetailsApi model={initialModel} endpointMap={endpointMap} />
    </QueryClientProvider>
  )
  return {
    ...view,
    showModel(nextModel: PricingModel) {
      view.rerender(
        <QueryClientProvider client={client}>
          <ModelDetailsApi model={nextModel} endpointMap={endpointMap} />
        </QueryClientProvider>
      )
    },
  }
}

beforeEach(() => {
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/status') return { data: { success: true } }
    return { data: { success: true, data: document } }
  })
})
afterEach(() => {
  for (const client of clients.splice(0)) client.clear()
  vi.restoreAllMocks()
})

test('selected Markdown replaces generated snippets and renders headings, lists, tables, fences, and links', async () => {
  renderApi()

  expect(
    await screen.findByRole('heading', { name: 'Video API' })
  ).toBeVisible()
  const section = screen.getByRole('region', { name: 'Code samples' })
  expect(within(section).getByText('Submit a task')).toBeVisible()
  expect(within(section).getByRole('cell', { name: '720p' })).toBeVisible()
  expect(section.querySelector('pre code')).toHaveTextContent(
    '"model": "video-demo"'
  )
  expect(section.querySelector('pre code')?.textContent).toContain('\n')
  expect(within(section).getByRole('link', { name: 'Docs' })).toHaveAttribute(
    'rel',
    'noopener noreferrer'
  )
  expect(screen.queryByRole('tab', { name: 'Python' })).not.toBeInTheDocument()
  expect(api.get).toHaveBeenCalledWith('/api/model-documentation', {
    params: { model: 'video-demo' },
    signal: expect.any(AbortSignal),
  })
})

test('selected document copy button copies the original Markdown rather than flattened rendered text', async () => {
  const user = userEvent.setup()
  renderApi()

  await user.click(await screen.findByRole('button', { name: 'Copy document' }))

  await waitFor(async () => {
    expect(await navigator.clipboard.readText()).toBe(document.content)
  })
  expect(screen.getByRole('button', { name: 'Copied' })).toBeVisible()
})

test('Word-derived normalized Markdown is displayed with the same structured preview', async () => {
  vi.mocked(api.get).mockResolvedValue({
    data: {
      success: true,
      data: {
        ...document,
        format: 'docx',
        original_filename: 'guide.docx',
        content:
          '# Word guide\n\n| Parameter | Notes |\n| --- | --- |\n| model | Required |',
      },
    },
  })
  renderApi()

  expect(
    await screen.findByRole('heading', { name: 'Word guide' })
  ).toBeVisible()
  expect(screen.getByRole('cell', { name: 'Required' })).toBeVisible()
})

test('document preview sanitizes scripts, event handlers, and unsafe links without executing uploaded markup', async () => {
  vi.mocked(api.get).mockResolvedValue({
    data: {
      success: true,
      data: {
        ...document,
        content:
          '# Safe guide\n\n<script>alert(1)</script>\n<img src="https://example.com/test.png" onerror="alert(1)">\n[Unsafe](javascript:alert(1))',
      },
    },
  })
  renderApi()

  await screen.findByRole('heading', { name: 'Safe guide' })
  const section = screen.getByRole('region', { name: 'Code samples' })
  expect(section.querySelector('script')).toBeNull()
  expect(section.querySelector('[onerror]')).toBeNull()
  expect(section.querySelector('[href^="javascript:"]')).toBeNull()
})

test('unbound model keeps the existing generated language snippets after documentation resolves to null', async () => {
  vi.mocked(api.get).mockResolvedValue({ data: { success: true, data: null } })
  renderApi()

  expect(await screen.findByRole('tab', { name: 'Python' })).toBeVisible()
  expect(screen.queryByRole('button', { name: 'Copy document' })).toBeNull()
})

test('pending template shows loading instead of misleading generated code and then resolves to the chosen document', async () => {
  let resolve!: (value: unknown) => void
  vi.mocked(api.get).mockImplementation(async (url) => {
    if (url === '/api/status') return { data: { success: true } }
    return new Promise((done) => {
      resolve = done
    })
  })
  renderApi()

  expect(screen.getByRole('status')).toHaveTextContent('Loading...')
  expect(screen.queryByRole('tab', { name: 'Python' })).toBeNull()
  resolve({ data: { success: true, data: document } })
  expect(
    await screen.findByRole('heading', { name: 'Video API' })
  ).toBeVisible()
})

test('failed documentation request shows the server reason and retry recovers without silently showing generic code', async () => {
  const user = userEvent.setup()
  vi.mocked(api.get).mockResolvedValueOnce({
    data: {
      success: false,
      message: 'Document source is temporarily unavailable',
    },
  })
  renderApi()

  expect(
    await screen.findByText('Document source is temporarily unavailable')
  ).toBeVisible()
  expect(screen.queryByRole('tab', { name: 'Python' })).toBeNull()
  await user.click(screen.getByRole('button', { name: 'Retry' }))
  expect(
    await screen.findByRole('heading', { name: 'Video API' })
  ).toBeVisible()
})

test('switching models never shows the previous model document as the new model example', async () => {
  const view = renderApi()
  await screen.findByRole('heading', { name: 'Video API' })
  vi.mocked(api.get).mockResolvedValue({ data: { success: true, data: null } })

  view.showModel({ ...model, model_name: 'text-demo' })

  expect(screen.queryByRole('heading', { name: 'Video API' })).toBeNull()
  expect(await screen.findByRole('tab', { name: 'Python' })).toBeVisible()
})

test('a selected guide is available even when the model has no inferred endpoint path', async () => {
  renderApi({ ...model, supported_endpoint_types: [] })

  expect(
    await screen.findByRole('heading', { name: 'Video API' })
  ).toBeVisible()
  expect(screen.getByRole('button', { name: 'Copy document' })).toBeVisible()
})

test('long document names stay truncated with full title and long content has a bounded scroll region', async () => {
  const longName = 'Long guide '.repeat(10).trim()
  vi.mocked(api.get).mockResolvedValue({
    data: {
      success: true,
      data: {
        ...document,
        name: longName,
        content: document.content,
      },
    },
  })
  renderApi()

  await screen.findByRole('heading', { name: 'Video API' })
  const name = screen.getByTitle(longName)
  expect(name).toHaveClass('truncate')
  const markdown = screen
    .getByRole('heading', { name: 'Video API' })
    .closest('.prose')
  expect(markdown?.parentElement).toHaveClass(
    'max-h-[70vh]',
    'overflow-y-auto',
    'min-w-0'
  )
  expect(markdown).toHaveClass(
    '[&_pre]:overflow-x-auto',
    '[&_table]:overflow-x-auto'
  )
})
