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
export type RequestConversationItem = {
  id: string
  role: string
  text: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function contentToText(value: unknown): string {
  if (typeof value === 'string') return value
  if (Array.isArray(value)) {
    return value.map(contentToText).filter(Boolean).join('\n')
  }
  if (!isRecord(value)) return value == null ? '' : String(value)

  for (const key of ['text', 'input_text', 'output_text', 'content']) {
    if (value[key] !== undefined) {
      const text = contentToText(value[key])
      if (text) return text
    }
  }
  return JSON.stringify(value, null, 2)
}

export function extractRequestConversation(
  content: unknown
): RequestConversationItem[] {
  if (!isRecord(content)) return []
  const items: RequestConversationItem[] = []
  const instructions = content.instructions ?? content.instruction
  if (instructions !== undefined) {
    const text = contentToText(instructions)
    if (text) items.push({ id: `item-${items.length}`, role: 'system', text })
  }

  const appendItems = (value: unknown, fallbackRole: string) => {
    if (typeof value === 'string') {
      items.push({
        id: `item-${items.length}`,
        role: fallbackRole,
        text: value,
      })
      return
    }
    if (!Array.isArray(value)) return
    for (const entry of value) {
      if (!isRecord(entry)) {
        const text = contentToText(entry)
        if (text) {
          items.push({ id: `item-${items.length}`, role: fallbackRole, text })
        }
        continue
      }
      let role = fallbackRole
      if (typeof entry.role === 'string') {
        role = entry.role
      } else if (typeof entry.type === 'string') {
        role = entry.type
      }
      const text = contentToText(
        entry.content ?? entry.input ?? entry.output ?? entry
      )
      if (text) items.push({ id: `item-${items.length}`, role, text })
    }
  }

  appendItems(content.messages, 'message')
  appendItems(content.input, 'user')
  if (content.prompt !== undefined) {
    const text = contentToText(content.prompt)
    if (text) items.push({ id: `item-${items.length}`, role: 'user', text })
  }
  return items
}
