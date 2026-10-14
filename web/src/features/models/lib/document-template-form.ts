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
import * as z from 'zod'

export const documentTemplateFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Document template name is required')
    .max(128, 'Document template name must contain 1 to 128 characters'),
  content: z.string(),
})

export const MAX_DOCUMENT_UPLOAD_BYTES = 10 * 1024 * 1024
export const MAX_DOCUMENT_CONTENT_BYTES = 2 * 1024 * 1024

export function documentUploadError(file: File): string | undefined {
  if (!/\.(md|markdown|docx)$/i.test(file.name)) {
    return 'Upload a Markdown (.md) or Word (.docx) document; save legacy .doc files as .docx first'
  }
  if (file.size === 0 || file.size > MAX_DOCUMENT_UPLOAD_BYTES) {
    return 'Document file must be between 1 byte and 10 MB'
  }
  return undefined
}
