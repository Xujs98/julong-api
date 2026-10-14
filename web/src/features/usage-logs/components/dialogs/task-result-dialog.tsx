import { useQuery } from '@tanstack/react-query'
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
import { ExternalLink } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CodeBlock } from '@/components/ai-elements/code-block'
import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { requireServerSuccess } from '@/lib/server-error-message'

import { getTaskDetail } from '../../api'

export function TaskResultCell(props: { taskId: string }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const result = useQuery({
    queryKey: ['usage-logs', 'task-detail', props.taskId],
    queryFn: async () =>
      requireServerSuccess(await getTaskDetail(props.taskId)),
    enabled: open,
    retry: false,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      if (!open || query.state.error) return false
      const status = query.state.data?.data.status
      return status === 'SUCCESS' || status === 'FAILURE' ? false : 3000
    },
  })
  const code = result.data ? JSON.stringify(result.data.data, null, 2) : ''
  const resultUrl =
    result.data?.data && typeof result.data.data.result_url === 'string'
      ? result.data.data.result_url
      : ''
  return (
    <>
      <div className='flex w-40 max-w-full min-w-0 items-center gap-1 overflow-hidden'>
        <Button
          variant='link'
          size='xs'
          className='min-w-0 flex-1 overflow-hidden px-0 font-mono'
          onClick={() => setOpen(true)}
          title={props.taskId}
        >
          <span className='block min-w-0 truncate'>{props.taskId}</span>
        </Button>
        <CopyButton value={props.taskId} className='size-6 shrink-0' />
      </div>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title={t('Task result')}
        description={t(
          'Latest saved polling result. Pending tasks refresh automatically.'
        )}
        contentClassName='sm:max-w-3xl'
        contentHeight='min(65dvh, 650px)'
        footer={
          <>
            <Button
              variant='outline'
              disabled={result.isFetching}
              onClick={() => void result.refetch()}
            >
              {t('Refresh')}
            </Button>
            {code ? (
              <CopyButton value={code} size='sm' variant='secondary'>
                {t('Copy')}
              </CopyButton>
            ) : null}
            {resultUrl ? (
              <Button
                render={<a href={resultUrl} target='_blank' rel='noreferrer' />}
                variant='outline'
                size='sm'
                title={resultUrl}
              >
                <ExternalLink className='mr-1 h-3.5 w-3.5' aria-hidden='true' />
                {t('Open result URL')}
              </Button>
            ) : null}
            <Button variant='secondary' onClick={() => setOpen(false)}>
              {t('Close')}
            </Button>
          </>
        }
      >
        {result.isPending ? <LoadingState /> : null}
        {result.isError ? (
          <ErrorState
            title={t('Failed to query task')}
            onRetry={() => void result.refetch()}
          />
        ) : null}
        {code ? (
          <CodeBlock
            code={code}
            language='json'
            wrapLines
            maxExpandedLines={24}
            className='my-0'
          />
        ) : null}
      </Dialog>
    </>
  )
}
