/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

import type { InvoiceApplication } from '../../types'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'HTMLInputElement',
  'HTMLSelectElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MouseEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act, useState } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { SectionPageLayout } = await import('@/components/layout')
const { ApplicationsTable } = await import('../applications-table')
const { ManageInvoiceDialog } = await import('../manage-invoice-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const application: InvoiceApplication = {
  id: 1,
  user_id: 2,
  username: 'invoice-user',
  email: 'invoice@example.com',
  amount_quota: 310,
  amount: 310,
  unit: 'USD',
  status: 'pending',
  admin_note: '',
  created_at: 1_800_000_000,
  updated_at: 1_800_000_000,
  completed_at: 0,
  has_attachment: false,
  items: [],
}

function ManagementHarness() {
  const [managed, setManaged] = useState<InvoiceApplication | null>(null)
  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>Self-service Invoices</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <ApplicationsTable
            title='User invoice applications'
            description='Review applications'
            applications={[application]}
            total={1}
            page={1}
            pageSize={20}
            admin
            onPageChange={() => {}}
            onManage={setManaged}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <ManageInvoiceDialog
        application={managed}
        saving={false}
        onOpenChange={(open) => {
          if (!open) setManaged(null)
        }}
        onSave={() => {}}
        onComplete={() => {}}
      />
    </>
  )
}

describe('invoice application management', () => {
  after(() => domWindow.close())

  test('opens the management dialog when the row action is clicked', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ManagementHarness />
        </I18nextProvider>
      )
    })

    const button = container.querySelector<HTMLButtonElement>(
      '[aria-label="View invoice application 1"]'
    )
    assert.ok(button)
    await act(async () => button.click())

    const dialog = document.body.querySelector('[data-slot="dialog-content"]')
    assert.ok(dialog)
    assert.match(dialog.textContent ?? '', /Invoice details/)

    await act(async () => root.unmount())
    container.remove()
  })

  test('keeps detail actions enabled for every terminal status', async () => {
    const terminalApplications: InvoiceApplication[] = [
      { ...application, id: 2, status: 'rejected' },
      { ...application, id: 3, status: 'non_reusable' },
      { ...application, id: 4, status: 'completed' },
      { ...application, id: 5, status: 'cancelled' },
    ]
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ApplicationsTable
            title='User invoice applications'
            description='Review applications'
            applications={terminalApplications}
            total={terminalApplications.length}
            page={1}
            pageSize={20}
            admin
            onPageChange={() => {}}
            onManage={() => {}}
          />
        </I18nextProvider>
      )
    })

    for (const applicationId of [2, 3, 4, 5]) {
      const button = container.querySelector<HTMLButtonElement>(
        `[aria-label="View invoice application ${applicationId}"]`
      )
      assert.ok(button)
      assert.equal(button.disabled, false)
    }

    await act(async () => root.unmount())
    container.remove()
  })

  test('shows source billing details and locks terminal status editing', async () => {
    const completedApplication: InvoiceApplication = {
      ...application,
      status: 'completed',
      completed_at: 1_800_000_200,
      has_attachment: true,
      items: [
        {
          id: 7,
          application_id: application.id,
          user_id: application.user_id,
          source_request_id: 'billing-request-7',
          source: 'online_recharge',
          quota: 120,
          source_created_at: 1_800_000_100,
          released_at: 0,
          frozen_until: 0,
          amount: 120,
          content: 'payment reference 7',
        },
      ],
    }
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ManageInvoiceDialog
            application={completedApplication}
            saving={false}
            onOpenChange={() => {}}
            onSave={() => {}}
            onComplete={() => {}}
          />
        </I18nextProvider>
      )
    })

    const dialog = document.body.querySelector('[data-slot="dialog-content"]')
    assert.ok(dialog)
    assert.ok(dialog.querySelector('[aria-label="Copy source request ID"]'))
    assert.match(dialog.textContent ?? '', /payment reference 7/)
    const status = dialog.querySelector<HTMLSelectElement>('#invoice-status')
    assert.ok(status)
    assert.equal(status.disabled, true)

    await act(async () => root.unmount())
    container.remove()
  })

  test('reveals invoice delivery controls before completing an application', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ManageInvoiceDialog
            application={application}
            saving={false}
            onOpenChange={() => {}}
            onSave={() => {}}
            onComplete={() => {}}
          />
        </I18nextProvider>
      )
    })

    const dialog = document.body.querySelector('[data-slot="dialog-content"]')
    assert.ok(dialog)
    const status = dialog.querySelector<HTMLSelectElement>('#invoice-status')
    assert.ok(status)
    await act(async () => {
      status.value = 'completed'
      status.dispatchEvent(new Event('change', { bubbles: true }))
    })

    assert.match(dialog.textContent ?? '', /Upload invoice attachment/)
    assert.match(dialog.textContent ?? '', /Send invoice by email/)
    assert.match(dialog.textContent ?? '', /Email settings/)
    const save = [...dialog.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.includes('Save Changes')
    )
    assert.ok(save)
    assert.equal(save.disabled, true)

    await act(async () => root.unmount())
    container.remove()
  })

  test('confirms withdrawal for a completed application', async () => {
    const completedApplication: InvoiceApplication = {
      ...application,
      status: 'completed',
      completed_at: 1_800_000_200,
      has_attachment: true,
    }
    let withdrawnId = 0
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ManageInvoiceDialog
            application={completedApplication}
            saving={false}
            onOpenChange={() => {}}
            onSave={() => {}}
            onComplete={() => {}}
            onWithdraw={(target) => {
              withdrawnId = target.id
            }}
          />
        </I18nextProvider>
      )
    })

    const dialog = document.body.querySelector('[data-slot="dialog-content"]')
    assert.ok(dialog)
    const withdraw = [
      ...dialog.querySelectorAll<HTMLButtonElement>('button'),
    ].find((button) => button.textContent?.includes('Withdraw'))
    assert.ok(withdraw)
    await act(async () => withdraw.click())

    const confirmation = document.body.querySelector(
      '[data-slot="alert-dialog-content"]'
    )
    assert.ok(confirmation)
    assert.match(confirmation.textContent ?? '', /uploaded invoice attachment/)
    const confirm = [
      ...confirmation.querySelectorAll<HTMLButtonElement>('button'),
    ].find((button) => button.textContent?.includes('Confirm withdrawal'))
    assert.ok(confirm)
    await act(async () => confirm.click())
    assert.equal(withdrawnId, completedApplication.id)

    await act(async () => root.unmount())
    container.remove()
  })
})
