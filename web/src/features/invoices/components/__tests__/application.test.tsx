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
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import type { AxiosAdapter } from 'axios'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { Invoices } from '../../index'
import {
  invoiceApplicationSchema,
  parseInvoiceInformation,
  validatePaymentProofs,
} from '../../lib/invoice-application'
import type { InvoiceApplication, InvoiceInfo } from '../../types'
import { ManageInvoiceDialog } from '../manage-invoice-dialog'

const originalAdapter = api.defaults.adapter
const originalAuth = useAuthStore.getState().auth
let client: QueryClient
let adapter: ReturnType<typeof vi.fn<AxiosAdapter>>
let info: InvoiceInfo
let submitSuccess: boolean

beforeEach(() => {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: { id: 2, username: 'invoice-user', role: ROLE.USER },
      accessToken: 'test-invoice-access-token',
    },
  })
  info = {
    enabled: true,
    email: 'bound@example.com',
    allowed_sources: ['online_recharge', 'redemption', 'admin_adjustment'],
    minimum_amount: 100,
    unit: 'USD',
    processing_days: '3',
    rejection_freeze_hours: 24,
    credits: [
      {
        request_id: 'credit-100',
        created_at: 1800000000,
        quota: 100,
        amount: 100,
        source: 'redemption',
        content: 'credit',
      },
    ],
    credits_total: 1,
    applications: [],
    applications_total: 0,
    page_size: 20,
  }
  submitSuccess = true
  adapter = vi.fn(async (config) => ({
    data:
      config.url === '/api/user/invoice/apply'
        ? {
            success: submitSuccess,
            message: submitSuccess
              ? ''
              : 'Payment source is no longer eligible',
            data: { id: 1 },
          }
        : { success: true, data: info },
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  }))
  api.defaults.adapter = adapter
  vi.stubGlobal(
    'URL',
    Object.assign(class extends URL {}, {
      createObjectURL: vi.fn(() => 'blob:invoice-proof'),
      revokeObjectURL: vi.fn(),
    })
  )
})

afterEach(() => {
  cleanup()
  client.clear()
  api.defaults.adapter = originalAdapter
  useAuthStore.setState({ auth: originalAuth })
  vi.unstubAllGlobals()
})

function InvoicePage() {
  return (
    <QueryClientProvider client={client}>
      <Invoices />
    </QueryClientProvider>
  )
}

async function openApplication() {
  await screen.findByRole('checkbox', { name: /Select credit from/ })
  fireEvent.click(screen.getByRole('checkbox', { name: /Select credit from/ }))
  fireEvent.click(
    screen.getByRole('button', { name: 'Invoice selected credits' })
  )
  return screen.findByRole('dialog', { name: 'Apply for invoice' })
}

