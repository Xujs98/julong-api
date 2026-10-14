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
import { z } from 'zod'

export const MAX_PAYMENT_PROOFS = 3
export const MAX_PAYMENT_PROOF_BYTES = 5 * 1024 * 1024
export const PAYMENT_PROOF_TYPES = ['image/png', 'image/jpeg', 'image/webp']

export function validatePaymentProofs(files: File[]): string | null {
  if (files.length === 0 || files.length > MAX_PAYMENT_PROOFS) {
    return 'Upload 1 to 3 payment screenshots'
  }
  if (files.some((file) => !PAYMENT_PROOF_TYPES.includes(file.type))) {
    return 'Payment screenshots must be PNG, JPG, or WebP'
  }
  if (
    files.some((file) => file.size === 0 || file.size > MAX_PAYMENT_PROOF_BYTES)
  ) {
    return 'Each payment screenshot must be nonempty and no larger than 5 MiB'
  }
  return null
}

export const invoiceApplicationSchema = z.object({
  invoice_type: z.enum(['ordinary', 'special']),
  company_name: z
    .string()
    .trim()
    .min(1, 'Company name is required')
    .max(200, 'Company name must not exceed 200 characters'),
  tax_number: z
    .string()
    .trim()
    .min(1, 'Tax number is required')
    .max(50, 'Tax number must not exceed 50 characters')
    .regex(
      /^[A-Za-z0-9]+$/,
      'Tax number must contain only letters and numbers'
    ),
  email: z
    .string()
    .trim()
    .min(1, 'Receiving email is required')
    .max(254, 'Receiving email must not exceed 254 characters')
    .email('Enter a valid receiving email'),
  notes: z.string().trim().max(1000, 'Notes must not exceed 1000 characters'),
  payment_proofs: z
    .array(z.custom<File>((value) => value instanceof File))
    .superRefine((files, context) => {
      const message = validatePaymentProofs(files)
      if (message) context.addIssue({ code: 'custom', message })
    }),
})

export type InvoiceApplicationForm = z.infer<typeof invoiceApplicationSchema>

/** Parse labelled text locally; unidentified fields never replace user input. */
export function parseInvoiceInformation(
  text: string
): Partial<
  Pick<InvoiceApplicationForm, 'company_name' | 'tax_number' | 'notes'>
> {
  const result: Partial<
    Pick<InvoiceApplicationForm, 'company_name' | 'tax_number' | 'notes'>
  > = {}
  const company = text
    .match(
      /(?:^|[\n;；])\s*(?:公司名称|公司全称|单位名称|纳税人名称|发票抬头|名称|抬头|company(?: name)?|name)\s*[:：][ \t]*([^\n;；]+)/i
    )?.[1]
    ?.trim()
  const tax = text
    .match(
      /(?:^|[\n;；])\s*(?:纳税人识别号|纳税识别号|统一社会信用代码|税务登记号|税号|tax(?: number| id)?|vat id)\s*[:：][ \t]*([^\n;；]+)/i
    )?.[1]
    ?.trim()
  if (company) result.company_name = company
  if (tax) result.tax_number = tax.replaceAll(/\s/g, '')
  const additional = text
    .split(/[\n;；]/)
    .map((line) => line.trim())
    .filter((line) =>
      /^(?:地址(?:[、，,及和 ]*电话)?|电话|开户行(?:及账号)?|银行账号|账号|开户银行|address|phone|bank(?: account)?|account)\s*[:：]\s*\S/i.test(
        line
      )
    )
  if (additional.length) result.notes = additional.join('\n')
  return result
}
