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

import { extractRequestConversation } from '../request-content'

describe('extractRequestConversation', () => {
  test('extracts instructions and Responses API message content in order', () => {
    const result = extractRequestConversation({
      instructions: 'Act as a coding assistant.',
      input: [
        {
          type: 'message',
          role: 'user',
          content: [
            { type: 'input_text', text: 'Inspect the failing build.' },
            { type: 'input_text', text: 'Then propose a fix.' },
          ],
        },
        {
          type: 'message',
          role: 'assistant',
          content: [{ type: 'output_text', text: 'I found the cause.' }],
        },
      ],
    })

    assert.deepEqual(
      result.map(({ role, text }) => ({ role, text })),
      [
        { role: 'system', text: 'Act as a coding assistant.' },
        {
          role: 'user',
          text: 'Inspect the failing build.\nThen propose a fix.',
        },
        { role: 'assistant', text: 'I found the cause.' },
      ]
    )
  })

  test('extracts Chat Completions messages and legacy prompts', () => {
    const result = extractRequestConversation({
      messages: [
        { role: 'system', content: 'Be concise.' },
        { role: 'user', content: 'Explain this function.' },
      ],
      prompt: 'Legacy completion prompt',
    })

    assert.deepEqual(
      result.map(({ role, text }) => ({ role, text })),
      [
        { role: 'system', text: 'Be concise.' },
        { role: 'user', text: 'Explain this function.' },
        { role: 'user', text: 'Legacy completion prompt' },
      ]
    )
  })

  test('returns no conversation for non-object content', () => {
    assert.deepEqual(extractRequestConversation(null), [])
    assert.deepEqual(extractRequestConversation('plain JSON preview'), [])
  })
})