describe('invoice application form', () => {
  test('opens the required invoicing fields before submitting selected credits', async () => {
    render(<InvoicePage />)
    await openApplication()
    expect(
      screen.getByRole('radio', { name: 'VAT ordinary invoice' })
    ).toHaveAttribute('aria-checked', 'true')
    expect(
      screen.getByRole('radio', { name: 'VAT special invoice' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('textbox', { name: 'Company name' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('textbox', { name: 'Tax number' })
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Payment screenshots')).toHaveAttribute(
      'accept',
      'image/png,image/jpeg,image/webp'
    )
    expect(
      adapter.mock.calls.filter(
        ([config]) => config.url === '/api/user/invoice/apply'
      )
    ).toHaveLength(0)
  })
})

function fillApplication() {
  fireEvent.change(screen.getByRole('textbox', { name: 'Company name' }), {
    target: { value: 'Julong Test Company' },
  })
  fireEvent.change(screen.getByRole('textbox', { name: 'Tax number' }), {
    target: { value: '913100001234567890' },
  })
  fireEvent.change(screen.getByLabelText('Payment screenshots'), {
    target: {
      files: [new File(['payment'], 'payment.png', { type: 'image/png' })],
    },
  })
}

describe('invoice application details and submission', () => {
  test('defaults receiving email to the bound address and preserves a user edit during data refresh', async () => {
    render(<InvoicePage />)
    await openApplication()
    const email = screen.getByRole('textbox', { name: 'Receiving email' })
    expect(email).toHaveValue('bound@example.com')
    fireEvent.change(email, { target: { value: 'finance@example.com' } })
    info = { ...info, email: 'new-bound@example.com' }
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['invoice'] })
    })
    expect(email).toHaveValue('finance@example.com')
  })

  test('recognizes invoicing text locally and preserves fields that are not identified', async () => {
    render(<InvoicePage />)
    await openApplication()
    const company = screen.getByRole('textbox', { name: 'Company name' })
    fireEvent.change(company, { target: { value: 'Existing company' } })
    const pasted = screen.getByRole('textbox', {
      name: 'Paste invoice information',
    })
    fireEvent.change(pasted, {
      target: {
        value:
          '名称：\n税号：913100001234567890\n地址、电话：上海 021-1234\n开户行及账号：测试银行 123456',
      },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Recognize and fill' }))
    expect(company).toHaveValue('Existing company')
    expect(screen.getByRole('textbox', { name: 'Tax number' })).toHaveValue(
      '913100001234567890'
    )
    expect(screen.getByRole('textbox', { name: 'Notes' })).toHaveValue(
      '地址、电话：上海 021-1234\n开户行及账号：测试银行 123456'
    )
    expect(
      adapter.mock.calls.filter(([config]) => config.method !== 'get')
    ).toHaveLength(0)
  })

  test('keeps invalid required fields visible and makes no application request', async () => {
    render(<InvoicePage />)
    await openApplication()
    fireEvent.change(screen.getByRole('textbox', { name: 'Receiving email' }), {
      target: { value: 'invalid-email' },
    })
    fireEvent.click(
      screen.getByRole('button', { name: 'Submit invoice application' })
    )
    expect(
      await screen.findByText('Company name is required')
    ).toBeInTheDocument()
    expect(screen.getByText('Tax number is required')).toBeInTheDocument()
    expect(
      screen.getByText('Enter a valid receiving email')
    ).toBeInTheDocument()
    expect(
      screen.getByText('Upload 1 to 3 payment screenshots')
    ).toBeInTheDocument()
    expect(
      screen.getByRole('textbox', { name: 'Company name' })
    ).toHaveAttribute('aria-invalid', 'true')
    expect(
      adapter.mock.calls.filter(
        ([config]) => config.url === '/api/user/invoice/apply'
      )
    ).toHaveLength(0)
  })

  test('adds three payment screenshots, rejects a fourth, and allows removal and preview', async () => {
    render(<InvoicePage />)
    await openApplication()
    const upload = screen.getByLabelText('Payment screenshots')
    const files = ['first.png', 'second.jpg', 'third.webp'].map(
      (name, index) =>
        new File(['payment'], name, {
          type: ['image/png', 'image/jpeg', 'image/webp'][index],
        })
    )
    fireEvent.change(upload, { target: { files } })
    expect(
      screen.getByRole('button', { name: 'Add screenshot' })
    ).toBeDisabled()
    fireEvent.change(upload, {
      target: {
        files: [new File(['extra'], 'fourth.png', { type: 'image/png' })],
      },
    })
    expect(
      screen.getByText('Upload 1 to 3 payment screenshots')
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', {
        name: 'Preview payment screenshot fourth.png',
      })
    ).not.toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', {
        name: 'Preview payment screenshot first.png',
      })
    )
    const preview = await screen.findByRole('dialog', {
      name: 'Payment screenshot',
    })
    expect(preview.querySelector('img')).toHaveAttribute('alt', 'first.png')
    fireEvent.click(within(preview).getByRole('button', { name: 'Close' }))
    await waitFor(() =>
      expect(
        screen.queryByRole('dialog', { name: 'Payment screenshot' })
      ).not.toBeInTheDocument()
    )
    fireEvent.click(
      screen.getByRole('button', {
        name: 'Remove payment screenshot first.png',
      })
    )
    expect(
      screen.queryByRole('button', {
        name: 'Preview payment screenshot first.png',
      })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add screenshot' })).toBeEnabled()
  })

  test('rejects oversized screenshots without removing existing valid uploads', async () => {
    render(<InvoicePage />)
    await openApplication()
    fillApplication()
    const large = new File(
      [new Uint8Array(5 * 1024 * 1024 + 1)],
      'oversized.png',
      { type: 'image/png' }
    )
    fireEvent.change(screen.getByLabelText('Payment screenshots'), {
      target: { files: [large] },
    })
    expect(
      screen.getByText(
        'Each payment screenshot must be nonempty and no larger than 5 MiB'
      )
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', {
        name: 'Preview payment screenshot payment.png',
      })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', {
        name: 'Preview payment screenshot oversized.png',
      })
    ).not.toBeInTheDocument()
  })

  test('submits the chosen invoice type, recipient override, details, and repeated images as multipart', async () => {
    render(<InvoicePage />)
    await openApplication()
    fillApplication()
    fireEvent.click(screen.getByRole('radio', { name: 'VAT special invoice' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Receiving email' }), {
      target: { value: 'finance@example.com' },
    })
    fireEvent.change(screen.getByRole('textbox', { name: 'Notes' }), {
      target: { value: 'Payment reference 123' },
    })
    fireEvent.change(screen.getByLabelText('Payment screenshots'), {
      target: {
        files: [new File(['payment2'], 'payment2.jpg', { type: 'image/jpeg' })],
      },
    })
    fireEvent.click(
      screen.getByRole('button', { name: 'Submit invoice application' })
    )
    await waitFor(() =>
      expect(
        adapter.mock.calls.some(
          ([config]) => config.url === '/api/user/invoice/apply'
        )
      ).toBe(true)
    )
    const submitCall = adapter.mock.calls.find(
      ([request]) => request.url === '/api/user/invoice/apply'
    )
    expect(submitCall).toBeDefined()
    if (!submitCall) throw new Error('Invoice request was not sent')
    const config = submitCall[0]
    const submitted = config.data as FormData
    expect(submitted).toBeInstanceOf(FormData)
    expect(submitted.get('request_ids')).toBe('["credit-100"]')
    expect(submitted.get('invoice_type')).toBe('special')
    expect(submitted.get('company_name')).toBe('Julong Test Company')
    expect(submitted.get('tax_number')).toBe('913100001234567890')
    expect(submitted.get('email')).toBe('finance@example.com')
    expect(submitted.get('notes')).toBe('Payment reference 123')
    expect(
      submitted.getAll('payment_proofs').map((file) => (file as File).name)
    ).toEqual(['payment.png', 'payment2.jpg'])
    await waitFor(() =>
      expect(
        screen.queryByRole('dialog', { name: 'Apply for invoice' })
      ).not.toBeInTheDocument()
    )
    await openApplication()
    expect(screen.getByRole('textbox', { name: 'Company name' })).toHaveValue(
      ''
    )
    expect(
      screen.queryByRole('button', {
        name: 'Preview payment screenshot payment.png',
      })
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('textbox', { name: 'Receiving email' })
    ).toHaveValue('bound@example.com')
  })

  test('preserves the complete application after a failed submission', async () => {
    submitSuccess = false
    render(<InvoicePage />)
    await openApplication()
    fillApplication()
    fireEvent.click(
      screen.getByRole('button', { name: 'Submit invoice application' })
    )
    await waitFor(() =>
      expect(
        adapter.mock.calls.some(
          ([config]) => config.url === '/api/user/invoice/apply'
        )
      ).toBe(true)
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Submit invoice application' })
      ).toBeEnabled()
    )
    expect(
      screen.getByRole('dialog', { name: 'Apply for invoice' })
    ).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Company name' })).toHaveValue(
      'Julong Test Company'
    )
    expect(
      screen.getByRole('button', {
        name: 'Preview payment screenshot payment.png',
      })
    ).toBeInTheDocument()
  })

  test('removes disabled sources from prior-page selections without restoring them when reenabled', async () => {
    info = { ...info, credits_total: 21 }
    render(<InvoicePage />)
    await screen.findByRole('checkbox', { name: /Select credit from/ })
    fireEvent.click(
      screen.getByRole('checkbox', { name: /Select credit from/ })
    )
    expect(
      screen.getByRole('button', { name: 'Invoice selected credits' })
    ).toBeEnabled()
    info = {
      ...info,
      credits: [
        {
          ...info.credits[0],
          request_id: 'online-credit-200',
          source: 'online_recharge',
        },
      ],
    }
    fireEvent.click(screen.getAllByRole('button', { name: 'Next page' })[0])
    await screen.findByText('Online top-up', { exact: true })
    expect(
      screen.getByRole('button', { name: 'Invoice selected credits' })
    ).toBeEnabled()
    info = { ...info, allowed_sources: ['online_recharge'] }
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['invoice'] })
    })
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: 'Invoice selected credits' })
      ).not.toBeInTheDocument()
    )
    expect(
      screen.getByRole('checkbox', { name: /Select credit from/ })
    ).toHaveAttribute('aria-checked', 'false')
    expect(
      screen.getByText('Eligible payment methods: Online top-up')
    ).toBeInTheDocument()
    info = {
      ...info,
      allowed_sources: ['online_recharge', 'redemption', 'admin_adjustment'],
    }
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['invoice'] })
    })
    expect(
      screen.queryByRole('button', { name: 'Invoice selected credits' })
    ).not.toBeInTheDocument()
  })

  test('prevents choosing a payment source disabled in the current settings', async () => {
    info = { ...info, allowed_sources: ['online_recharge'] }
    render(<InvoicePage />)
    const checkbox = await screen.findByRole('checkbox', {
      name: /Select credit from/,
    })
    expect(checkbox).toHaveAttribute('aria-disabled', 'true')
    fireEvent.click(checkbox)
    expect(checkbox).toHaveAttribute('aria-checked', 'false')
    expect(
      screen.queryByRole('button', { name: 'Invoice selected credits' })
    ).not.toBeInTheDocument()
  })

  test('shows administrator invoicing details and loads payment screenshots through the authenticated API', async () => {
    const application: InvoiceApplication = {
      id: 7,
      user_id: 2,
      email: 'finance@example.com',
      amount_quota: 100,
      amount: 100,
      unit: 'USD',
      status: 'pending',
      admin_note: '',
      created_at: 1800000000,
      updated_at: 1800000000,
      completed_at: 0,
      has_attachment: false,
      invoice_type: 'special',
      company_name: 'Julong Test Company',
      tax_number: '913100001234567890',
      notes: 'Bank details',
      payment_proofs: [
        { id: 5, filename: 'private.png', content_type: 'image/png', size: 7 },
      ],
    }
    adapter.mockImplementation(async (config) => ({
      data: new Blob(['payment'], { type: 'image/png' }),
      status: 200,
      statusText: 'OK',
      headers: { 'content-type': 'image/png' },
      config,
    }))
    render(
      <QueryClientProvider client={client}>
        <ManageInvoiceDialog
          application={application}
          saving={false}
          onOpenChange={() => {}}
          onSave={() => {}}
          onComplete={() => {}}
        />
      </QueryClientProvider>
    )
    expect(screen.getByText('VAT special invoice')).toBeInTheDocument()
    expect(screen.getByText('Julong Test Company')).toBeInTheDocument()
    expect(screen.getByText('913100001234567890')).toBeInTheDocument()
    expect(screen.getByText('Bank details')).toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', {
        name: 'View payment screenshot private.png',
      })
    )
    expect(
      await screen.findByRole('img', { name: 'private.png' })
    ).toHaveAttribute('src', 'blob:invoice-proof')
    expect(adapter.mock.calls[0][0].url).toBe(
      '/api/user/invoice/applications/7/payment-proofs/5'
    )
    expect(adapter.mock.calls[0][0].responseType).toBe('blob')
  })

  test('continues to display historical applications with no added invoicing fields', async () => {
    const application: InvoiceApplication = {
      id: 1,
      user_id: 2,
      email: 'old@example.com',
      amount_quota: 100,
      amount: 100,
      unit: 'USD',
      status: 'completed',
      admin_note: '',
      created_at: 1800000000,
      updated_at: 1800000000,
      completed_at: 1800000001,
      has_attachment: false,
    }
    render(
      <ManageInvoiceDialog
        application={application}
        saving={false}
        onOpenChange={() => {}}
        onSave={() => {}}
        onComplete={() => {}}
      />
    )
    expect(screen.getByText('old@example.com')).toBeInTheDocument()
    expect(screen.getByText('No payment screenshots')).toBeInTheDocument()
    expect(screen.getByLabelText('Status')).toBeDisabled()
  })
})

