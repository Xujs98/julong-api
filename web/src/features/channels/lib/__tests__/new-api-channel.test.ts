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
import { describe, expect, test } from 'vitest'

import type { TaskPluginOption } from '../../api'
import {
  CHANNEL_TYPE_NEW_API,
  CHANNEL_TYPE_VLLM,
  CHANNEL_TYPE_SGLANG,
  CHANNEL_TYPE_OPTIONS,
  MODEL_FETCHABLE_TYPES,
} from '../../constants'
import { channelSchema } from '../../types'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
} from '../channel-form'
import {
  getChannelPluginExtensions,
  readTaskExtendPluginKeys,
  supportsNewAPIUpstream,
} from '../channel-plugin-extensions'
import { getChannelTypeConfig } from '../channel-type-config'
import { getChannelTypeIcon, getKeyPromptForType } from '../channel-utils'
import {
  createSuanliVideoModelPresets,
  SUANLIAI_VIDEO_MODELS,
} from '../suanliai-video'

function newAPIForm(baseUrl: string) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'New API upstream',
    type: CHANNEL_TYPE_NEW_API,
    base_url: baseUrl,
    key: 'test-key',
    models: 'gpt-5',
  }
}

describe('New API channel', () => {
  test('registers selection, ordering, model discovery, and icon metadata', () => {
    const option = CHANNEL_TYPE_OPTIONS.find(
      (item) => item.value === CHANNEL_TYPE_NEW_API
    )

    expect(option).toEqual({
      value: CHANNEL_TYPE_NEW_API,
      label: 'New API',
    })
    expect(
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_NEW_API
      ) + 1
    ).toBe(CHANNEL_TYPE_OPTIONS.findIndex((item) => item.value === 58))
    expect(MODEL_FETCHABLE_TYPES.has(CHANNEL_TYPE_NEW_API)).toBe(true)
    expect(getChannelTypeIcon(CHANNEL_TYPE_NEW_API)).toBe('NewAPI')
    expect(getKeyPromptForType(CHANNEL_TYPE_NEW_API)).toBe(
      'Enter API key for this channel'
    )
    expect(getChannelTypeConfig(CHANNEL_TYPE_NEW_API).icon).toBe('NewAPI')
  })

  test('requires a non-blank Base URL', () => {
    const blankResult = channelFormSchema.safeParse(newAPIForm('  '))

    expect(blankResult.success).toBe(false)
    if (!blankResult.success) {
      expect(
        blankResult.error.issues.some(
          (issue) =>
            issue.path[0] === 'base_url' &&
            issue.message === 'Base URL is required for this channel type'
        )
      ).toBe(true)
    }

    expect(
      channelFormSchema.safeParse(newAPIForm('https://new-api.example')).success
    ).toBe(true)
  })

  test('keeps Sub2API Base URL validation unchanged', () => {
    const result = channelFormSchema.safeParse({
      ...newAPIForm(''),
      type: 59,
    })

    expect(result.success).toBe(true)
  })
})

describe.each([
  { type: CHANNEL_TYPE_VLLM, name: 'vLLM', icon: 'Vllm' },
  { type: CHANNEL_TYPE_SGLANG, name: 'SGLang', icon: 'SGLang' },
])('$name channel', ({ type, name, icon }) => {
  test('can be selected and discover served models', () => {
    expect(CHANNEL_TYPE_OPTIONS).toContainEqual({
      value: type,
      label: name,
    })
    expect(MODEL_FETCHABLE_TYPES.has(type)).toBe(true)
    expect(getChannelTypeIcon(type)).toBe(icon)
    expect(getChannelTypeConfig(type).icon).toBe(icon)
    expect(getKeyPromptForType(type)).toBe(
      `${name} API key, or EMPTY if authentication is disabled`
    )
  })

  test('requires an upstream address and submits the served model name', () => {
    const form = {
      ...newAPIForm(''),
      type,
      models: 'deepseek-v4-flash-vision-exp',
      key: 'EMPTY',
    }
    const blank = channelFormSchema.safeParse(form)
    expect(blank.success).toBe(false)
    if (!blank.success) {
      expect(blank.error.issues).toEqual(
        expect.arrayContaining([
          expect.objectContaining({
            path: ['base_url'],
            message: 'Base URL is required for this channel type',
          }),
        ])
      )
    }
    const parsed = channelFormSchema.parse({
      ...form,
      base_url: 'http://vllm:8000/',
    })
    const payload = transformFormDataToCreatePayload(parsed)
    expect(payload.channel).toMatchObject({
      type,
      base_url: 'http://vllm:8000',
      models: 'deepseek-v4-flash-vision-exp',
      key: 'EMPTY',
    })
  })
})

