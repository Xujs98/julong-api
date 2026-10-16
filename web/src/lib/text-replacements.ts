import * as z from 'zod'

const textReplacementRuleSchema = z.object({
  from: z.string().trim().min(1),
  to: z.string(),
})

export const textReplacementConfigSchema = z
  .object({
    request: z.array(textReplacementRuleSchema).max(100).optional(),
    response: z.array(textReplacementRuleSchema).max(100).optional(),
  })
  .strict()

export function isOptionalTextReplacementConfig(
  value: string | undefined
): boolean {
  if (!value?.trim()) return true
  try {
    return textReplacementConfigSchema.safeParse(JSON.parse(value)).success
  } catch {
    return false
  }
}

export function normalizeTextReplacementConfig(
  value: string | undefined
): string {
  if (!value?.trim()) return ''
  try {
    const parsed = textReplacementConfigSchema.parse(JSON.parse(value))
    if (!parsed.request?.length && !parsed.response?.length) return ''
    return JSON.stringify(parsed)
  } catch {
    return value
  }
}
