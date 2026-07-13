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
import { Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { CustomEndpoint } from '@/features/custom-endpoints/types'

import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

function parseEndpoints(data: string): CustomEndpoint[] {
  try {
    const parsed = JSON.parse(data || '[]')
    if (!Array.isArray(parsed)) return []
    return parsed.map((item, index) => ({
      id: item.id || index + 1,
      name: item.name || '',
      url: item.url || '',
      description: item.description || '',
    }))
  } catch {
    return []
  }
}

export function CustomEndpointsSection({ data }: { data: string }) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [endpoints, setEndpoints] = useState<CustomEndpoint[]>(() =>
    parseEndpoints(data)
  )
  const [changed, setChanged] = useState(false)

  useEffect(() => {
    if (!changed) setEndpoints(parseEndpoints(data))
  }, [changed, data])

  const updateEndpoint = (id: number, patch: Partial<CustomEndpoint>) => {
    setEndpoints((current) =>
      current.map((endpoint) =>
        endpoint.id === id ? { ...endpoint, ...patch } : endpoint
      )
    )
    setChanged(true)
  }

  const addEndpoint = () => {
    const id = Math.max(0, ...endpoints.map((endpoint) => endpoint.id)) + 1
    setEndpoints((current) => [
      ...current,
      { id, name: '', url: '', description: '' },
    ])
    setChanged(true)
  }

  const save = async () => {
    if (
      endpoints.some(
        (endpoint) =>
          !endpoint.name.trim() ||
          !endpoint.url.trim() ||
          !endpoint.description.trim()
      )
    ) {
      toast.error(t('Complete all custom endpoint fields'))
      return
    }
    try {
      for (const endpoint of endpoints) new URL(endpoint.url)
      await updateOption.mutateAsync({
        key: 'console_setting.custom_endpoints',
        value: JSON.stringify(endpoints),
      })
      setChanged(false)
      toast.success(t('Custom endpoints saved'))
    } catch {
      toast.error(t('Failed to save custom endpoints'))
    }
  }

  return (
    <SettingsSection title={t('Custom Endpoints')}>
      <div className='space-y-4'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Add extra API endpoint addresses for quick copying on the API Keys page.'
            )}
          </p>
          <Button
            size='sm'
            variant='secondary'
            disabled={!changed || updateOption.isPending}
            onClick={save}
          >
            <Save className='size-4' />
            {t('Save Settings')}
          </Button>
        </div>

        {endpoints.map((endpoint, index) => (
          <div key={endpoint.id} className='space-y-4 rounded-md border p-4'>
            <div className='flex items-center justify-between'>
              <h3 className='font-medium'>
                {t('Endpoint #{{index}}', { index: index + 1 })}
              </h3>
              <Button
                type='button'
                size='icon-sm'
                variant='ghost'
                title={t('Delete')}
                onClick={() => {
                  setEndpoints((current) =>
                    current.filter((item) => item.id !== endpoint.id)
                  )
                  setChanged(true)
                }}
              >
                <Trash2 className='text-destructive size-4' />
              </Button>
            </div>
            <div className='grid gap-4 md:grid-cols-2'>
              <div className='space-y-2'>
                <Label>{t('Name')}</Label>
                <Input
                  value={endpoint.name}
                  placeholder={t('e.g., OpenAI Compatible')}
                  onChange={(event) =>
                    updateEndpoint(endpoint.id, { name: event.target.value })
                  }
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Endpoint URL')}</Label>
                <Input
                  value={endpoint.url}
                  placeholder='https://api2.example.com'
                  onChange={(event) =>
                    updateEndpoint(endpoint.id, { url: event.target.value })
                  }
                />
              </div>
            </div>
            <div className='space-y-2'>
              <Label>{t('Introduction')}</Label>
              <Input
                value={endpoint.description}
                placeholder={t('e.g., Supports OpenAI-format requests')}
                onChange={(event) =>
                  updateEndpoint(endpoint.id, {
                    description: event.target.value,
                  })
                }
              />
            </div>
          </div>
        ))}

        <Button
          type='button'
          variant='outline'
          className='h-14 w-full border-dashed'
          disabled={endpoints.length >= 20}
          onClick={addEndpoint}
        >
          <Plus className='size-4' />
          {t('Add Endpoint')}
        </Button>
      </div>
    </SettingsSection>
  )
}
