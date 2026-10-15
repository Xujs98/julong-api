import { Eye, Trash2, Upload } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'

export function SupportQRCodeField(props: {
  id: string
  url?: string
  file?: File
  disabled: boolean
  onChange: (file: File | null) => void
}) {
  const { t } = useTranslation()
  const input = useRef<HTMLInputElement>(null)
  const [localURL, setLocalURL] = useState('')
  const [preview, setPreview] = useState(false)
  useEffect(() => {
    return () => {
      if (localURL) URL.revokeObjectURL(localURL)
    }
  }, [localURL, props.file])
  const url = props.file ? localURL : props.url
  return (
    <div className='space-y-2 sm:col-span-4'>
      <Label htmlFor={`qr-${props.id}`}>{t('Group QR code')}</Label>
      <input
        ref={input}
        id={`qr-${props.id}`}
        type='file'
        className='sr-only'
        accept='image/png,image/jpeg,image/webp'
        disabled={props.disabled}
        onChange={(event) => {
          const file = event.target.files?.[0]
          event.target.value = ''
          if (!file) return
          if (
            !['image/png', 'image/jpeg', 'image/webp'].includes(file.type) ||
            file.size > 5 * 1024 * 1024
          ) {
            toast.error(
              t(
                'QR code must be a PNG, JPG or WebP image up to 5 MB and 4096 pixels'
              )
            )
            return
          }
          setLocalURL(URL.createObjectURL(file))
          props.onChange(file)
        }}
      />
      <div className='flex flex-wrap items-center gap-2'>
        {url && (
          <img
            src={url}
            alt={t('Group QR code')}
            className='size-20 rounded border bg-white object-contain p-1'
          />
        )}
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={props.disabled}
          onClick={() => input.current?.click()}
        >
          <Upload className='size-4' />
          {t('Upload QR code')}
        </Button>
        {url && (
          <>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => setPreview(true)}
            >
              <Eye className='size-4' />
              {t('View QR code')}
            </Button>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={props.disabled}
              onClick={() => {
                setPreview(false)
                props.onChange(null)
              }}
            >
              <Trash2 className='size-4' />
              {t('Delete QR code')}
            </Button>
          </>
        )}
      </div>
      <p className='text-muted-foreground text-xs'>
        {t('QR code changes take effect after saving.')}
      </p>
      <Dialog
        open={preview}
        onOpenChange={setPreview}
        title={t('Group QR code')}
        contentClassName='sm:max-w-md'
      >
        {url && (
          <img
            src={url}
            alt={t('Group QR code')}
            className='mx-auto max-h-[60dvh] max-w-full bg-white object-contain p-3'
          />
        )}
      </Dialog>
    </div>
  )
}
