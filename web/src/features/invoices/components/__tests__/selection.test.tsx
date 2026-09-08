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
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

import type { InvoiceCredit } from '../../types'

const domWindow = new Window()
const domGlobals = [
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
const { EligibleCreditsTable } = await import('../eligible-credits-table')
const { InvoiceAvailabilityNotice } =
  await import('../invoice-availability-notice')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const credits: InvoiceCredit[] = [
  {
    request_id: 'credit-40',
    created_at: 1_800_000_000,
    quota: 40,
    amount: 40,
    source: 'online_recharge',
    content: 'first credit',
  },
  {
    request_id: 'credit-60',
    created_at: 1_800_000_100,
    quota: 60,
    amount: 60,
    source: 'redemption',
    content: 'second credit',
  },
]

const frozenCredit: InvoiceCredit = {
  ...credits[0],
  request_id: 'frozen-credit',
  frozen_until: Math.floor(Date.now() / 1000) + 3600,
}

function SelectionHarness() {
  const [selected, setSelected] = useState(new Map<string, InvoiceCredit>())

  return (
    <EligibleCreditsTable
      credits={credits}
      total={credits.length}
      page={1}
      pageSize={20}
      unit='USD'
      minimumAmount={100}
      enabled
      selected={selected}
      applying={false}
      onToggle={(credit, checked) => {
        setSelected((current) => {
          const next = new Map(current)
          if (checked) next.set(credit.request_id, credit)
          else next.delete(credit.request_id)
          return next
        })
      }}
      onTogglePage={() => {}}
      onClear={() => setSelected(new Map())}
      onApply={() => {}}
      onRefresh={() => {}}
      onPageChange={() => {}}
    />
  )
}

function getApplyButton(container: HTMLElement) {
  const button = [...container.querySelectorAll('button')].find((candidate) =>
    candidate.textContent?.includes('Invoice selected credits')
  )
  assert.ok(button)
  assert.equal(button.tagName, 'BUTTON')
  return button as HTMLButtonElement
}

describe('invoice credit selection', () => {
  after(() => domWindow.close())

  test('keeps the apply action disabled while selected credits are below the threshold', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <SelectionHarness />
        </I18nextProvider>
      )
    })

    const checkboxes = container.querySelectorAll<HTMLElement>(
      '[aria-label^="Select credit from"]'
    )
    assert.equal(checkboxes.length, 2)
    await act(async () => checkboxes[0].click())

    assert.equal(getApplyButton(container).disabled, true)
    await act(async () => root.unmount())
    container.remove()
  })

  test('enables the apply action when multiple selected credits reach the threshold', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <SelectionHarness />
        </I18nextProvider>
      )
    })

    const checkboxes = container.querySelectorAll<HTMLElement>(
      '[aria-label^="Select credit from"]'
    )
    await act(async () => {
      checkboxes[0].click()
      checkboxes[1].click()
    })

    assert.equal(getApplyButton(container).disabled, false)
    await act(async () => root.unmount())
    container.remove()
  })

  test('shows the private testing notice while self-service invoices are closed', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <InvoiceAvailabilityNotice enabled={false} />
        </I18nextProvider>
      )
    })

    assert.match(
      container.textContent ?? '',
      /This feature is currently in private testing/
    )
    await act(async () => root.unmount())
    container.remove()
  })

  test('shows a released frozen credit but keeps its selector disabled', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <EligibleCreditsTable
            credits={[frozenCredit]}
            total={1}
            page={1}
            pageSize={20}
            unit='USD'
            minimumAmount={100}
            enabled
            selected={new Map()}
            applying={false}
            onToggle={() => {}}
            onTogglePage={() => {}}
            onClear={() => {}}
            onApply={() => {}}
            onRefresh={() => {}}
            onPageChange={() => {}}
          />
        </I18nextProvider>
      )
    })

    const checkbox = container.querySelector<HTMLElement>(
      '[aria-label^="Select credit from"]'
    )
    assert.ok(checkbox)
    assert.equal(checkbox.getAttribute('data-disabled') !== null, true)
    assert.match(container.textContent ?? '', /Frozen until/)

    await act(async () => root.unmount())
    container.remove()
  })
})