describe('invoice application validation', () => {
  test('parses name and taxpayer aliases and returns no values for unrecognized text', () => {
    expect(
      parseInvoiceInformation(
        '公司全称：矩龙公司\n统一社会信用代码：91310000 1234567890'
      )
    ).toEqual({ company_name: '矩龙公司', tax_number: '913100001234567890' })
    expect(parseInvoiceInformation('unlabelled invoice information')).toEqual(
      {}
    )
    expect(parseInvoiceInformation('名称：\n税号：')).toEqual({})
  })

  test('accepts nonempty supported screenshots at the five MiB limit and rejects invalid boundaries', () => {
    const boundary = new File([new Uint8Array(5 * 1024 * 1024)], 'limit.png', {
      type: 'image/png',
    })
    expect(validatePaymentProofs([boundary])).toBeNull()
    expect(validatePaymentProofs([])).toBe('Upload 1 to 3 payment screenshots')
    expect(
      validatePaymentProofs([new File([], 'empty.png', { type: 'image/png' })])
    ).toBe('Each payment screenshot must be nonempty and no larger than 5 MiB')
    expect(
      validatePaymentProofs([
        new File(['pdf'], 'proof.pdf', { type: 'application/pdf' }),
      ])
    ).toBe('Payment screenshots must be PNG, JPG, or WebP')
  })

  test('rejects non-alphanumeric tax identifiers and overly long applicant details', () => {
    const valid = {
      invoice_type: 'ordinary',
      company_name: 'Company',
      tax_number: '913100001234567890',
      email: 'bound@example.com',
      notes: '',
      payment_proofs: [
        new File(['payment'], 'proof.png', { type: 'image/png' }),
      ],
    }
    expect(invoiceApplicationSchema.safeParse(valid).success).toBe(true)
    for (const patch of [
      { tax_number: '123-456' },
      { company_name: 'c'.repeat(201) },
      { tax_number: '1'.repeat(51) },
      { notes: 'n'.repeat(1001) },
    ]) {
      expect(
        invoiceApplicationSchema.safeParse({ ...valid, ...patch }).success
      ).toBe(false)
    }
  })
})
