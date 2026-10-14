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
import { useEffect, useRef } from 'react'

/** Opt-in wheel navigation for a column, without trapping normal page scrolling. */
export function useHorizontalWheelScroll(columnId: string | undefined) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = ref.current
    if (!container || !columnId) return

    const handleWheel = (event: WheelEvent) => {
      if (event.ctrlKey || event.defaultPrevented || !event.cancelable) return
      if (!(event.target instanceof Element)) return
      const cell = event.target.closest(
        'th[data-column-id], td[data-column-id]'
      )
      if (cell?.getAttribute('data-column-id') !== columnId) return
      const viewport = cell.closest<HTMLElement>(
        '[data-table-scroll-viewport], [data-slot="table-container"]'
      )
      if (!viewport || !container.contains(viewport)) return

      const maxScroll = viewport.scrollWidth - viewport.clientWidth
      if (maxScroll <= 0) return
      const delta =
        Math.abs(event.deltaX) > Math.abs(event.deltaY)
          ? event.deltaX
          : event.deltaY
      let scale = 1
      if (event.deltaMode === WheelEvent.DOM_DELTA_LINE) {
        scale = 16
      } else if (event.deltaMode === WheelEvent.DOM_DELTA_PAGE) {
        scale = viewport.clientWidth
      }
      const nextScroll = Math.max(
        0,
        Math.min(maxScroll, viewport.scrollLeft + delta * scale)
      )
      if (nextScroll === viewport.scrollLeft) return
      event.preventDefault()
      viewport.scrollLeft = nextScroll
    }

    // React's delegated wheel listener is passive in Chromium. A native,
    // explicitly non-passive listener is required to cancel vertical scrolling.
    container.addEventListener('wheel', handleWheel, { passive: false })
    return () => container.removeEventListener('wheel', handleWheel)
  }, [columnId])

  return ref
}
