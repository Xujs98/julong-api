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
import { useQuery } from '@tanstack/react-query'
import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { LoadingState } from '@/components/loading-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { getServerErrorMessage } from '@/lib/server-error-message'

import {
  documentTemplatesQueryKeys,
  listDocumentTemplates,
  type DocumentTemplate,
} from '../document-templates-api'
import type { ModelFormInput } from '../lib/model-form'

export function ModelDocumentTemplateField(props: {
  control: Control<ModelFormInput>
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: [...documentTemplatesQueryKeys.all, 'selector'],
    queryFn: async () => {
      const documents = new Map<number, DocumentTemplate>()
      // The server clamps page_size to 100; keep loading until the catalog ends.
      for (let page = 1; ; page++) {
        const result = await listDocumentTemplates({ p: page, page_size: 100 })
        for (const item of result.items) documents.set(item.id, item)
        if (!result.items.length || page * result.page_size >= result.total) {
          break
        }
      }
      return [...documents.values()]
    },
    retry: false,
    meta: { errorToast: false },
  })
  const documents = query.data ?? []

  return (
    <FormField
      control={props.control}
      name='document_template_id'
      render={({ field }) => {
        const selectedId = Number(field.value ?? 0)
        const missing =
          selectedId > 0 &&
          query.isSuccess &&
          !documents.some((item) => item.id === selectedId)
        const options = [
          { value: '0', label: t('Use generated code samples') },
          ...documents.map((item) => ({
            value: String(item.id),
            label: item.name,
          })),
        ]
        if (missing) {
          options.push({
            value: String(selectedId),
            label: t('Missing document template'),
          })
        }
        return (
          <FormItem>
            <FormLabel>{t('Document template')}</FormLabel>
            <FormControl>
              <Combobox
                value={String(selectedId)}
                options={options}
                onValueChange={(value) => {
                  if (value != null) field.onChange(Number(value))
                }}
                onBlur={field.onBlur}
                ref={field.ref}
                disabled={props.disabled || query.isPending || query.isError}
                placeholder={t('Document template')}
                aria-label={t('Document template')}
              />
            </FormControl>
            <FormDescription>
              {t(
                'Choose a reusable document for API code samples in the model square. No template keeps generated examples.'
              )}
            </FormDescription>
            {query.isPending && (
              <div role='status'>
                <LoadingState
                  inline
                  message={t('Loading document templates...')}
                />
              </div>
            )}
            {query.isError && (
              <Alert variant='destructive'>
                <AlertDescription>
                  {getServerErrorMessage(
                    query.error,
                    t('Failed to load document templates')
                  )}
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    className='mt-2'
                    onClick={() => void query.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </AlertDescription>
              </Alert>
            )}
            {query.isSuccess && documents.length === 0 && !missing && (
              <p className='text-muted-foreground text-xs'>
                {t(
                  'No document templates uploaded. Upload one from Document templates first.'
                )}
              </p>
            )}
            {missing && (
              <Alert>
                <AlertDescription>
                  {t(
                    'Selected document template no longer exists. Choose another template or clear the selection.'
                  )}
                </AlertDescription>
              </Alert>
            )}
            <FormMessage />
          </FormItem>
        )
      }}
    />
  )
}
