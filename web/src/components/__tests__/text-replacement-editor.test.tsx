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

For commercial licensing, please contact support@quantumnous.com
*/
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import { TextReplacementEditor } from '../text-replacement-editor'

const initialValue = JSON.stringify(
  {
    request: [{ from: '矩龙API启动', to: 'By倾城' }],
    response: [{ from: 'upstream', to: 'client' }],
  },
  null,
  2
)

function renderEditor(
  value: string,
  onChange: (nextValue: string) => void = vi.fn()
) {
  return render(<TextReplacementEditor value={value} onChange={onChange} />)
}

describe('TextReplacementEditor', () => {
  test('renders request and response rules in visual mode', () => {
    renderEditor(initialValue)

    expect(screen.getByRole('tab', { name: 'Visual' })).toHaveAttribute(
      'aria-selected',
      'true'
    )
    expect(
      screen.getByRole('textbox', { name: 'Request Replace text' })
    ).toHaveValue('矩龙API启动')
    expect(
      screen.getByRole('textbox', { name: 'Request Replace with' })
    ).toHaveValue('By倾城')
    expect(
      screen.getByRole('textbox', { name: 'Response Replace text' })
    ).toHaveValue('upstream')
    expect(
      screen.getByRole('textbox', { name: 'Response Replace with' })
    ).toHaveValue('client')
  })

  test('adds, edits, and deletes a visual rule', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderEditor('', onChange)

    const addButtons = screen.getAllByRole('button', { name: 'Add Row' })
    await user.click(addButtons[0])

    const requestFrom = screen.getByRole('textbox', {
      name: 'Request Replace text',
    })
    const requestTo = screen.getByRole('textbox', {
      name: 'Request Replace with',
    })
    await user.type(requestFrom, '矩龙API启动')
    await user.type(requestTo, 'By倾城')

    expect(onChange).toHaveBeenLastCalledWith(
      JSON.stringify(
        { request: [{ from: '矩龙API启动', to: 'By倾城' }] },
        null,
        2
      )
    )

    await user.click(screen.getByRole('button', { name: 'Request Delete' }))

    expect(
      screen.getAllByText(
        'No mappings configured. Click "Add Row" to get started.'
      )
    ).toHaveLength(2)
    expect(onChange).toHaveBeenLastCalledWith('')
  })

  test('synchronizes visual rules with JSON mode', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderEditor(initialValue, onChange)

    await user.click(screen.getByRole('tab', { name: 'JSON' }))
    const textarea = await screen.findByRole('textbox', {
      name: 'Text Replacement Rules',
    })
    expect(textarea).toHaveValue(initialValue)
    expect(onChange).toHaveBeenLastCalledWith(initialValue)

    const updatedValue = JSON.stringify(
      { request: [{ from: 'source', to: 'upstream' }] },
      null,
      2
    )
    fireEvent.input(textarea, { target: { value: updatedValue } })
    expect(onChange).toHaveBeenLastCalledWith(updatedValue)

    await user.click(screen.getByRole('tab', { name: 'Visual' }))
    expect(
      screen.getByRole('textbox', { name: 'Request Replace text' })
    ).toHaveValue('source')
    expect(
      screen.getByRole('textbox', { name: 'Request Replace with' })
    ).toHaveValue('upstream')
    expect(
      screen.queryByRole('textbox', { name: 'Response Replace text' })
    ).not.toBeInTheDocument()
  })

  test('shows an error and stays in JSON mode for invalid JSON', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    renderEditor('', onChange)

    await user.click(screen.getByRole('tab', { name: 'JSON' }))
    const textarea = await screen.findByRole('textbox', {
      name: 'Text Replacement Rules',
    })
    fireEvent.input(textarea, { target: { value: '{invalid' } })

    expect(screen.getByText('JSON format error')).toBeInTheDocument()
    expect(onChange).toHaveBeenLastCalledWith('{invalid')

    await user.click(screen.getByRole('tab', { name: 'Visual' }))
    expect(screen.getByRole('tab', { name: 'JSON' })).toHaveAttribute(
      'aria-selected',
      'true'
    )
    expect(screen.getByText('JSON format error')).toBeInTheDocument()
  })
})
