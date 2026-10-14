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
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { useState } from 'react'
import { I18nextProvider } from 'react-i18next'
import { expect, test, vi } from 'vitest'

import en from '@/i18n/locales/en.json'
import zh from '@/i18n/locales/zh.json'

import {
  createSuanliVideoModelPresets,
  SUANLIAI_VIDEO_MODELS,
  type VideoModelConfigs,
} from '../../lib/suanliai-video'
import { SuanliVideoModelSettings } from '../suanli-video-model-settings'

function Harness(props: {
  configs?: VideoModelConfigs
  models?: string[]
  onChange?: (value: VideoModelConfigs) => void
}) {
  const [value, setValue] = useState(
    props.configs || createSuanliVideoModelPresets()
  )
  return (
    <SuanliVideoModelSettings
      value={value}
      models={props.models || [...SUANLIAI_VIDEO_MODELS]}
      onChange={(next) => {
        setValue(next)
        props.onChange?.(next)
      }}
    />
  )
}

test('selecting different model names preserves independent limits and rejects reference videos in the verified presets', async () => {
  const user = userEvent.setup()
  const change = vi.fn()
  render(<Harness onChange={change} />)
  expect(screen.getByLabelText('Maximum duration (seconds)')).toHaveValue(15)
  expect(
    screen.getByRole('checkbox', { name: 'Reference videos' })
  ).not.toBeChecked()
  expect(screen.getByLabelText('Maximum reference videos')).toHaveValue(0)
  await user.clear(screen.getByLabelText('Maximum reference images'))
  await user.type(screen.getByLabelText('Maximum reference images'), '8')
  await user.click(screen.getByRole('combobox', { name: 'Video model' }))
  await user.click(
    screen.getByRole('option', { name: SUANLIAI_VIDEO_MODELS[1] })
  )
  expect(screen.getByLabelText('Maximum duration (seconds)')).toHaveValue(30)
  expect(screen.getByLabelText('Maximum reference images')).toHaveValue(30)
  await user.click(screen.getByRole('combobox', { name: 'Video model' }))
  await user.click(
    screen.getByRole('option', { name: SUANLIAI_VIDEO_MODELS[0] })
  )
  expect(screen.getByLabelText('Maximum reference images')).toHaveValue(8)
  expect(change.mock.lastCall?.[0][SUANLIAI_VIDEO_MODELS[1]].max_images).toBe(
    30
  )
})

test('typing comma separated aspect ratios retains separators and saves the complete list', async () => {
  const user = userEvent.setup()
  const change = vi.fn()
  render(
    <Harness
      configs={{ 'custom.video': {} }}
      models={['custom.video']}
      onChange={change}
    />
  )
  const input = screen.getByLabelText('Supported aspect ratios')
  await user.type(input, '16:9, 9:16')
  expect(input).toHaveValue('16:9, 9:16')
  expect(change.mock.lastCall?.[0]['custom.video'].aspect_ratios).toEqual([
    '16:9',
    '9:16',
  ])
  expect(
    screen.getByRole('checkbox', { name: 'Reference images' })
  ).toHaveAttribute('aria-checked', 'mixed')
})

test('an incompatible default duration displays a validation error before channel submission', async () => {
  const user = userEvent.setup()
  render(<Harness />)
  await user.type(screen.getByLabelText('Default duration (seconds)'), '30')
  expect(screen.getByRole('alert')).toHaveTextContent(
    'Default duration must be supported by this model'
  )
})

test('read-only model configuration prevents changes and keeps all protocol paths visible', async () => {
  const change = vi.fn()
  const user = userEvent.setup()
  render(
    <SuanliVideoModelSettings
      value={createSuanliVideoModelPresets()}
      models={[...SUANLIAI_VIDEO_MODELS]}
      onChange={change}
      disabled
    />
  )
  expect(screen.getByLabelText('Maximum duration (seconds)')).toBeDisabled()
  await user.click(screen.getByRole('checkbox', { name: 'Reference images' }))
  expect(change).not.toHaveBeenCalled()
  expect(screen.getByLabelText('Video submission path')).toHaveAttribute(
    'readonly'
  )
  expect(screen.getByLabelText('Video content path')).toHaveValue(
    '/v1/videos/{task_id}/content'
  )
  expect(
    screen.queryByRole('checkbox', { name: /cancel|webhook/i })
  ).not.toBeInTheDocument()
})

test('a channel with no model displays guidance instead of editable capability inputs', () => {
  render(<SuanliVideoModelSettings value={{}} models={[]} onChange={vi.fn()} />)
  expect(
    screen.getByText('Add a model to configure its video capabilities')
  ).toBeVisible()
  expect(
    screen.queryByLabelText('Maximum duration (seconds)')
  ).not.toBeInTheDocument()
})

