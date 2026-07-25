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
import { zodResolver } from '@hookform/resolvers/zod'
import { ImageIcon, Trash2, Upload } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useWatch, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

import { deleteSiteLogo, uploadSiteLogo } from '../api'
import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const maxLogoFileBytes = 5 * 1024 * 1024
const supportedLogoTypes = new Set(['image/png', 'image/jpeg', 'image/webp'])

function isValidLogoURL(value: string): boolean {
  if (value === '' || value.startsWith('/')) return true
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

const _systemInfoSchema = z.object({
  SystemName: z.string().min(1),
  ServerAddress: z.string().optional(),
  Logo: z.string().refine(isValidLogoURL),
  Footer: z.string().optional(),
  About: z.string().optional(),
  HomePageContent: z.string().optional(),
  legal: z.object({
    user_agreement: z.string().optional(),
    privacy_policy: z.string().optional(),
  }),
})

type SystemInfoFormValues = z.infer<typeof _systemInfoSchema>

type SystemInfoSectionProps = {
  defaultValues: SystemInfoFormValues
}

function normalizeValue(value: unknown): string {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : String(value)
}

export function SystemInfoSection({ defaultValues }: SystemInfoSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const logoFileInputRef = useRef<HTMLInputElement>(null)
  const [pendingLogoFile, setPendingLogoFile] = useState<File | null>(null)
  const [pendingLogoPreview, setPendingLogoPreview] = useState('')
  const [logoPreviewFailed, setLogoPreviewFailed] = useState(false)

  const normalizedDefaults: SystemInfoFormValues = {
    SystemName: normalizeValue(defaultValues.SystemName),
    ServerAddress: normalizeValue(defaultValues.ServerAddress),
    Logo: normalizeValue(defaultValues.Logo),
    Footer: normalizeValue(defaultValues.Footer),
    About: normalizeValue(defaultValues.About),
    HomePageContent: normalizeValue(defaultValues.HomePageContent),
    legal: {
      user_agreement: normalizeValue(defaultValues.legal?.user_agreement),
      privacy_policy: normalizeValue(defaultValues.legal?.privacy_policy),
    },
  }

  const systemInfoSchemaWithI18n = z.object({
    SystemName: z.string().min(1, {
      error: () => t('System name is required'),
    }),
    ServerAddress: z.string().optional(),
    Logo: z.string().refine(isValidLogoURL, {
      error: () => t('Enter a valid URL or upload an image'),
    }),
    Footer: z.string().optional(),
    About: z.string().optional(),
    HomePageContent: z.string().optional(),
    legal: z.object({
      user_agreement: z.string().optional(),
      privacy_policy: z.string().optional(),
    }),
  })

  const clearPendingLogo = () => {
    setPendingLogoFile(null)
    if (logoFileInputRef.current) logoFileInputRef.current.value = ''
  }

  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm<SystemInfoFormValues>({
      resolver: zodResolver(systemInfoSchemaWithI18n) as Resolver<
        SystemInfoFormValues,
        unknown,
        SystemInfoFormValues
      >,
      defaultValues: normalizedDefaults,
      externalDirty: pendingLogoFile !== null,
      onExternalReset: clearPendingLogo,
      onSubmit: async (data, changedFields) => {
        let uploadedLogoURL = ''
        const valuesToSave = { ...data }
        const fieldsToSave = { ...changedFields }

        try {
          if (pendingLogoFile) {
            const uploadResponse = await uploadSiteLogo(pendingLogoFile)
            if (!uploadResponse.success || !uploadResponse.data?.url) {
              toast.error(
                uploadResponse.message || t('Failed to upload logo image')
              )
              return false
            }
            uploadedLogoURL = uploadResponse.data.url
            valuesToSave.Logo = uploadedLogoURL
            fieldsToSave.Logo = uploadedLogoURL
          }

          for (const [key, value] of Object.entries(fieldsToSave)) {
            let v = normalizeValue(value)
            if (key === 'ServerAddress') {
              v = v.replace(/\/+$/, '')
              valuesToSave.ServerAddress = v
            }
            const response = await updateOption.mutateAsync({ key, value: v })
            if (!response.success) {
              if (uploadedLogoURL) await deleteSiteLogo(uploadedLogoURL)
              return false
            }
          }

          clearPendingLogo()
          return valuesToSave
        } catch {
          if (uploadedLogoURL) await deleteSiteLogo(uploadedLogoURL)
          return false
        }
      },
    })

  const savedLogoURL = useWatch({ control: form.control, name: 'Logo' })
  const logoPreviewURL = pendingLogoPreview || savedLogoURL

  useEffect(() => {
    if (!pendingLogoFile) {
      setPendingLogoPreview('')
      return
    }
    const objectURL = URL.createObjectURL(pendingLogoFile)
    setPendingLogoPreview(objectURL)
    return () => URL.revokeObjectURL(objectURL)
  }, [pendingLogoFile])

  useEffect(() => {
    setLogoPreviewFailed(false)
  }, [logoPreviewURL])

  const handleLogoFileChange = (file: File | undefined) => {
    if (!file) return
    if (!supportedLogoTypes.has(file.type)) {
      toast.error(t('Please select a PNG, JPG, or WebP image'))
      if (logoFileInputRef.current) logoFileInputRef.current.value = ''
      return
    }
    if (file.size > maxLogoFileBytes) {
      toast.error(t('Logo image cannot exceed 5 MB'))
      if (logoFileInputRef.current) logoFileInputRef.current.value = ''
      return
    }
    setPendingLogoFile(file)
  }

  const removeLogo = () => {
    clearPendingLogo()
    form.setValue('Logo', '', { shouldDirty: true, shouldValidate: true })
  }

  return (
    <>
      <FormNavigationGuard when={isDirty} />

      <SettingsSection title={t('System Information')}>
        <Form {...form}>
          <SettingsForm onSubmit={handleSubmit}>
            <SettingsPageFormActions
              onSave={handleSubmit}
              onReset={handleReset}
              isSaving={isSubmitting || updateOption.isPending}
              isResetDisabled={!isDirty}
            />
            <FormDirtyIndicator isDirty={isDirty} />
            <SettingsFormGrid>
              <FormField
                control={form.control}
                name='SystemName'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('System Name')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('New API')} {...field} />
                    </FormControl>
                    <FormDescription>
                      {t('The name displayed across the application')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='ServerAddress'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Server Address')}</FormLabel>
                    <FormControl>
                      <Input placeholder='https://yourdomain.com' {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'The public URL of your server, used for OAuth callbacks, webhooks, and other external integrations'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='Logo'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Logo URL')}</FormLabel>
                    <div className='flex items-start gap-3'>
                      <div className='bg-muted/40 flex size-20 shrink-0 items-center justify-center overflow-hidden rounded-md border'>
                        {logoPreviewURL && !logoPreviewFailed ? (
                          <img
                            src={logoPreviewURL}
                            alt={t('Logo preview')}
                            className='size-full object-contain p-2'
                            onError={() => setLogoPreviewFailed(true)}
                          />
                        ) : (
                          <ImageIcon
                            className='text-muted-foreground size-7'
                            aria-hidden='true'
                          />
                        )}
                      </div>
                      <div className='min-w-0 flex-1 space-y-2'>
                        <FormControl>
                          <Input
                            placeholder={t('https://example.com/logo.png')}
                            {...field}
                            onChange={(event) => {
                              clearPendingLogo()
                              field.onChange(event)
                            }}
                          />
                        </FormControl>
                        <div className='flex flex-wrap items-center gap-2'>
                          <input
                            ref={logoFileInputRef}
                            type='file'
                            accept='image/png,image/jpeg,image/webp'
                            className='hidden'
                            onChange={(event) =>
                              handleLogoFileChange(event.target.files?.[0])
                            }
                          />
                          <Button
                            type='button'
                            size='sm'
                            variant='outline'
                            onClick={() => logoFileInputRef.current?.click()}
                          >
                            <Upload data-icon='inline-start' />
                            {t('Upload')}
                          </Button>
                          {logoPreviewURL ? (
                            <Button
                              type='button'
                              size='icon-sm'
                              variant='destructive'
                              onClick={removeLogo}
                              aria-label={t('Remove logo')}
                              title={t('Remove logo')}
                            >
                              <Trash2 />
                            </Button>
                          ) : null}
                          {pendingLogoFile ? (
                            <span className='text-muted-foreground max-w-full truncate text-xs'>
                              {t('Selected: {{filename}}', {
                                filename: pendingLogoFile.name,
                              })}
                            </span>
                          ) : null}
                        </div>
                      </div>
                    </div>
                    <FormDescription>
                      {t(
                        'Enter an image URL or upload PNG, JPG, or WebP. Maximum 5 MB and 4096 x 4096 pixels.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='Footer'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Footer')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          '© 2025 Your Company. All rights reserved.'
                        )}
                        rows={4}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Footer text displayed at the bottom of pages')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='About'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('About')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          'Enter HTML code (e.g., <p>About us...</p>) or a URL (e.g., https://example.com) to embed as iframe'
                        )}
                        rows={4}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Supports HTML markup or iframe embedding. Enter HTML code directly, or provide a complete URL to automatically embed it as an iframe.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <SettingsFormGridItem span='full'>
                <FormField
                  control={form.control}
                  name='HomePageContent'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Home Page Content')}</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder={t('Welcome to our New API...')}
                          rows={6}
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Content displayed on the home page (supports Markdown)'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </SettingsFormGridItem>

              <FormField
                control={form.control}
                name='legal.user_agreement'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('User Agreement')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          'Provide Markdown, HTML, or an external URL for the user agreement'
                        )}
                        rows={6}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Leave empty to disable the agreement requirement. Supports Markdown, HTML, or a full URL to redirect users.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='legal.privacy_policy'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Privacy Policy')}</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={t(
                          'Provide Markdown, HTML, or an external URL for the privacy policy'
                        )}
                        rows={6}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Leave empty to disable the privacy policy requirement. Supports Markdown, HTML, or a full URL to redirect users.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGrid>
          </SettingsForm>
        </Form>
      </SettingsSection>
    </>
  )
}
