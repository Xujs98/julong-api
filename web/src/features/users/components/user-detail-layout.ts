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
export const userDetailDialogLayoutClasses = {
  content:
    'flex h-[calc(100dvh-2rem)] max-h-[calc(100dvh-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[900px]',
  header:
    'shrink-0 gap-0 border-b px-4 pt-4 pr-12 pb-0 sm:px-6 sm:pt-5 sm:pr-14',
  tabs: 'min-h-0 flex-1 gap-0 overflow-hidden',
  tabsList:
    'mx-4 mt-2 max-w-[calc(100%-2rem)] shrink-0 overflow-x-auto sm:mx-6 sm:max-w-[calc(100%-3rem)]',
  scrollArea: 'min-h-0 flex-1 overflow-hidden',
} as const
