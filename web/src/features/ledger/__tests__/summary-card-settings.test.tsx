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
import { after, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
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

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { LedgerSummaryCards } =
  await import('../components/ledger-summary-cards')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

after(() => {
  domWindow.close()
})

test('current user quota title opens estimate-ratio settings for editors', async () => {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  let opened = 0

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <LedgerSummaryCards
          loading={false}
          canEditEstimateRatio
          onEditEstimateRatio={() => {
            opened += 1
          }}
          summary={{
            user_quota: { real: 500_000, estimated: 250_000 },
            usage_quota: { real: 100_000, estimated: 50_000 },
            daily_operating_cost: '1',
            total_operating_cost: '1',
            operational_quota: { real: 500_000, cost_ratio: '2' },
            cost_ratios: { plus: '2', pro: null, k12: null },
            estimate_ratio: '2',
            days: 1,
            ledger_entry_count: 1,
            included_user_count: 1,
          }}
        />
      </I18nextProvider>
    )
  })

  const settingsButton = container.querySelector<HTMLButtonElement>(
    'button[aria-label="Configure estimate ratio"]'
  )
  assert.ok(settingsButton)
  await act(async () => settingsButton.click())
  assert.equal(opened, 1)

  await act(async () => root.unmount())
  container.remove()
})