describe('New API channel task plugin extensions', () => {
  const plugins: TaskPluginOption[] = [
    {
      key: 'doubao',
      name: 'Doubao',
      models: ['seedance'],
      channelTypes: [45],
      upstreams: ['vendor', 'new_api'],
    },
    {
      key: 'kling',
      name: 'Kling',
      models: ['kling-v2'],
      upstreams: ['new_api'],
    },
    {
      key: 'hailuo',
      name: 'Hailuo',
      models: ['minimax'],
      upstreams: ['vendor'],
    },
  ]

  test('only plugins declaring a New API upstream can be bound to a New API channel', () => {
    expect(plugins.filter(supportsNewAPIUpstream).map((p) => p.key)).toEqual([
      'doubao',
      'kling',
    ])
    expect(supportsNewAPIUpstream({ upstreams: null })).toBe(false)
    expect(supportsNewAPIUpstream({})).toBe(false)
  })

  test('reads the extension list together with a single key on New API channels only', () => {
    expect(
      readTaskExtendPluginKeys(CHANNEL_TYPE_NEW_API, {
        task_plugin_key: 'kling',
        task_extend_plugin_keys: ['doubao', 'kling', '', 'doubao'],
      })
    ).toEqual(['kling', 'doubao'])
    expect(readTaskExtendPluginKeys(CHANNEL_TYPE_NEW_API, {})).toEqual([])
    expect(
      readTaskExtendPluginKeys(1, { task_extend_plugin_keys: ['doubao'] })
    ).toEqual([])
  })

  test('lists only bound plugins as extensions while other types keep declared channel types', () => {
    expect(
      getChannelPluginExtensions(CHANNEL_TYPE_NEW_API, plugins, ['kling'])
    ).toEqual([plugins[1]])
    expect(getChannelPluginExtensions(CHANNEL_TYPE_NEW_API, plugins)).toEqual(
      []
    )
    expect(getChannelPluginExtensions(45, plugins, ['kling'])).toEqual([
      plugins[0],
    ])
  })

  test('round-trips bound plugins through the channel setting for New API channels only', () => {
    const channel = channelSchema.parse({
      id: 7,
      name: 'Gateway',
      type: CHANNEL_TYPE_NEW_API,
      key: '',
      status: 1,
      created_time: 1,
      test_time: 0,
      response_time: 0,
      balance_updated_time: 0,
      other: '',
      remark: '',
      models: 'seedance',
      group: 'default',
      base_url: 'https://gateway.example',
      setting: JSON.stringify({
        task_plugin_key: 'kling',
        task_extend_plugin_keys: ['doubao'],
      }),
    })
    expect(
      transformChannelToFormDefaults(channel).task_extend_plugin_keys
    ).toEqual(['kling', 'doubao'])

    const gateway = transformFormDataToCreatePayload(
      channelFormSchema.parse({
        ...newAPIForm('https://gateway.example'),
        task_extend_plugin_keys: ['doubao', 'kling'],
      })
    )
    const gatewaySetting = JSON.parse(gateway.channel.setting ?? '{}')
    expect(gatewaySetting).toMatchObject({
      task_extend_plugin_keys: ['doubao', 'kling'],
    })
    expect(gatewaySetting).not.toHaveProperty('task_plugin_key')

    const openai = transformFormDataToCreatePayload(
      channelFormSchema.parse({
        ...newAPIForm('https://gateway.example'),
        type: 1,
        task_extend_plugin_keys: ['doubao'],
      })
    )
    expect(JSON.parse(openai.channel.setting ?? '{}')).not.toHaveProperty(
      'task_extend_plugin_keys'
    )
  })
})

describe('Suanli AI gateway model configuration', () => {
  test('round-trips exact Chinese model names, disabled capability flags, zero limits, and separate pricing', () => {
    const profiles = createSuanliVideoModelPresets()
    profiles[SUANLIAI_VIDEO_MODELS[1]] = {
      ...profiles[SUANLIAI_VIDEO_MODELS[1]],
      default_seconds: 10,
      billing_mode: 'per_task',
      unit_price: 0.12,
      unit_cost: 0.5,
      cost_currency: 'CNY',
    }
    const form = channelFormSchema.parse({
      ...newAPIForm('https://suanliai.top'),
      models: SUANLIAI_VIDEO_MODELS.join(','),
      gateway_provider: 'suanliai',
      task_extend_plugin_keys: ['suanliai-video'],
      video_model_configs: profiles,
    })
    const payload = transformFormDataToCreatePayload(form)
    const setting = JSON.parse(payload.channel.setting || '{}')
    expect(setting.gateway_provider).toBe('suanliai')
    expect(setting.video_model_configs).toEqual(profiles)
    const channel = channelSchema.parse({
      ...payload.channel,
      id: 7,
      created_time: 1,
      test_time: 0,
      response_time: 0,
      balance_updated_time: 0,
      other: '',
      remark: '',
    })
    const restored = transformChannelToFormDefaults(channel)
    expect(restored.gateway_provider).toBe('suanliai')
    expect(restored.video_model_configs).toEqual(profiles)
    expect(
      restored.video_model_configs?.[SUANLIAI_VIDEO_MODELS[0]]
    ).toMatchObject({
      supports_videos: false,
      max_videos: 0,
      max_seconds: 15,
      max_images: 9,
    })
  })

  test.each([
    { min_seconds: 30, max_seconds: 15 },
    { default_seconds: 20, max_seconds: 15 },
    { max_seconds: 3601 },
    { max_images: -1 },
    { supported_seconds: [5, 8], default_seconds: 10 },
    { resolutions: ['720p'], default_resolution: '1080p' },
  ])(
    'rejects unsupported or unsafe video capability configuration %#',
    (profile) => {
      expect(
        channelFormSchema.safeParse({
          ...newAPIForm('https://suanliai.top'),
          video_model_configs: { 'custom.video': profile },
        }).success
      ).toBe(false)
    }
  )

  test('changing away from a gateway omits its protocol marker and per-model settings', () => {
    const payload = transformFormDataToCreatePayload({
      ...newAPIForm('https://suanliai.top'),
      type: 1,
      gateway_provider: 'suanliai',
      video_model_configs: createSuanliVideoModelPresets(),
    })
    const setting = JSON.parse(payload.channel.setting || '{}')
    expect(setting).not.toHaveProperty('gateway_provider')
    expect(setting).not.toHaveProperty('video_model_configs')
  })
})
