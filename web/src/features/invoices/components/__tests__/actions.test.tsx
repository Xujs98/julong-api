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

import type { InvoiceApplication, InvoiceStatus } from '../../types'

const domWindow = new Window()
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
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
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { ApplicationsTable } = await import('../applications-table')

const i18n = createInstance()
await i18n
  .use(initReactI18next)
  .init({ lng: 'en', resources: { en: { translation: {} } } })
;(
  globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true

function application(
  id: number,
  status: InvoiceStatus,
  hasAttachment = false
): InvoiceApplication {
  return {
    id,
    user_id: 1,
    email: 'user@example.com',
    amount_quota: 100,
    amount: 1,
    unit: 'USD',
    status,
    admin_note: '',
    created_at: 1_800_000_000,
    updated_at: 1_800_000_000,
    completed_at: status === 'completed' ? 1_800_000_100 : 0,
    has_attachment: hasAttachment,
    items: [],
  }
}

describe('invoice application actions', () => {
  after(() => domWindow.close())

  test('shows cancel only for pending and download only for completed attachment', async () => {
    const applications = [
      application(1, 'pending'),
      application(2, 'completed', true),
      application(3, 'rejected'),
    ]
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ApplicationsTable
            title='History'
            description='History'
            applications={applications}
            total={3}
            page={1}
            pageSize={20}
            onPageChange={() => {}}
            onCancel={() => {}}
            onDownload={() => {}}
          />
        </I18nextProvider>
      )
    })

    assert.ok(
      container.querySelector('[aria-label="Cancel invoice application 1"]')
    )
    assert.equal(
      container.querySelector('[aria-label="Cancel invoice application 2"]'),
      null
    )
    assert.ok(
      container.querySelector('[aria-label="Download invoice application 2"]')
    )
    assert.equal(
      container.querySelector('[aria-label="Download invoice application 3"]'),
      null
    )

    await act(async () => root.unmount())
    container.remove()
  })
})