test('Chinese model settings translate every capability label and validation message when switching language', async () => {
  const language = createInstance()
  await language.init({
    lng: 'zh',
    fallbackLng: false,
    nsSeparator: false,
    resources: { en, zh },
  })
  const user = userEvent.setup()
  render(
    <I18nextProvider i18n={language}>
      <Harness />
    </I18nextProvider>
  )
  const translated: Record<string, string> = zh.translation
  const checkboxLabels = [
    'Reference images',
    'Reference videos',
    'Reference audio',
    'Face processing',
    'Generate audio',
  ]
  const inputLabels = [
    'Maximum reference images',
    'Maximum reference videos',
    'Maximum reference audio files',
    'Minimum duration (seconds)',
    'Maximum duration (seconds)',
    'Default duration (seconds)',
    'Request timeout (seconds)',
    'Maximum concurrent tasks',
    'Video selling unit price (USD)',
    'Video cost unit price',
    'Supported aspect ratios',
    'Supported resolutions',
    'Supported durations (seconds)',
    'Default aspect ratio',
    'Default resolution',
  ]

  for (const label of checkboxLabels) {
    expect(translated[label]).toMatch(/\p{Script=Han}/u)
    expect(
      screen.getByRole('checkbox', { name: translated[label] })
    ).toBeVisible()
  }
  for (const label of inputLabels) {
    expect(translated[label]).toMatch(/\p{Script=Han}/u)
    expect(screen.getByLabelText(translated[label])).toBeVisible()
  }
  await user.type(
    screen.getByLabelText(translated['Default duration (seconds)']),
    '30'
  )
  const validationKey = 'Default duration must be supported by this model'
  expect(translated[validationKey]).toMatch(/\p{Script=Han}/u)
  expect(screen.getByRole('alert')).toHaveTextContent(translated[validationKey])

  await act(() => language.changeLanguage('en'))
  for (const label of checkboxLabels) {
    expect(screen.getByRole('checkbox', { name: label })).toBeVisible()
  }
  for (const label of inputLabels) {
    expect(screen.getByLabelText(label)).toBeVisible()
  }
  expect(screen.getByRole('alert')).toHaveTextContent(validationKey)
})

test('translated billing and currency choices preserve protocol values through selection and language changes', async () => {
  const language = createInstance()
  await language.init({
    lng: 'zh',
    fallbackLng: false,
    nsSeparator: false,
    resources: { en, zh },
  })
  const user = userEvent.setup()
  const change = vi.fn()
  render(
    <I18nextProvider i18n={language}>
      <Harness
        configs={{ 'custom.video': {} }}
        models={['custom.video']}
        onChange={change}
      />
    </I18nextProvider>
  )
  const billing = screen.getByRole('combobox', { name: '视频计费模式' })
  const currency = screen.getByRole('combobox', { name: '成本货币' })
  expect(billing).toHaveTextContent('继承模型定价')
  expect(billing).not.toHaveTextContent('inherit')
  expect(currency).toHaveTextContent('美元（USD）')

  await user.click(billing)
  await user.click(screen.getByRole('option', { name: '按秒' }))
  expect(change.mock.lastCall?.[0]['custom.video'].billing_mode).toBe(
    'per_second'
  )
  expect(billing).toHaveTextContent('按秒')
  await user.click(billing)
  await user.click(screen.getByRole('option', { name: '按任务' }))
  expect(change.mock.lastCall?.[0]['custom.video'].billing_mode).toBe(
    'per_task'
  )

  await user.click(currency)
  await user.click(screen.getByRole('option', { name: '人民币（CNY）' }))
  expect(currency).toHaveTextContent('人民币（CNY）')
  expect(change.mock.lastCall?.[0]['custom.video'].cost_currency).toBe('CNY')
  await user.click(currency)
  await user.click(screen.getByRole('option', { name: '美元（USD）' }))
  expect(change.mock.lastCall?.[0]['custom.video'].cost_currency).toBe('USD')

  await act(() => language.changeLanguage('en'))
  expect(
    screen.getByRole('combobox', { name: 'Video billing mode' })
  ).toHaveTextContent('Per task')
  expect(
    screen.getByRole('combobox', { name: 'Cost currency' })
  ).toHaveTextContent('US dollar (USD)')
  expect(change.mock.lastCall?.[0]['custom.video']).toMatchObject({
    billing_mode: 'per_task',
    cost_currency: 'USD',
  })
})
