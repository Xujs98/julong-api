/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useMemo } from 'react'

import { parsePageSizeOptions } from '@/lib/page-size-options'

import { useStatus } from './use-status'

export function usePageSizeOptions(): number[] {
  const { status } = useStatus()
  return useMemo(() => parsePageSizeOptions(status), [status])
}
