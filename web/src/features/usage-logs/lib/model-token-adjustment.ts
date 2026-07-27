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
import type { UsageLog } from '../data/schema'
import type { LogOtherData, ModelTokenUsage } from '../types'

export type TokenUsageDisplay = {
  displayed: ModelTokenUsage
  actual: ModelTokenUsage
  hasAdjustment: boolean
}

function cacheCreationTotal(usage: ModelTokenUsage) {
  const splitTotal =
    (usage.cache_creation_5m || 0) + (usage.cache_creation_1h || 0)
  return splitTotal > 0 ? splitTotal : usage.cache_creation || 0
}

function normalizeTokenUsage(usage: ModelTokenUsage): ModelTokenUsage {
  return {
    input: usage.input || 0,
    output: usage.output || 0,
    cache_read: usage.cache_read || 0,
    cache_creation: cacheCreationTotal(usage),
    cache_creation_5m: usage.cache_creation_5m || 0,
    cache_creation_1h: usage.cache_creation_1h || 0,
  }
}

export function getTokenUsageDisplay(
  log: UsageLog,
  other: LogOtherData | null,
  isAdmin: boolean
): TokenUsageDisplay {
  const fallback = normalizeTokenUsage({
    input: log.prompt_tokens || 0,
    output: log.completion_tokens || 0,
    cache_read: other?.cache_tokens || 0,
    cache_creation: other?.cache_creation_tokens || 0,
    cache_creation_5m: other?.cache_creation_tokens_5m || 0,
    cache_creation_1h: other?.cache_creation_tokens_1h || 0,
  })
  const audit = isAdmin ? other?.admin_info?.model_token_adjustment : undefined
  if (!audit) {
    return { displayed: fallback, actual: fallback, hasAdjustment: false }
  }

  return {
    displayed: normalizeTokenUsage(audit.billed),
    actual: normalizeTokenUsage(audit.actual),
    hasAdjustment: true,
  }
}
