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
import { ImageIcon, Trash2 } from 'lucide-react'
import { useMemo, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'

import { getInvoicePaymentProof } from '../api'
import type { InvoicePaymentProof } from '../types'

function createPaymentProofResource(file: Blob) {
  let url: string | null = null
  return {
    getSnapshot: () => url,
    subscribe: (onChange: () => void) => {
      url = URL.createObjectURL(file)
      onChange()
      return () => {
        if (url) URL.revokeObjectURL(url)
        url = null
      }
    },
  }
}

function PaymentProofImage(props: {
  file: Blob
  filename: string
  thumbnail?: boolean
}) {
  const resource = useMemo(
    () => createPaymentProofResource(props.file),
    [props.file]
  )
  const url = useSyncExternalStore(
    resource.subscribe,
    resource.getSnapshot,
    () => null
  )
  if (!url) return null
  return (
    <img
      src={url}
      alt={props.filename}
      className={
        props.thumbnail
          ? 'bg-muted h-24 w-full rounded-md object-contain'
          : 'max-h-[70vh] w-full rounded-lg object-contain'
      }
    />
  )
}

export function InvoicePaymentProofFiles(props: {
  files: File[]
  disabled: boolean
  onRemove: (index: number) => void
}) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<File | null>(null)
  return (
    <>
      <div className='grid gap-3 sm:grid-cols-3'>
        {props.files.map((file, index) => (
          <div
            key={`${file.name}-${file.size}-${file.lastModified}`}
            className='min-w-0 rounded-lg border p-2'
          >
            <Button
              type='button'
              variant='ghost'
              className='h-auto w-full p-0'
              aria-label={t('Preview payment screenshot {{filename}}', {
                filename: file.name,
              })}
              onClick={() => setPreview(file)}
            >
              <PaymentProofImage file={file} filename={file.name} thumbnail />
            </Button>
            <p className='mt-2 truncate text-xs' title={file.name}>
              {file.name}
            </p>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              className='mt-1 w-full'
              aria-label={t('Remove payment screenshot {{filename}}', {
                filename: file.name,
              })}
              disabled={props.disabled}
              onClick={() => props.onRemove(index)}
            >
              <Trash2 aria-hidden='true' />
              {t('Remove')}
            </Button>
          </div>
        ))}
      </div>
      <Dialog
        open={preview !== null}
        onOpenChange={(open) => {
          if (!open) setPreview(null)
        }}
        title={t('Payment screenshot')}
        description={preview?.name}
        contentClassName='sm:max-w-4xl'
      >
        {preview && (
          <PaymentProofImage file={preview} filename={preview.name} />
        )}
      </Dialog>
    </>
  )
}

function StoredPaymentProofPreview(props: {
  applicationId: number
  proof: InvoicePaymentProof
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['invoice', 'payment-proof', props.applicationId, props.proof.id],
    queryFn: ({ signal }) =>
      getInvoicePaymentProof(props.applicationId, props.proof.id, signal),
    retry: false,
    meta: { errorToast: false },
  })
  return (
    <Dialog
      open
      onOpenChange={props.onOpenChange}
      title={t('Payment screenshot')}
      description={props.proof.filename}
      contentClassName='sm:max-w-4xl'
    >
      {query.isPending && <LoadingState />}
      {query.isError && (
        <ErrorState
          title={t('Failed to load payment screenshot')}
          onRetry={() => query.refetch()}
        />
      )}
      {query.data && (
        <PaymentProofImage file={query.data} filename={props.proof.filename} />
      )}
    </Dialog>
  )
}

export function InvoicePaymentProofs(props: {
  applicationId: number
  proofs: InvoicePaymentProof[]
}) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<InvoicePaymentProof | null>(null)
  return (
    <section className='grid gap-3'>
      <h3 className='font-medium'>{t('Payment screenshots')}</h3>
      {props.proofs.length === 0 ? (
        <p className='text-muted-foreground text-sm'>
          {t('No payment screenshots')}
        </p>
      ) : (
        <div className='flex flex-wrap gap-2'>
          {props.proofs.map((proof) => (
            <Button
              key={proof.id}
              type='button'
              variant='outline'
              className='max-w-full'
              aria-label={t('View payment screenshot {{filename}}', {
                filename: proof.filename,
              })}
              onClick={() => setPreview(proof)}
            >
              <ImageIcon aria-hidden='true' />
              <span className='truncate'>{proof.filename}</span>
            </Button>
          ))}
        </div>
      )}
      {preview && (
        <StoredPaymentProofPreview
          applicationId={props.applicationId}
          proof={preview}
          onOpenChange={(open) => {
            if (!open) setPreview(null)
          }}
        />
      )}
    </section>
  )
}
