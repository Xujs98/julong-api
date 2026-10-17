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
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { usePageSizeOptions } from '@/hooks/use-page-size-options'
import { resolvePageSize } from '@/lib/page-size-options'
import { cn } from '@/lib/utils'

type PageSizeSelectProps = {
  value: number
  onValueChange: (value: number) => void
  className?: string
  disabled?: boolean
}

export function PageSizeSelect({
  value,
  onValueChange,
  className,
  disabled = false,
}: PageSizeSelectProps) {
  const { t } = useTranslation()
  const options = usePageSizeOptions()
  const effectiveValue = resolvePageSize(value, options)

  useEffect(() => {
    if (effectiveValue !== value) onValueChange(effectiveValue)
  }, [effectiveValue, onValueChange, value])

  return (
    <label className={cn('flex items-center gap-2 text-sm', className)}>
      <span className='text-muted-foreground whitespace-nowrap'>
        {t('Rows per page')}
      </span>
      <NativeSelect
        aria-label={t('Rows per page')}
        disabled={disabled}
        size='sm'
        value={String(effectiveValue)}
        onChange={(event) => onValueChange(Number(event.target.value))}
      >
        {options.map((option) => (
          <NativeSelectOption key={option} value={String(option)}>
            {option}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </label>
  )
}
