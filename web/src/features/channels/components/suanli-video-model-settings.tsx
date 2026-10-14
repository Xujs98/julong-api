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
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { EmptyState } from '@/components/empty-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

import {
  createSuanliVideoModelPresets,
  videoModelProfileSchema,
  type VideoModelConfigs,
  type VideoModelProfile,
} from '../lib/suanliai-video'

const booleanFields = [
  { key: 'supports_images', label: 'Reference images' },
  { key: 'supports_videos', label: 'Reference videos' },
  { key: 'supports_audios', label: 'Reference audio' },
  { key: 'supports_face', label: 'Face processing' },
  { key: 'supports_generate_audio', label: 'Generate audio' },
] as const
const numberFields = [
  {
    key: 'max_images',
    label: 'Maximum reference images',
    min: 0,
    max: 128,
    step: '1',
  },
  {
    key: 'max_videos',
    label: 'Maximum reference videos',
    min: 0,
    max: 128,
    step: '1',
  },
  {
    key: 'max_audios',
    label: 'Maximum reference audio files',
    min: 0,
    max: 128,
    step: '1',
  },
  {
    key: 'min_seconds',
    label: 'Minimum duration (seconds)',
    min: 1,
    max: 3600,
    step: 'any',
  },
  {
    key: 'max_seconds',
    label: 'Maximum duration (seconds)',
    min: 1,
    max: 3600,
    step: 'any',
  },
  {
    key: 'default_seconds',
    label: 'Default duration (seconds)',
    min: 1,
    max: 3600,
    step: 'any',
  },
  {
    key: 'timeout_seconds',
    label: 'Request timeout (seconds)',
    min: 1,
    max: 1800,
    step: '1',
  },
  {
    key: 'max_concurrency',
    label: 'Maximum concurrent tasks',
    min: 0,
    max: 1280,
    step: '1',
  },
  {
    key: 'unit_price',
    label: 'Video selling unit price (USD)',
    min: 0,
    max: 12800000,
    step: 'any',
  },
  {
    key: 'unit_cost',
    label: 'Video cost unit price',
    min: 0,
    max: 12800000,
    step: 'any',
  },
] as const
const listFields = [
  {
    key: 'aspect_ratios',
    label: 'Supported aspect ratios',
    placeholder: '16:9, 9:16, 1:1',
  },
  {
    key: 'resolutions',
    label: 'Supported resolutions',
    placeholder: '480p, 720p, 1080p',
  },
  {
    key: 'supported_seconds',
    label: 'Supported durations (seconds)',
    placeholder: '5, 8, 10',
  },
] as const

