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
import { Download, FileJson } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api } from '@/lib/api'

import type { ImageGenerationLog } from '../../types'

interface Props {
  log: ImageGenerationLog
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface LoadedImage {
  previewUrl: string
  sourceUrl: string
  blob?: Blob
}

export function ImageGenerationPreviewDialog(props: Props) {
  const { t } = useTranslation()
  const [images, setImages] = useState<LoadedImage[]>([])
  const [loading, setLoading] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!props.open || props.log.image_urls.length === 0) return
    let cancelled = false
    const objectUrls: string[] = []

    async function loadImages() {
      setLoading(true)
      setFailed(false)
      try {
        const responses = await Promise.all(
          props.log.image_urls.map(async (url) => {
            if (url.startsWith('https://') || url.startsWith('http://')) {
              return { previewUrl: url, sourceUrl: url }
            }
            const response = await api.get(url, { responseType: 'blob' })
            const blob = response.data as Blob
            const objectUrl = URL.createObjectURL(blob)
            objectUrls.push(objectUrl)
            return { previewUrl: objectUrl, sourceUrl: url, blob }
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
  }, [props.log.image_urls, props.open])

  const jsonData = useMemo(
    () => ({
      id: props.log.id,
      task_id: props.log.task_id,
      status: props.log.status,
      request_id: props.log.request_id,
      created_at: props.log.created_at,
      updated_at: props.log.updated_at,
      user_id: props.log.user_id,
      username: props.log.username,
      token_id: props.log.token_id,
      token_name: props.log.token_name,
      channel_id: props.log.channel_id,
      channel_name: props.log.channel_name,
      model: props.log.model_name,
      prompt: props.log.prompt,
      size: props.log.size,
      quality: props.log.quality,
      image_count: props.log.image_count,
      image_urls: props.log.image_urls,
      quota: props.log.quota,
      duration_seconds: props.log.use_time,
      error_message: props.log.error_message,
    }),
    [props.log]
  )
  const formattedJson = useMemo(
    () => JSON.stringify(jsonData, null, 2),
    [jsonData]
  )

  const downloadBlob = (blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
  }

  const handleImageDownload = (image: LoadedImage, index: number) => {
    const filename = `image-${props.log.request_id || props.log.id}-${index + 1}`
    if (image.blob) {
      downloadBlob(image.blob, filename)
      return
    }
    const link = document.createElement('a')
    link.href = image.sourceUrl
    link.download = filename
    link.target = '_blank'
    link.rel = 'noopener noreferrer'
    link.click()
  }

  const handleJsonDownload = () => {
    downloadBlob(
      new Blob([formattedJson], { type: 'application/json;charset=utf-8' }),
      `image-log-${props.log.request_id || props.log.id}.json`
    )
  }

  let content = (
    <div className='grid gap-3 py-4 sm:grid-cols-2'>
      {images.map((image, index) => (
        <div key={image.previewUrl} className='group relative'>
          <img
            src={image.previewUrl}
            alt={`${t('Generated image')} ${index + 1}`}
            className='bg-muted aspect-square w-full rounded-lg border object-contain'
          />
          <Button
            type='button'
            variant='secondary'
            size='icon'
            className='absolute top-2 right-2 size-9 shadow-sm'
            title={t('Download image')}
            aria-label={t('Download image')}
            onClick={() => handleImageDownload(image, index)}
          >
            <Download className='size-4' />
          </Button>
        </div>
      ))}
    </div>
  )
  if (loading) {
    content = (
      <div className='grid gap-3 py-4 sm:grid-cols-2'>
        {props.log.image_urls.map((url) => (
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
      description={props.log.request_id || t('Image Generation Logs')}
      contentClassName='sm:max-w-4xl'
      contentHeight='auto'
    >
      <Tabs defaultValue='images'>
        <TabsList>
          <TabsTrigger value='images'>{t('Images')}</TabsTrigger>
          <TabsTrigger value='json'>{t('JSON data')}</TabsTrigger>
        </TabsList>
        <TabsContent value='images'>
          <ScrollArea className='max-h-[70vh]'>{content}</ScrollArea>
        </TabsContent>
        <TabsContent value='json'>
          <div className='flex justify-end py-3'>
            <Button
              type='button'
              variant='outline'
              onClick={handleJsonDownload}
            >
              <FileJson className='size-4' />
              {t('Download JSON')}
            </Button>
          </div>
          <ScrollArea className='bg-muted/30 max-h-[60vh] rounded-lg border'>
            <pre className='min-w-0 p-4 font-mono text-xs leading-5 break-words whitespace-pre-wrap'>
              {formattedJson}
            </pre>
          </ScrollArea>
        </TabsContent>
      </Tabs>
    </Dialog>
  )
}
