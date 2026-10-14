import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CodeBlock } from '@/components/ai-elements/code-block'
import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { requireServerSuccess } from '@/lib/server-error-message'

import { getTaskRequest } from '../../api'
import type { TaskLog } from '../../types'

export function TaskRequestCell(props: { log: TaskLog }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const request = useQuery({
    queryKey: ['usage-logs', 'task-request', props.log.task_id],
    queryFn: async () =>
      requireServerSuccess(await getTaskRequest(props.log.task_id)),
    enabled: open,
    retry: false,
    staleTime: 60_000,
  })

  if (!props.log.request_available) {
    return <span className='text-muted-foreground/60 text-xs'>-</span>
  }

  let content = <LoadingState />
  if (request.isError) {
    content = (
      <ErrorState
        title={t('Failed to load request body')}
        onRetry={() => void request.refetch()}
      />
    )
  } else if (request.isSuccess) {
    const body = request.data.data.request_body
    content = <EmptyState title={t('Request body not recorded')} />
    if (body != null) {
      const code = JSON.stringify(body, null, 2)
      content = (
        <div className='min-w-0 space-y-3'>
          {request.data.data.truncated ? (
            <p className='text-muted-foreground text-xs'>
              {t('The request body was truncated to the capture limit')}
            </p>
          ) : null}
          {typeof body === 'object' &&
          !Array.isArray(body) &&
          'prompt' in body ? (
            <section className='space-y-2'>
              <h3 className='text-sm font-semibold'>{t('Prompt')}</h3>
              <div className='bg-muted rounded-xl p-4 text-sm break-words whitespace-pre-wrap'>
                {String(body.prompt ?? '')}
              </div>
            </section>
          ) : null}
          <section className='space-y-2'>
            <h3 className='text-sm font-semibold'>{t('Other parameters')}</h3>
            <CodeBlock
              code={
                typeof body === 'object' && !Array.isArray(body)
                  ? JSON.stringify(
                      Object.fromEntries(
                        Object.entries(body).filter(([key]) => key !== 'prompt')
                      ),
                      null,
                      2
                    )
                  : code
              }
              language='json'
              wrapLines
              maxExpandedLines={24}
              className='my-0'
            />
          </section>
        </div>
      )
    }
  }

  return (
    <>
      <Button
        type='button'
        variant='outline'
        size='xs'
        onClick={() => setOpen(true)}
      >
        {t('View request')}
      </Button>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title={t('Model request body')}
        description={
          <>
            <span className='block break-all'>
              {t('Task ID')}: {props.log.task_id}
            </span>
            <span className='block'>
              {t(
                'Saved submission request. Credentials are hidden and media links use placeholders.'
              )}
            </span>
          </>
        }
        footer={
          <>
            {request.data?.data.request_body != null ? (
              <CopyButton
                value={JSON.stringify(request.data.data.request_body, null, 2)}
                size='sm'
                variant='secondary'
              >
                {t('Copy')}
              </CopyButton>
            ) : null}
            <Button variant='secondary' onClick={() => setOpen(false)}>
              {t('Close')}
            </Button>
          </>
        }
        contentClassName='min-w-0 overflow-hidden sm:max-w-3xl'
        contentHeight='min(72dvh, 720px)'
      >
        {content}
      </Dialog>
    </>
  )
}
