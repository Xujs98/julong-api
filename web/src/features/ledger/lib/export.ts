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
import type { LedgerListParams } from '../types'

export type LedgerExportSelection =
  | { scope: 'page' }
  | { scope: 'today'; now?: Date }
  | { scope: 'custom'; start: Date; end: Date }
  | { scope: 'all' }

export function buildLedgerExportParams(
  params: LedgerListParams,
  selection: LedgerExportSelection
): LedgerListParams {
  const result = { ...params }
  if (selection.scope === 'page') return result

  delete result.p
  delete result.page_size
  if (selection.scope === 'all') {
    delete result.start_timestamp
    delete result.end_timestamp
    return result
  }

  if (selection.scope === 'today') {
    const start = new Date(selection.now ?? new Date())
    start.setHours(0, 0, 0, 0)
    const end = new Date(start)
    end.setHours(23, 59, 59, 999)
    result.start_timestamp = Math.floor(start.getTime() / 1000)
    result.end_timestamp = Math.floor(end.getTime() / 1000)
    return result
  }

  const startTimestamp = selection.start.getTime()
  const endTimestamp = selection.end.getTime()
  if (
    !Number.isFinite(startTimestamp) ||
    !Number.isFinite(endTimestamp) ||
    startTimestamp > endTimestamp
  ) {
    throw new RangeError('Invalid ledger export time range')
  }
  result.start_timestamp = Math.floor(startTimestamp / 1000)
  result.end_timestamp = Math.floor(endTimestamp / 1000)
  return result
}
