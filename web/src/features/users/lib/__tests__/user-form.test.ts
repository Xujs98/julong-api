/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  transformFormDataToPayload,
  USER_FORM_DEFAULT_VALUES,
} from '../user-form.ts'

const bindingUpdates = {
  email: 'billing@example.com',
  github_id: 'github-user',
  discord_id: 'discord-user',
  oidc_id: 'oidc-user',
  wechat_id: 'wechat-user',
  telegram_id: 'telegram-user',
  linux_do_id: 'linuxdo-user',
}

describe('transformFormDataToPayload binding updates', () => {
  test('omits binding updates when an existing user keeps the switch off', () => {
    const payload = transformFormDataToPayload(
      {
        ...USER_FORM_DEFAULT_VALUES,
        username: 'alice',
        binding_updates: bindingUpdates,
      },
      42,
      undefined,
      false
    )

    assert.equal('binding_updates' in payload, false)
    assert.equal(payload.id, 42)
  })

  test('includes binding updates when an existing user enables the switch', () => {
    const payload = transformFormDataToPayload(
      {
        ...USER_FORM_DEFAULT_VALUES,
        username: 'alice',
        binding_updates: bindingUpdates,
      },
      42,
      undefined,
      true
    )

    assert.deepEqual(payload.binding_updates, bindingUpdates)
  })

  test('keeps binding updates in new-user payloads', () => {
    const payload = transformFormDataToPayload({
      ...USER_FORM_DEFAULT_VALUES,
      username: 'new-user',
      binding_updates: bindingUpdates,
    })

    assert.deepEqual(payload.binding_updates, bindingUpdates)
  })
})
