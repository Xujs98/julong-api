import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'

interface Props {
  imageUrls: string[]
  requestId?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ImageGenerationPreviewDialog(props: Props) {
  const { t } = useTranslation()
  const [images, setImages] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!props.open || props.imageUrls.length === 0) return
    let cancelled = false
    const objectUrls: string[] = []

    async function loadImages() {
      setLoading(true)
      setFailed(false)
      try {
        const responses = await Promise.all(
          props.imageUrls.map(async (url) => {
            if (url.startsWith('https://') || url.startsWith('http://')) {
              return url
            }
            const response = await api.get(url, { responseType: 'blob' })
            const objectUrl = URL.createObjectURL(response.data as Blob)
            objectUrls.push(objectUrl)
            return objectUrl
          })
        )
        if (cancelled) return
        setImages(responses)
      } catch {
        if (!cancelled) setFailed(true)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadImages()
    return () => {
      cancelled = true
      objectUrls.forEach((url) => URL.revokeObjectURL(url))
      setImages([])
    }
  }, [props.imageUrls, props.open])

  let content = (
    <div className='grid gap-3 py-4 sm:grid-cols-2'>
      {images.map((url, index) => (
        <img
          key={url}
          src={url}
          alt={`${t('Generated image')} ${index + 1}`}
          className='bg-muted aspect-square w-full rounded-lg border object-contain'
        />
      ))}
    </div>
  )
  if (loading) {
    content = (
      <div className='grid gap-3 py-4 sm:grid-cols-2'>
        {props.imageUrls.map((url) => (
          <Skeleton key={url} className='aspect-square w-full' />
        ))}
      </div>
    )
  } else if (failed) {
    content = (
      <div className='text-muted-foreground py-16 text-center text-sm'>
        {t('Failed to load image')}
      </div>
    )
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Generated Images')}
      description={props.requestId || t('Image Generation Logs')}
      contentClassName='sm:max-w-4xl'
      contentHeight='auto'
    >
      <ScrollArea className='max-h-[70vh]'>{content}</ScrollArea>
    </Dialog>
  )
}
