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
import * as z from 'zod'

export const SUANLIAI_VIDEO_PLUGIN_KEY = 'suanliai-video'
export const SUANLIAI_GATEWAY_BASE_URL = 'https://suanliai.top'
export const SUANLIAI_VIDEO_MODELS = [
  '专享sd2.0(fast933/4-15秒/720p)',
  '专享sd2.5(30图10音/4-30秒/720p)',
] as const

const seconds = z.number().finite().int().min(1).max(3600)
const referenceCount = z.number().int().min(0).max(128)
export const videoModelProfileSchema = z
  .object({
    enabled: z.boolean().optional(),
    min_seconds: seconds.optional(),
    max_seconds: seconds.optional(),
    default_seconds: seconds.optional(),
    supported_seconds: z.array(seconds).max(64).optional(),
    aspect_ratios: z.array(z.string().trim().min(1).max(20)).max(24).optional(),
    resolutions: z.array(z.string().trim().min(1).max(20)).max(24).optional(),
    default_aspect_ratio: z.string().trim().max(20).optional(),
    default_resolution: z.string().trim().max(20).optional(),
    supports_images: z.boolean().optional(),
    supports_videos: z.boolean().optional(),
    supports_audios: z.boolean().optional(),
    supports_face: z.boolean().optional(),
    supports_generate_audio: z.boolean().optional(),
    max_images: referenceCount.optional(),
    max_videos: referenceCount.optional(),
    max_audios: referenceCount.optional(),
    face_default: z.boolean().optional(),
    timeout_seconds: z.number().int().min(1).max(1800).optional(),
    max_concurrency: z.number().int().min(0).max(10000).optional(),
    billing_mode: z.enum(['inherit', 'per_second', 'per_task']).optional(),
    unit_price: z.number().finite().min(0).max(100000000).optional(),
    unit_cost: z.number().finite().min(0).max(100000000).optional(),
    cost_currency: z.enum(['USD', 'CNY']).optional(),
  })
  .superRefine((profile, context) => {
    if (
      profile.billing_mode &&
      profile.billing_mode !== 'inherit' &&
      profile.unit_price === undefined
    ) {
      context.addIssue({
        code: 'custom',
        message:
          'A selling unit price is required for the selected billing mode',
        path: ['unit_price'],
      })
    }
    if (
      profile.min_seconds !== undefined &&
      profile.max_seconds !== undefined &&
      profile.min_seconds > profile.max_seconds
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Minimum duration cannot exceed maximum duration',
        path: ['min_seconds'],
      })
    }
    if (
      profile.default_seconds !== undefined &&
      ((profile.min_seconds !== undefined &&
        profile.default_seconds < profile.min_seconds) ||
        (profile.max_seconds !== undefined &&
          profile.default_seconds > profile.max_seconds) ||
        (profile.supported_seconds?.length &&
          !profile.supported_seconds.includes(profile.default_seconds)))
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Default duration must be supported by this model',
        path: ['default_seconds'],
      })
    }
    if (
      profile.default_aspect_ratio &&
      profile.aspect_ratios?.length &&
      !profile.aspect_ratios.includes(profile.default_aspect_ratio)
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Default aspect ratio must be supported by this model',
        path: ['default_aspect_ratio'],
      })
    }
    if (
      profile.default_resolution &&
      profile.resolutions?.length &&
      !profile.resolutions.includes(profile.default_resolution)
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Default resolution must be supported by this model',
        path: ['default_resolution'],
      })
    }
  })

export type VideoModelProfile = z.infer<typeof videoModelProfileSchema>
export type VideoModelConfigs = Record<string, VideoModelProfile>
export const videoModelConfigsSchema = z.record(
  z.string().trim().min(1).max(500),
  videoModelProfileSchema
)

// Limits verified in the provider's public documentation. A model's branding is
// not used to infer reference-video support or prices.
export function createSuanliVideoModelPresets(): VideoModelConfigs {
  const shared: VideoModelProfile = {
    enabled: true,
    min_seconds: 4,
    supports_images: true,
    supports_videos: false,
    supports_audios: true,
    supports_face: true,
    max_videos: 0,
    resolutions: ['720p'],
  }
  return {
    [SUANLIAI_VIDEO_MODELS[0]]: {
      ...shared,
      max_seconds: 15,
      max_images: 9,
      max_audios: 3,
    },
    [SUANLIAI_VIDEO_MODELS[1]]: {
      ...shared,
      max_seconds: 30,
      max_images: 30,
      max_audios: 10,
    },
  }
}
