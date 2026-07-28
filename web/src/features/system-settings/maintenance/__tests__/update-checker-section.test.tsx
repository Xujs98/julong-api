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
const { UpdateCheckerSection } = await import('../update-checker-section')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Check for updates': 'Check for updates',
        'Julong version': 'Julong version',
        'Merged New API version': 'Merged New API version',
        'System maintenance': 'System maintenance',
        Unknown: 'Unknown',
        'Upstream commit': 'Upstream commit',
        'Uptime since': 'Uptime since',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('system maintenance version display', () => {
  after(() => {
    domWindow.close()
  })

  test('shows Julong and merged upstream versions separately', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <UpdateCheckerSection
            julongVersion='julong-20260728-a1b2c3d'
            upstreamVersion='v1.0.0-rc.22'
            upstreamCommit='afe16c64c'
            startTime={1_700_000_000}
          />
        </I18nextProvider>
      )
    })

    assert.match(container.textContent ?? '', /Julong version/)
    assert.match(container.textContent ?? '', /julong-20260728-a1b2c3d/)
    assert.match(container.textContent ?? '', /Merged New API version/)
    assert.match(container.textContent ?? '', /v1\.0\.0-rc\.22/)
    assert.match(container.textContent ?? '', /Upstream commit: afe16c64c/)

    await act(async () => root.unmount())
    container.remove()
  })
})
