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
import { Copy, Link2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { CustomEndpoint } from '@/features/custom-endpoints/types'
import { useStatus } from '@/hooks/use-status'
import { copyToClipboard } from '@/lib/copy-to-clipboard'

export function CustomEndpoints() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const endpoints = (status?.custom_endpoints || []) as CustomEndpoint[]

  if (endpoints.length === 0) return null

  const copyEndpoint = async (endpoint: CustomEndpoint) => {
    const copied = await copyToClipboard(endpoint.url)
    toast[copied ? 'success' : 'error'](
      copied ? t('Endpoint copied') : t('Copy failed')
    )
  }

  return (
    <section className='space-y-3'>
      <div>
        <h2 className='text-sm font-semibold'>{t('Custom Endpoints')}</h2>
        <p className='text-muted-foreground text-xs'>
          {t('Click an endpoint to copy its address.')}
        </p>
      </div>
      <div className='flex flex-wrap gap-2'>
        {endpoints.map((endpoint) => (
          <Tooltip key={endpoint.id}>
            <TooltipTrigger
              render={
                <Button
                  type='button'
                  variant='outline'
                  className='h-9 max-w-full gap-2 px-3'
                  onClick={() => copyEndpoint(endpoint)}
                />
              }
            >
              <Link2 className='size-4 shrink-0 text-teal-600' />
              <span className='shrink-0 font-medium'>{endpoint.name}</span>
              <span className='text-border'>|</span>
              <span className='truncate font-mono text-xs text-teal-700 dark:text-teal-400'>
                {endpoint.url}
              </span>
              <Copy className='text-muted-foreground size-3.5 shrink-0' />
            </TooltipTrigger>
            <TooltipContent side='top' className='max-w-xs items-start'>
              <div className='space-y-1'>
                <div className='font-medium'>{endpoint.name}</div>
                <div>{endpoint.description}</div>
                <div className='text-teal-300'>
                  {t('Click to copy this endpoint')}
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </section>
  )
}
