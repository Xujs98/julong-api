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
import { createEvent, fireEvent, render, screen } from '@testing-library/react'
import { expect, test } from 'vitest'

import { DataTableView } from '../../core/data-table-view'
import { useDataTable } from '../use-data-table'

const rows = [{ name: 'A long primitive output that remains truncated' }]
const columns = [
  { accessorKey: 'name', header: 'Output', size: 240, cell: rows[0].name },
]

function Fixture(props: { splitHeader: boolean; enabled?: boolean }) {
  const { table } = useDataTable({
    data: rows,
    columns,
  })
  return (
    <DataTableView
      table={table}
      splitHeader={props.splitHeader}
      horizontalWheelColumnId={props.enabled === false ? undefined : 'name'}
      pinnedColumns={[{ columnId: 'name', side: 'right' }]}
    />
  )
}

test.each([false, true])(
  'the %s split-header table converts line/page wheel units on its own viewport',
  (splitHeader) => {
    render(<Fixture splitHeader={splitHeader} />)
    const cell = screen.getByRole('cell')
    const viewport = cell.closest('table')?.parentElement
    if (!viewport) throw new Error('Expected table scroll viewport')
    Object.defineProperties(viewport, {
      scrollWidth: { configurable: true, value: 2000 },
      clientWidth: { configurable: true, value: 500 },
      scrollLeft: { configurable: true, writable: true, value: 100 },
    })
    fireEvent.wheel(cell, {
      deltaY: 2,
      deltaMode: WheelEvent.DOM_DELTA_LINE,
      bubbles: true,
      cancelable: true,
    })
    expect(viewport.scrollLeft).toBe(132)
    fireEvent.wheel(cell, {
      deltaY: 1,
      deltaMode: WheelEvent.DOM_DELTA_PAGE,
      bubbles: true,
      cancelable: true,
    })
    expect(viewport.scrollLeft).toBe(632)
    fireEvent.wheel(cell, {
      deltaX: -40,
      deltaY: 10,
      bubbles: true,
      cancelable: true,
    })
    expect(viewport.scrollLeft).toBe(592)
    // Edge gradients may paint outside, while the text wrapper still truncates.
    expect(cell).toHaveClass('overflow-visible')
    expect(cell).not.toHaveClass('overflow-hidden')
    expect(screen.getByText(rows[0].name)).toHaveClass('truncate')
  }
)

test('removing the opt-in restores native wheel behavior on existing tables', () => {
  const view = render(<Fixture splitHeader={false} />)
  const cell = screen.getByRole('cell')
  const viewport = cell.closest('table')?.parentElement
  if (!viewport) throw new Error('Expected table scroll viewport')
  Object.defineProperties(viewport, {
    scrollWidth: { configurable: true, value: 2000 },
    clientWidth: { configurable: true, value: 500 },
    scrollLeft: { configurable: true, writable: true, value: 100 },
  })
  view.rerender(<Fixture splitHeader={false} enabled={false} />)
  const wheel = createEvent.wheel(cell, {
    deltaY: 80,
    bubbles: true,
    cancelable: true,
  })
  fireEvent(cell, wheel)
  expect(viewport.scrollLeft).toBe(100)
  expect(wheel.defaultPrevented).toBe(false)
})