export function SuanliVideoModelSettings(props: {
  models: string[]
  value: VideoModelConfigs
  onChange: (value: VideoModelConfigs) => void
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const id = useId()
  const [selectedModel, setSelectedModel] = useState(props.models[0] || '')
  const [listDrafts, setListDrafts] = useState<
    Record<string, { text: string; serialized: string | undefined }>
  >({})
  const model = props.models.includes(selectedModel)
    ? selectedModel
    : props.models[0] || ''
  const profile = props.value[model] || {}
  const preset = createSuanliVideoModelPresets()[model]
  const validation = videoModelProfileSchema.safeParse(profile)
  const requestExample = JSON.stringify(
    {
      model,
      prompt: '{{prompt}}',
      seconds: profile.default_seconds || '{{seconds}}',
      aspect_ratio: profile.default_aspect_ratio || '{{aspect_ratio}}',
      resolution: profile.default_resolution || '{{resolution}}',
      images: [],
      videos: [],
      audios: [],
    },
    null,
    2
  )
  const update = (patch: Partial<VideoModelProfile>) => {
    props.onChange({ ...props.value, [model]: { ...profile, ...patch } })
  }

  return (
    <section
      aria-label={t('Video model capabilities')}
      className='border-border/60 space-y-5 rounded-xl border p-4 sm:p-5'
    >
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div className='space-y-1'>
          <h3 className='font-medium'>{t('Video model capabilities')}</h3>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Configure each model independently. These limits and defaults are enforced when submitting video tasks.'
            )}
          </p>
        </div>
        <Badge variant='secondary'>SuanliAI · /v1/videos</Badge>
      </div>
      <div className='grid gap-3 sm:grid-cols-3'>
        {[
          { label: t('Video submission path'), value: '/v1/videos' },
          {
            label: t('Video status query path'),
            value: '/v1/videos/{task_id}',
          },
          {
            label: t('Video content path'),
            value: '/v1/videos/{task_id}/content',
          },
        ].map((path, index) => (
          <Field key={path.value}>
            <FieldLabel htmlFor={`${id}-path-${index}`}>
              {path.label}
            </FieldLabel>
            <Input
              id={`${id}-path-${index}`}
              value={path.value}
              readOnly
              className='font-mono text-xs'
            />
          </Field>
        ))}
      </div>
      <p className='text-muted-foreground text-xs'>
        {t(
          'Asynchronous querying is supported. Upstream cancellation, webhooks, and batch generation are not available in this protocol.'
        )}
      </p>
      {props.models.length === 0 ? (
        <EmptyState
          title={t('Add a model to configure its video capabilities')}
        />
      ) : (
        <fieldset
          disabled={props.disabled}
          className='space-y-5 disabled:opacity-60'
        >
          <div className='flex flex-wrap items-end gap-3'>
            <Field className='min-w-0 flex-1'>
              <FieldLabel htmlFor={`${id}-model`}>
                {t('Video model')}
              </FieldLabel>
              <Select
                disabled={props.disabled}
                value={model}
                onValueChange={(value) => {
                  if (value) setSelectedModel(value)
                }}
              >
                <SelectTrigger id={`${id}-model`} className='w-full min-w-0'>
                  <SelectValue>{model}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {props.models.map((item) => (
                    <SelectItem key={item} value={item}>
                      {item}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            {preset && (
              <Button
                type='button'
                variant='outline'
                onClick={() =>
                  props.onChange({ ...props.value, [model]: preset })
                }
              >
                {t('Apply documented model preset')}
              </Button>
            )}
            <Button
              type='button'
              variant='ghost'
              onClick={() => {
                const next = { ...props.value }
                delete next[model]
                props.onChange(next)
              }}
            >
              {t('Reset model overrides')}
            </Button>
          </div>
          <Field orientation='horizontal'>
            <Checkbox
              disabled={props.disabled}
              id={`${id}-enabled`}
              checked={profile.enabled !== false}
              onCheckedChange={(checked) =>
                update({ enabled: checked === true })
              }
            />
            <FieldLabel htmlFor={`${id}-enabled`}>
              {t('Enable this video model')}
            </FieldLabel>
          </Field>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Blank values and indeterminate checkboxes inherit protocol defaults. An unknown model has no verified provider capability preset; enter limits from its documentation.'
            )}
          </p>
          <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
            {booleanFields.map((field) => (
              <Field key={field.key} orientation='horizontal'>
                <Checkbox
                  disabled={props.disabled}
                  id={`${id}-${field.key}`}
                  checked={profile[field.key] === true}
                  indeterminate={profile[field.key] === undefined}
                  onCheckedChange={(checked) =>
                    update({ [field.key]: checked === true })
                  }
                />
                <FieldLabel htmlFor={`${id}-${field.key}`}>
                  {t(field.label)}
                </FieldLabel>
              </Field>
            ))}
            <Field orientation='horizontal'>
              <Checkbox
                id={`${id}-face-default`}
                checked={profile.face_default === true}
                disabled={props.disabled || profile.supports_face === false}
                onCheckedChange={(checked) =>
                  update({ face_default: checked === true })
                }
              />
              <FieldLabel htmlFor={`${id}-face-default`}>
                {t('Enable face processing by default')}
              </FieldLabel>
            </Field>
          </div>
          <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
            {numberFields.map((field) => (
              <Field key={field.key}>
                <FieldLabel htmlFor={`${id}-${field.key}`}>
                  {t(field.label)}
                </FieldLabel>
                <Input
                  id={`${id}-${field.key}`}
                  type='number'
                  min={field.min}
                  max={field.max}
                  step={field.step}
                  value={profile[field.key] ?? ''}
                  placeholder={t('Inherit')}
                  onChange={(event) =>
                    update({
                      [field.key]:
                        event.target.value === ''
                          ? undefined
                          : Number(event.target.value),
                    })
                  }
                />
              </Field>
            ))}
            {listFields.map((field) => (
              <Field key={field.key}>
                <FieldLabel htmlFor={`${id}-${field.key}`}>
                  {t(field.label)}
                </FieldLabel>
                <Input
                  id={`${id}-${field.key}`}
                  value={
                    listDrafts[`${model}:${field.key}`]?.serialized ===
                    JSON.stringify(profile[field.key])
                      ? listDrafts[`${model}:${field.key}`]?.text || ''
                      : profile[field.key]?.join(', ') || ''
                  }
                  placeholder={field.placeholder}
                  onChange={(event) => {
                    const items = event.target.value
                      .split(/[,，]/)
                      .map((item) => item.trim())
                      .filter(Boolean)
                    const nextValue =
                      field.key === 'supported_seconds'
                        ? items.map(Number)
                        : items
                    setListDrafts((previous) => ({
                      ...previous,
                      [`${model}:${field.key}`]: {
                        text: event.target.value,
                        serialized: JSON.stringify(
                          nextValue.length ? nextValue : undefined
                        ),
                      },
                    }))
                    if (field.key === 'supported_seconds') {
                      update({
                        supported_seconds: items.length
                          ? items.map(Number)
                          : undefined,
                      })
                    } else {
                      update({ [field.key]: items.length ? items : undefined })
                    }
                  }}
                />
              </Field>
            ))}
            {[
              { key: 'default_aspect_ratio', label: 'Default aspect ratio' },
              { key: 'default_resolution', label: 'Default resolution' },
            ].map((field) => (
              <Field key={field.key}>
                <FieldLabel htmlFor={`${id}-${field.key}`}>
                  {t(field.label)}
                </FieldLabel>
                <Input
                  id={`${id}-${field.key}`}
                  value={
                    profile[
                      field.key as 'default_aspect_ratio' | 'default_resolution'
                    ] || ''
                  }
                  placeholder={t('Inherit')}
                  onChange={(event) =>
                    update({ [field.key]: event.target.value || undefined })
                  }
                />
              </Field>
            ))}
            <Field>
              <FieldLabel htmlFor={`${id}-billing`}>
                {t('Video billing mode')}
              </FieldLabel>
              <Select
                disabled={props.disabled}
                value={profile.billing_mode || 'inherit'}
                items={[
                  { value: 'inherit', label: t('Inherit model pricing') },
                  { value: 'per_second', label: t('Per second') },
                  { value: 'per_task', label: t('Per task') },
                ]}
                onValueChange={(value) =>
                  update({
                    billing_mode: value as VideoModelProfile['billing_mode'],
                  })
                }
              >
                <SelectTrigger id={`${id}-billing`} className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='inherit'>
                    {t('Inherit model pricing')}
                  </SelectItem>
                  <SelectItem value='per_second'>{t('Per second')}</SelectItem>
                  <SelectItem value='per_task'>{t('Per task')}</SelectItem>
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel htmlFor={`${id}-currency`}>
                {t('Cost currency')}
              </FieldLabel>
              <Select
                disabled={props.disabled}
                value={profile.cost_currency || 'USD'}
                items={[
                  { value: 'USD', label: t('US dollar (USD)') },
                  { value: 'CNY', label: t('Chinese yuan (CNY)') },
                ]}
                onValueChange={(value) =>
                  update({
                    cost_currency: value as VideoModelProfile['cost_currency'],
                  })
                }
              >
                <SelectTrigger id={`${id}-currency`} className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='USD'>{t('US dollar (USD)')}</SelectItem>
                  <SelectItem value='CNY'>{t('Chinese yuan (CNY)')}</SelectItem>
                </SelectContent>
              </Select>
            </Field>
          </div>
          <FieldDescription>
            {t(
              'Per-second or per-task billing requires an administrator-defined USD selling price; zero means free. Inherited pricing keeps existing model prices. Costs are audit metadata. Zero concurrent tasks means no additional per-model limit.'
            )}
          </FieldDescription>
          {!validation.success && (
            <div role='alert' className='text-destructive text-sm'>
              {validation.error.issues
                .map((issue) => t(issue.message))
                .join(' · ')}
            </div>
          )}
          <Field>
            <div className='flex items-center justify-between gap-2'>
              <FieldLabel htmlFor={`${id}-template`}>
                {t('Unified video request example')}
              </FieldLabel>
              <CopyButton value={requestExample} />
            </div>
            <Textarea
              id={`${id}-template`}
              value={requestExample}
              readOnly
              rows={8}
              className='font-mono text-xs'
            />
            <FieldDescription>
              {t(
                'Response fields: id, status, progress, url, error.message, and optional origin_task_id. Clients submit to the gateway and query the returned task ID.'
              )}
            </FieldDescription>
          </Field>
        </fieldset>
      )}
    </section>
  )
}
