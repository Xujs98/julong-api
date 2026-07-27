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
export const MAX_MODEL_TOKEN_ADJUSTMENT = 100

export type ModelTokenAdjustmentKey =
  | 'input'
  | 'output'
  | 'cache_read'
  | 'cache_creation'

export type ModelTokenAdjustment = Partial<
  Record<ModelTokenAdjustmentKey, number>
>

export type ModelTokenAdjustmentMap = Record<
  string,
  Record<string, Record<string, ModelTokenAdjustment>>
>

export type ModelTokenAdjustmentDraft = Record<
  ModelTokenAdjustmentKey,
  { enabled: boolean; value: string }
>

export type ModelTokenAdjustmentValidationError =
  | 'billing_group_required'
  | 'model_required'
  | 'adjustment_required'
  | 'adjustment_out_of_range'

export function parseModelTokenAdjustmentMap(
  value: string
): ModelTokenAdjustmentMap {
  if (!value.trim()) return {}
  try {
    const parsed: unknown = JSON.parse(value)
    if (
      typeof parsed !== 'object' ||
      parsed === null ||
      Array.isArray(parsed)
    ) {
      return {}
    }
    return parsed as ModelTokenAdjustmentMap
  } catch {
    return {}
  }
}

export function createModelTokenAdjustmentDraft(
  adjustment?: ModelTokenAdjustment
): ModelTokenAdjustmentDraft {
  return {
    input: toDraftValue(adjustment?.input),
    output: toDraftValue(adjustment?.output),
    cache_read: toDraftValue(adjustment?.cache_read),
    cache_creation: toDraftValue(adjustment?.cache_creation),
  }
}

function toDraftValue(value?: number) {
  return {
    enabled: value !== undefined,
    value: value === undefined ? '' : String(value),
  }
}

export function validateModelTokenAdjustmentDraft(
  billingGroup: string,
  modelName: string,
  draft: ModelTokenAdjustmentDraft
): ModelTokenAdjustmentValidationError | null {
  if (!billingGroup.trim()) return 'billing_group_required'
  if (!modelName.trim()) return 'model_required'

  const enabledValues = Object.values(draft).filter((item) => item.enabled)
  if (enabledValues.length === 0) return 'adjustment_required'

  const hasInvalidValue = enabledValues.some((item) => {
    const value = Number(item.value)
    return (
      item.value.trim() === '' ||
      !Number.isFinite(value) ||
      value < 0 ||
      value > MAX_MODEL_TOKEN_ADJUSTMENT
    )
  })
  return hasInvalidValue ? 'adjustment_out_of_range' : null
}

export function modelTokenAdjustmentFromDraft(
  draft: ModelTokenAdjustmentDraft
): ModelTokenAdjustment {
  const adjustment: ModelTokenAdjustment = {}
  const keys: ModelTokenAdjustmentKey[] = [
    'input',
    'output',
    'cache_read',
    'cache_creation',
  ]
  for (const key of keys) {
    if (draft[key].enabled) adjustment[key] = Number(draft[key].value)
  }
  return adjustment
}
