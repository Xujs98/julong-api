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
import { lazy, Suspense, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { getServerErrorMessage } from '@/lib/server-error-message'

import { getModelDocumentation } from '../api'

const Markdown = lazy(() =>
  import('@/components/ui/markdown').then((module) => ({
    default: module.Markdown,
  }))
)

/** Render the selected document in place of generated examples, never as raw HTML. */
export function ModelDocumentTemplateSection(props: {
  modelName: string
  fallback: ReactNode
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['model-documentation', props.modelName],
    queryFn: ({ signal }) => getModelDocumentation(props.modelName, signal),
    staleTime: 60 * 1000,
    retry: false,
    meta: { errorToast: false },
  })

  if (query.isPending) {
    return (
      <div role='status'>
        <LoadingState message={t('Loading...')} />
      </div>
    )
  }
  if (query.isError) {
    return (
      <ErrorState
        className='min-h-48'
        title={t('Could not load the document template')}
        description={getServerErrorMessage(query.error)}
        onRetry={() => void query.refetch()}
      />
    )
  }
  if (!query.data) return props.fallback

  const document = query.data
  return (
    <section className='min-w-0 space-y-3' aria-label={t('Code samples')}>
      <div className='flex min-w-0 items-center justify-between gap-3'>
        <div className='min-w-0'>
          <h3 className='text-sm font-semibold'>{t('Code samples')}</h3>
          <p
            className='text-muted-foreground mt-1 truncate text-xs'
            title={document.name}
          >
            {document.name}
          </p>
        </div>
        <CopyButton
          value={document.content}
          variant='outline'
          size='sm'
          aria-label={t('Copy document')}
        >
          {t('Copy document')}
        </CopyButton>
      </div>
      <div className='max-h-[70vh] min-w-0 overflow-y-auto rounded-lg border p-4'>
        <Suspense fallback={<LoadingState message={t('Loading...')} />}>
          <Markdown className='min-w-0'>{document.content}</Markdown>
        </Suspense>
      </div>
    </section>
  )
}
