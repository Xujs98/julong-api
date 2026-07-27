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

export const USER_TAG_COLOR_PRESETS = [
  '#EF4444',
  '#F97316',
  '#F59E0B',
  '#22C55E',
  '#3B82F6',
] as const

export const USER_TAG_ROW_MARKER_CLASS_NAME =
  'absolute -left-2 top-1/2 z-10 h-15 w-1 -translate-y-1/2'

export function getUserTagFilterValue(tagId?: number): string {
  return String(tagId ?? 0)
}
