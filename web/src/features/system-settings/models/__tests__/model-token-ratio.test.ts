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
import { describe, test } from 'node:test'

import {
  createModelTokenAdjustmentDraft,
  modelTokenAdjustmentFromDraft,
  validateModelTokenAdjustmentDraft,
} from '../model-token-ratio.ts'

describe('model token ratio editor state', () => {
  test('omits disabled token categories from saved JSON', () => {
    const draft = createModelTokenAdjustmentDraft({
      input: 0.1,
      cache_read: 0.2,
    })
    draft.cache_read.enabled = false

    assert.deepEqual(modelTokenAdjustmentFromDraft(draft), { input: 0.1 })
  })

  test('requires one enabled finite adjustment in the supported range', () => {
    const emptyDraft = createModelTokenAdjustmentDraft()
    assert.equal(
      validateModelTokenAdjustmentDraft('codex-v1', 'gpt-test', emptyDraft),
      'adjustment_required'
    )

    const invalidDraft = createModelTokenAdjustmentDraft({ input: 0.1 })
    invalidDraft.input.value = '101'
    assert.equal(
      validateModelTokenAdjustmentDraft('codex-v1', 'gpt-test', invalidDraft),
      'adjustment_out_of_range'
    )
  })

  test('requires a billing group before saving a model adjustment', () => {
    const draft = createModelTokenAdjustmentDraft({ input: 0.1 })

    assert.equal(
      validateModelTokenAdjustmentDraft('', 'gpt-test', draft),
      'billing_group_required'
    )
  })
})
