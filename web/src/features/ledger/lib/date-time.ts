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
function padDateTimePart(value: number): string {
  return String(value).padStart(2, '0')
}

export function formatLedgerDateTimeInput(timestampSeconds?: number): string {
  const date = new Date(
    timestampSeconds == null ? Date.now() : timestampSeconds * 1000
  )
  return [
    date.getFullYear(),
    '-',
    padDateTimePart(date.getMonth() + 1),
    '-',
    padDateTimePart(date.getDate()),
    'T',
    padDateTimePart(date.getHours()),
    ':',
    padDateTimePart(date.getMinutes()),
    ':',
    padDateTimePart(date.getSeconds()),
  ].join('')
}

export function parseLedgerDateTimeInput(value: string): number | null {
  if (value.trim() === '') return null
  const timestamp = new Date(value).getTime()
  if (!Number.isFinite(timestamp)) return null
  return Math.floor(timestamp / 1000)
}
