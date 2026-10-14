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
/* oxlint-disable react/incompatible-library -- this harness intentionally observes react-hook-form state. */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { Form } from '@/components/ui/form'
import { api } from '@/lib/api'

import { ModelDocumentTemplateField } from '../components/model-document-template-field'
import {
  transformFormDataToModelPayload,
  transformModelToFormDefaults,
  type ModelFormInput,
} from '../lib/model-form'
import type { Model } from '../types'

const defaults = (documentTemplateId = 0): ModelFormInput => ({
  model_name: 'example-model',
  description: '',
  icon: '',
  tags: [],
  endpoints: '',
  document_template_id: documentTemplateId,
  name_rule: 0,
  status: true,
  sync_official: true,
  enable_groups: [],
  quota_types: [],
})

function renderField(documentTemplateId = 0) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  function Harness() {
    const form = useForm<ModelFormInput>({
      defaultValues: defaults(documentTemplateId),
    })
    return (
      <Form {...form}>
        <ModelDocumentTemplateField control={form.control} />
        <output data-testid='selected-template'>
          {/* oxlint-disable-line react/incompatible-library -- watch is the observable assertion for this form harness. */}
          {String(form.watch('document_template_id') ?? 0)}
        </output>
      </Form>
    )
  }
  render(
    <QueryClientProvider client={client}>
      <Harness />
    </QueryClientProvider>
  )
  return client
}

function template(id: number, name: string) {
  return {
    id,
    name,
    format: 'markdown' as const,
    original_filename: `${name}.md`,
    created_time: 1,
    updated_time: 1,
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('model document template field', () => {
  it('defaults new models to generated examples and preserves the selected id in update payloads', () => {
    const freshModel = transformModelToFormDefaults({
      id: 0,
      model_name: 'new-model',
      status: 1,
      sync_official: 1,
      name_rule: 0,
    } as Model)
    expect(freshModel.document_template_id).toBe(0)
    const existingModel = transformModelToFormDefaults({
      id: 3,
      model_name: 'existing-model',
      document_template_id: 42,
      status: 1,
      sync_official: 1,
      name_rule: 0,
    } as Model)
    expect(existingModel.document_template_id).toBe(42)
    expect(transformFormDataToModelPayload(existingModel)).toMatchObject({
      id: 3,
      document_template_id: 42,
    })
  })

  it('loads all pages, saves a template, and supports clearing it', async () => {
    const get = vi
      .spyOn(api, 'get')
      .mockImplementation(async (_url, config) => {
        const page = Number((config as { params?: { p?: number } })?.params?.p)
        const item =
          page === 2 ? template(2, 'Video API') : template(1, 'Chat API')
        return {
          data: {
            success: true,
            data: { items: [item], total: 2, page, page_size: 1 },
          },
        }
      })
    renderField()
    const user = userEvent.setup()
    const input = await screen.findByRole('combobox', {
      name: 'Document template',
    })
    await user.click(input)
    expect(
      await screen.findByRole('option', { name: 'Video API' })
    ).toBeVisible()
    expect(get).toHaveBeenCalledTimes(2)
    await user.click(screen.getByRole('option', { name: 'Video API' }))
    expect(screen.getByTestId('selected-template')).toHaveTextContent('2')
    await user.click(input)
    await user.click(
      screen.getByRole('option', { name: 'Use generated code samples' })
    )
    expect(screen.getByTestId('selected-template')).toHaveTextContent('0')
  })

  it('keeps an existing deleted binding visible and warns the administrator', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          items: [template(1, 'Current template')],
          total: 1,
          page: 1,
          page_size: 100,
        },
      },
    })
    renderField(99)
    expect(
      await screen.findByText(
        'Selected document template no longer exists. Choose another template or clear the selection.'
      )
    ).toBeVisible()
    expect(
      screen.getByRole('combobox', { name: 'Document template' })
    ).toHaveValue('Missing document template')
    expect(screen.getByTestId('selected-template')).toHaveTextContent('99')
  })

  it('allows retry after a catalog request fails', async () => {
    const get = vi
      .spyOn(api, 'get')
      .mockRejectedValueOnce(new Error('catalog unavailable'))
      .mockResolvedValue({
        data: {
          success: true,
          data: {
            items: [template(1, 'Retry template')],
            total: 1,
            page: 1,
            page_size: 100,
          },
        },
      })
    renderField()
    expect(await screen.findByText('catalog unavailable')).toBeVisible()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => expect(get).toHaveBeenCalledTimes(2))
    expect(
      await screen.findByRole('combobox', { name: 'Document template' })
    ).toBeEnabled()
  })
})
