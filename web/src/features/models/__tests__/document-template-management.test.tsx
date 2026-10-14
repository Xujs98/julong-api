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
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { DocumentTemplateDialog } from '../components/dialogs/document-template-dialog'
import { DocumentTemplatesTable } from '../components/document-templates-table'
import type { DocumentTemplate } from '../document-templates-api'
import {
  documentUploadError,
  MAX_DOCUMENT_UPLOAD_BYTES,
} from '../lib/document-template-form'

const document: DocumentTemplate = {
  id: 8,
  name: 'Video API',
  format: 'markdown',
  original_filename: 'video-api.md',
  content:
    '# Video API\n\n| Field | Meaning |\n| --- | --- |\n| model | Name |\n\n```json\n{"model":"video"}\n```',
  created_time: 10,
  updated_time: 20,
}

function renderWithClient(component: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(<QueryClientProvider client={client}>{component}</QueryClientProvider>)
  return client
}

describe('document template management', () => {
  it('uploads Markdown with a inferred editable name and invalidates linked views only after success', async () => {
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true, data: document } })
    const close = vi.fn()
    const client = renderWithClient(
      <DocumentTemplateDialog open mode='upload' onOpenChange={close} />
    )
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    const user = userEvent.setup()
    const file = new File(['# Video API'], 'video-api.md', {
      type: 'text/markdown',
    })
    await user.upload(screen.getByLabelText('Document file'), file)
    expect(screen.getByLabelText('Document name *')).toHaveValue('video-api')
    await user.clear(screen.getByLabelText('Document name *'))
    await user.type(screen.getByLabelText('Document name *'), ' Video API ')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(post).toHaveBeenCalled())
    const [url, data] = post.mock.calls[0]
    expect(url).toBe('/api/model-document-templates')
    expect(data).toBeInstanceOf(FormData)
    expect((data as FormData).get('file')).toBe(file)
    expect((data as FormData).get('name')).toBe('Video API')
    await waitFor(() => expect(close).toHaveBeenCalledWith(false))
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: ['model-document-templates'],
    })
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: ['model-documentation'],
    })
  })

  it('rejects unsupported legacy Word and empty files before sending and permits a corrected upload', async () => {
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true, data: document } })
    renderWithClient(
      <DocumentTemplateDialog open mode='upload' onOpenChange={() => {}} />
    )
    const user = userEvent.setup({ applyAccept: false })
    await user.upload(
      screen.getByLabelText('Document file'),
      new File(['word'], 'legacy.doc')
    )
    expect(screen.getByRole('alert')).toHaveTextContent(
      'save legacy .doc files as .docx first'
    )
    expect(screen.getByLabelText('Document file')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    await user.upload(
      screen.getByLabelText('Document file'),
      new File([], 'empty.md')
    )
    expect(screen.getByRole('alert')).toHaveTextContent(
      'between 1 byte and 10 MB'
    )
    expect(post).not.toHaveBeenCalled()
    await user.upload(
      screen.getByLabelText('Document file'),
      new File(['# valid'], 'valid.md', { type: 'text/markdown' })
    )
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(post).toHaveBeenCalled())
  })

  it('keeps file selection and the dialog on server rejection so a user can retry', async () => {
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValueOnce({
        data: { success: false, message: 'Invalid Word document' },
      })
      .mockResolvedValueOnce({ data: { success: true, data: document } })
    const close = vi.fn()
    renderWithClient(
      <DocumentTemplateDialog open mode='upload' onOpenChange={close} />
    )
    const user = userEvent.setup()
    await user.upload(
      screen.getByLabelText('Document file'),
      new File(['word'], 'guide.docx', {
        type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      })
    )
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Invalid Word document')).toBeVisible()
    expect(close).not.toHaveBeenCalled()
    expect(screen.getByText('guide.docx')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(close).toHaveBeenCalledWith(false))
    expect(post).toHaveBeenCalledTimes(2)
  })

  it('renders Markdown tables/code in preview, sanitizes active content and saves edited source', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: document },
    })
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true, data: document } })
    const close = vi.fn()
    renderWithClient(
      <DocumentTemplateDialog
        open
        mode='edit'
        document={document}
        onOpenChange={close}
      />
    )
    const content = await screen.findByLabelText('Markdown content')
    const user = userEvent.setup()
    await user.clear(content)
    fireEvent.change(content, {
      target: {
        value:
          '# Rendered API\n\n| Field | Meaning |\n| --- | --- |\n| model | Name |\n\n```json\n{"model":"video"}\n```\n<script>alert(1)</script>',
      },
    })
    await user.click(screen.getByRole('tab', { name: 'Preview' }))
    expect(
      await screen.findByRole('heading', { name: 'Rendered API' })
    ).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Field' })).toBeVisible()
    expect(screen.getByText('{"model":"video"}')).toBeVisible()
    expect(screen.getByRole('dialog').querySelector('script')).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/model-document-templates/8', {
        name: 'Video API',
        content: expect.stringContaining('# Rendered API'),
      })
    )
    await waitFor(() => expect(close).toHaveBeenCalledWith(false))
  })

  it('protects unsaved renames and submits only the new name without overwriting the document', async () => {
    const get = vi.spyOn(api, 'get')
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true, data: document } })
    const close = vi.fn()
    renderWithClient(
      <DocumentTemplateDialog
        open
        mode='rename'
        document={document}
        onOpenChange={close}
      />
    )
    const user = userEvent.setup()
    await user.clear(screen.getByLabelText('Document name *'))
    await user.type(screen.getByLabelText('Document name *'), 'Renamed')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(await screen.findByText('Discard unsaved changes?')).toBeVisible()
    expect(close).not.toHaveBeenCalled()
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Cancel',
      })
    )
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/model-document-templates/8', {
        name: 'Renamed',
      })
    )
    expect(get).not.toHaveBeenCalled()
  })

  it('replaces a saved Word document by multipart upload without sending stale Markdown content', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: document },
    })
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true, data: document } })
    renderWithClient(
      <DocumentTemplateDialog
        open
        mode='edit'
        document={document}
        onOpenChange={() => {}}
      />
    )
    await screen.findByLabelText('Markdown content')
    const user = userEvent.setup()
    const file = new File(['word-xml'], 'new-guide.docx', {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })
    await user.upload(
      screen.getByLabelText('Replace document file (optional)'),
      file
    )
    expect(screen.queryByLabelText('Markdown content')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(put).toHaveBeenCalled())
    expect(put.mock.calls[0][0]).toBe('/api/model-document-templates/8')
    const data = put.mock.calls[0][1] as FormData
    expect(data.get('file')).toBe(file)
    expect(data.get('name')).toBe('Video API')
    expect(data.has('content')).toBe(false)
  })

  it('shows upload time/name/actions and requires confirmation before deletion, preserving the row on failure', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { items: [document], total: 1 } },
    })
    const remove = vi.spyOn(api, 'delete').mockResolvedValue({
      data: { success: false, message: 'Delete was not applied' },
    })
    renderWithClient(<DocumentTemplatesTable />)
    expect(await screen.findByText('Video API')).toBeVisible()
    expect(
      screen.getByRole('columnheader', { name: 'Uploaded At' })
    ).toBeVisible()
    expect(screen.getByRole('button', { name: 'Edit' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Rename' })).toBeVisible()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Delete' }))
    expect(remove).not.toHaveBeenCalled()
    expect(screen.getByText('Delete document template?')).toBeVisible()
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Delete',
      })
    )
    expect(await screen.findByText('Delete was not applied')).toBeVisible()
    expect(remove).toHaveBeenCalledWith('/api/model-document-templates/8')
    expect(screen.getAllByText('Video API')).toHaveLength(2)
  })

  it('shows a useful empty state and sends search/pagination to the server', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { items: [], total: 0 } },
    })
    renderWithClient(<DocumentTemplatesTable />)
    expect(await screen.findByText('No document templates')).toBeVisible()
    await userEvent
      .setup()
      .type(screen.getByPlaceholderText('Search document names'), 'video')
    await waitFor(() =>
      expect(get).toHaveBeenCalledWith('/api/model-document-templates', {
        params: { p: 1, page_size: 20, keyword: 'video' },
      })
    )
  })

  it('enforces upload byte limits including a zero byte file and uppercase supported suffixes', () => {
    const file = new File(['x'], 'guide.MD')
    expect(documentUploadError(file)).toBeUndefined()
    Object.defineProperty(file, 'size', {
      value: MAX_DOCUMENT_UPLOAD_BYTES + 1,
    })
    expect(documentUploadError(file)).toBe(
      'Document file must be between 1 byte and 10 MB'
    )
    expect(documentUploadError(new File([], 'empty.md'))).toBe(
      'Document file must be between 1 byte and 10 MB'
    )
  })
})
