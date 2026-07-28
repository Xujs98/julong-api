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
import type { GetLogsParams } from '../types'

export type UsageLogExportSelection =
  | { scope: 'page' }
  | { scope: 'today'; now?: Date }
  | { scope: 'custom'; start: Date; end: Date }
  | { scope: 'all' }

export function buildUsageLogExportParams(
  params: GetLogsParams,
  selection: UsageLogExportSelection
): GetLogsParams {
  const exportParams = { ...params }

  if (selection.scope === 'page') return exportParams

  delete exportParams.p
  delete exportParams.page_size

  if (selection.scope === 'all') {
    delete exportParams.start_timestamp
    delete exportParams.end_timestamp
    return exportParams
  }

  if (selection.scope === 'today') {
    const start = new Date(selection.now ?? new Date())
    start.setHours(0, 0, 0, 0)
    const end = new Date(start)
    end.setHours(23, 59, 59, 999)
    exportParams.start_timestamp = Math.floor(start.getTime() / 1000)
    exportParams.end_timestamp = Math.floor(end.getTime() / 1000)
    return exportParams
  }

  const startTimestamp = selection.start.getTime()
  const endTimestamp = selection.end.getTime()
  if (
    !Number.isFinite(startTimestamp) ||
    !Number.isFinite(endTimestamp) ||
    startTimestamp > endTimestamp
  ) {
    throw new RangeError('Invalid usage log export time range')
  }
  exportParams.start_timestamp = Math.floor(startTimestamp / 1000)
  exportParams.end_timestamp = Math.floor(endTimestamp / 1000)
  return exportParams
}
