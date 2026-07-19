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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { DateTimePicker } from '@/components/datetime-picker'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { api } from '@/lib/api'
import dayjs from '@/lib/dayjs'
import { formatTimestampToDate } from '@/lib/format'

import {
  getCurrentLogCleanupTask,
  getSystemTask,
  startLogCleanupTask,
} from '../api'
import {
  SettingsControlGroup,
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import type { LogCleanupTask } from '../types'

const logSettingsSchema = z.object({
  LogConsumeEnabled: z.boolean(),
  ImageGenerationLogEnabled: z.boolean(),
  ImageGenerationLogRetentionDays: z.number().int().min(0).max(3650),
  ImageGenerationLogPollingIntervalSeconds: z.number().int().min(5).max(3600),
  ImageGenerationLogImageAuthWhitelistEnabled: z.boolean(),
  ImageGenerationLogImageAuthWhitelist: z.string(),
})

type LogSettingsFormValues = z.infer<typeof logSettingsSchema>

type LogSettingsSectionProps = {
  defaultEnabled: boolean
  defaultImageLogEnabled: boolean
  defaultImageLogRetentionDays: number
  defaultImageLogPollingIntervalSeconds: number
  defaultImageLogImageAuthWhitelistEnabled: boolean
  defaultImageLogImageAuthWhitelist: string
}

type ServerLogInfo = {
  enabled: boolean
  log_dir: string
  file_count: number
  total_size: number
  oldest_time?: string
  newest_time?: string
}

type ImageStorageConfig = {
  enabled: boolean
  endpoint: string
  bucket: string
  region: string
  access_key: string
  secret_key: string
  has_secret_key: boolean
  use_ssl: boolean
  use_path_style: boolean
  object_prefix: string
}

const defaultImageStorageConfig: ImageStorageConfig = {
  enabled: false,
  endpoint: '',
  bucket: 'julong-media',
  region: 'us-east-1',
  access_key: '',
  secret_key: '',
  has_secret_key: false,
  use_ssl: true,
  use_path_style: true,
  object_prefix: 'generated/images',
}

const HOURS_IN_DAY = 24

function formatBytes(bytes: number, decimals = 2): string {
  if (!bytes || Number.isNaN(bytes)) return '0 Bytes'
  if (bytes === 0) return '0 Bytes'
  if (bytes < 0) return `-${formatBytes(-bytes, decimals)}`
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k))
  if (i < 0 || i >= sizes.length) return `${bytes} Bytes`
  return `${Number.parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${
    sizes[i]
  }`
}

const getDateHoursAgo = (hours: number) => {
  const date = new Date()
  date.setHours(date.getHours() - hours)
  return date
}

const getDateDaysAgo = (days: number) => getDateHoursAgo(days * HOURS_IN_DAY)

const quickSelectOptions = [
  {
    label: '24 hours ago',
    getValue: () => getDateHoursAgo(24),
  },
  {
    label: '7 days ago',
    getValue: () => getDateDaysAgo(7),
  },
  {
    label: '30 days ago',
    getValue: () => getDateDaysAgo(30),
  },
]

function ImageStorageSettings() {
  const { t } = useTranslation()
  const [config, setConfig] = useState<ImageStorageConfig>(
    defaultImageStorageConfig
  )
  const [loading, setLoading] = useState(true)
  const [testing, setTesting] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    async function fetchImageStorageConfig() {
      try {
        const response = await api.get('/api/performance/image-storage')
        if (!cancelled && response.data.success && response.data.data) {
          setConfig({ ...defaultImageStorageConfig, ...response.data.data })
        }
      } catch {
        // Keep defaults when the current administrator cannot load this config.
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void fetchImageStorageConfig()
    return () => {
      cancelled = true
    }
  }, [])

  const update = <K extends keyof ImageStorageConfig>(
    key: K,
    value: ImageStorageConfig[K]
  ) => setConfig((current) => ({ ...current, [key]: value }))

  const testConnection = async () => {
    setTesting(true)
    try {
      const response = await api.post(
        '/api/performance/image-storage/test',
        config
      )
      if (!response.data.success) {
        throw new Error(response.data.message || t('MinIO connection failed'))
      }
      toast.success(t('MinIO connection successful'))
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('MinIO connection failed')
      )
    } finally {
      setTesting(false)
    }
  }

  const save = async () => {
    setSaving(true)
    try {
      const response = await api.put('/api/performance/image-storage', config)
      if (!response.data.success) {
        throw new Error(
          response.data.message || t('Failed to save MinIO settings')
        )
      }
      setConfig({
        ...defaultImageStorageConfig,
        ...response.data.data,
        secret_key: '',
      })
      toast.success(t('MinIO settings saved'))
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save MinIO settings')
      )
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className='text-muted-foreground text-sm'>{t('Loading...')}</div>
    )
  }

  return (
    <div className='space-y-4 border-t pt-5'>
      <div>
        <h4 className='text-sm font-medium'>{t('MinIO image storage')}</h4>
        <p className='text-muted-foreground mt-1 text-sm'>
          {t(
            'Store completed async image results in the shared private media bucket.'
          )}
        </p>
      </div>
      <SettingsSwitchItem>
        <SettingsSwitchContent>
          <FormLabel>{t('Enable MinIO storage')}</FormLabel>
          <FormDescription>
            {t(
              'When upload fails, the image log falls back to the existing local or upstream result so completed tasks remain available.'
            )}
          </FormDescription>
        </SettingsSwitchContent>
        <Switch
          checked={config.enabled}
          onCheckedChange={(value) => update('enabled', value)}
        />
      </SettingsSwitchItem>
      <div className='grid gap-4 md:grid-cols-2'>
        <div className='space-y-2 md:col-span-2'>
          <Label>{t('MinIO endpoint')}</Label>
          <Input
            value={config.endpoint}
            onChange={(event) => update('endpoint', event.target.value)}
            placeholder='https://media.julongkj.top'
          />
        </div>
        <div className='space-y-2'>
          <Label>{t('Bucket')}</Label>
          <Input
            value={config.bucket}
            onChange={(event) => update('bucket', event.target.value)}
          />
        </div>
        <div className='space-y-2'>
          <Label>{t('Region')}</Label>
          <Input
            value={config.region}
            onChange={(event) => update('region', event.target.value)}
          />
        </div>
        <div className='space-y-2'>
          <Label>{t('Access Key')}</Label>
          <Input
            value={config.access_key}
            onChange={(event) => update('access_key', event.target.value)}
            autoComplete='off'
          />
        </div>
        <div className='space-y-2'>
          <Label>{t('Secret Key')}</Label>
          <Input
            type='password'
            value={config.secret_key}
            onChange={(event) => update('secret_key', event.target.value)}
            placeholder={
              config.has_secret_key
                ? t('Configured; leave blank to keep unchanged')
                : ''
            }
            autoComplete='new-password'
          />
        </div>
        <div className='space-y-2 md:col-span-2'>
          <Label>{t('Generated image object prefix')}</Label>
          <Input
            value={config.object_prefix}
            onChange={(event) => update('object_prefix', event.target.value)}
            placeholder='generated/images'
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Objects use generated/images/YYYY/MM/DD/{sha256}.{ext}; the same SHA-256 result is stored only once.'
            )}
          </p>
        </div>
      </div>
      <div className='flex flex-wrap gap-6'>
        <label className='flex items-center gap-2 text-sm'>
          <Switch
            checked={config.use_ssl}
            onCheckedChange={(value) => update('use_ssl', value)}
          />
          {t('Use HTTPS')}
        </label>
        <label className='flex items-center gap-2 text-sm'>
          <Switch
            checked={config.use_path_style}
            onCheckedChange={(value) => update('use_path_style', value)}
          />
          {t('Use S3 path style')}
        </label>
      </div>
      <div className='flex flex-wrap gap-3'>
        <Button
          type='button'
          variant='outline'
          disabled={testing || saving}
          onClick={testConnection}
        >
          {testing ? t('Testing...') : t('Test connection')}
        </Button>
        <Button type='button' disabled={saving || testing} onClick={save}>
          {saving ? t('Saving...') : t('Save MinIO settings')}
        </Button>
      </div>
    </div>
  )
}

function isActiveLogCleanupTask(task: LogCleanupTask | null) {
  return task?.status === 'pending' || task?.status === 'running'
}

export function LogSettingsSection({
  defaultEnabled,
  defaultImageLogEnabled,
  defaultImageLogRetentionDays,
  defaultImageLogPollingIntervalSeconds,
  defaultImageLogImageAuthWhitelistEnabled,
  defaultImageLogImageAuthWhitelist,
}: LogSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<LogSettingsFormValues>({
    resolver: zodResolver(logSettingsSchema),
    defaultValues: {
      LogConsumeEnabled: defaultEnabled,
      ImageGenerationLogEnabled: defaultImageLogEnabled,
      ImageGenerationLogRetentionDays: defaultImageLogRetentionDays,
      ImageGenerationLogPollingIntervalSeconds:
        defaultImageLogPollingIntervalSeconds,
      ImageGenerationLogImageAuthWhitelistEnabled:
        defaultImageLogImageAuthWhitelistEnabled,
      ImageGenerationLogImageAuthWhitelist: defaultImageLogImageAuthWhitelist,
    },
  })

  const [purgeDate, setPurgeDate] = useState<Date | undefined>(() =>
    getDateDaysAgo(30)
  )
  const [isStartingLogCleanup, setIsStartingLogCleanup] = useState(false)
  const [logCleanupTask, setLogCleanupTask] = useState<LogCleanupTask | null>(
    null
  )
  const [showConfirmDialog, setShowConfirmDialog] = useState(false)
  const [serverLogInfo, setServerLogInfo] = useState<ServerLogInfo | null>(null)
  const [serverLogCleanupMode, setServerLogCleanupMode] = useState('by_count')
  const [serverLogCleanupValue, setServerLogCleanupValue] = useState(10)
  const [serverLogCleanupLoading, setServerLogCleanupLoading] = useState(false)
  const imageAuthWhitelistEnabled = form.watch(
    'ImageGenerationLogImageAuthWhitelistEnabled'
  )

  const fetchServerLogInfo = useCallback(async () => {
    try {
      const res = await api.get('/api/performance/logs')
      if (res.data.success) setServerLogInfo(res.data.data)
    } catch {
      /* ignore */
    }
  }, [])

  useEffect(() => {
    form.reset({
      LogConsumeEnabled: defaultEnabled,
      ImageGenerationLogEnabled: defaultImageLogEnabled,
      ImageGenerationLogRetentionDays: defaultImageLogRetentionDays,
      ImageGenerationLogPollingIntervalSeconds:
        defaultImageLogPollingIntervalSeconds,
      ImageGenerationLogImageAuthWhitelistEnabled:
        defaultImageLogImageAuthWhitelistEnabled,
      ImageGenerationLogImageAuthWhitelist: defaultImageLogImageAuthWhitelist,
    })
  }, [
    defaultEnabled,
    defaultImageLogEnabled,
    defaultImageLogImageAuthWhitelist,
    defaultImageLogImageAuthWhitelistEnabled,
    defaultImageLogPollingIntervalSeconds,
    defaultImageLogRetentionDays,
    form,
  ])

  useEffect(() => {
    fetchServerLogInfo()
  }, [fetchServerLogInfo])

  useEffect(() => {
    let cancelled = false

    async function fetchCurrentLogCleanupTask() {
      try {
        const res = await getCurrentLogCleanupTask()
        if (!cancelled && res.success && res.data) {
          setLogCleanupTask(res.data)
        }
      } catch {
        /* ignore */
      }
    }

    fetchCurrentLogCleanupTask()

    return () => {
      cancelled = true
    }
  }, [])

  const purgeTimestamp = useMemo(() => {
    if (!purgeDate) return null
    return Math.floor(purgeDate.getTime() / 1000)
  }, [purgeDate])

  const formattedPurgeDate = useMemo(() => {
    if (!purgeDate) return ''
    return formatTimestampToDate(purgeDate.getTime(), 'milliseconds')
  }, [purgeDate])

  const logCleanupActive = isActiveLogCleanupTask(logCleanupTask)
  const logCleanupState = logCleanupTask?.state
  const logCleanupProgress = Math.min(
    100,
    Math.max(0, logCleanupState?.progress ?? 0)
  )
  const logCleanupProcessed = logCleanupState?.processed ?? 0
  const logCleanupTotal = logCleanupState?.total ?? 0
  const logCleanupTaskId = logCleanupTask?.task_id

  useEffect(() => {
    if (!logCleanupTaskId || !logCleanupActive) return

    let cancelled = false
    const interval = window.setInterval(async () => {
      try {
        const res = await getSystemTask(logCleanupTaskId)
        if (cancelled || !res.success || !res.data) return

        setLogCleanupTask(res.data)
        if (!isActiveLogCleanupTask(res.data)) {
          if (res.data.status === 'succeeded') {
            const count =
              res.data.result?.deleted_count ?? res.data.state?.processed ?? 0
            toast.success(
              count > 0
                ? t('{{count}} log entries removed.', { count })
                : t('No log entries matched the selected time.')
            )
          } else if (res.data.status === 'failed') {
            toast.error(res.data.error || t('Failed to clean logs'))
          }
        }
      } catch {
        /* keep polling */
      }
    }, 1000)

    return () => {
      cancelled = true
      window.clearInterval(interval)
    }
  }, [logCleanupActive, logCleanupTaskId, t])

  const onSubmit = async (values: LogSettingsFormValues) => {
    const updates = [
      {
        key: 'LogConsumeEnabled',
        value: values.LogConsumeEnabled,
      },
      {
        key: 'ImageGenerationLogEnabled',
        value: values.ImageGenerationLogEnabled,
      },
      {
        key: 'ImageGenerationLogRetentionDays',
        value: values.ImageGenerationLogRetentionDays,
      },
      {
        key: 'ImageGenerationLogPollingIntervalSeconds',
        value: values.ImageGenerationLogPollingIntervalSeconds,
      },
      {
        key: 'ImageGenerationLogImageAuthWhitelist',
        value: values.ImageGenerationLogImageAuthWhitelist,
      },
      {
        key: 'ImageGenerationLogImageAuthWhitelistEnabled',
        value: values.ImageGenerationLogImageAuthWhitelistEnabled,
      },
    ] as Array<{ key: string; value: boolean | number | string }>

    for (const update of updates) {
      const result = await updateOption.mutateAsync(update)
      if (!result.success) return
    }
  }

  const handleRequestCleanLogs = () => {
    if (!purgeTimestamp) {
      toast.error(t('Select a timestamp before clearing logs.'))
      return
    }

    setShowConfirmDialog(true)
  }

  const handleCleanLogs = async () => {
    if (!purgeTimestamp) {
      toast.error(t('Select a timestamp before clearing logs.'))
      return
    }

    setIsStartingLogCleanup(true)
    try {
      const res = await startLogCleanupTask(purgeTimestamp)
      if (!res.success) {
        throw new Error(res.message || t('Failed to clean logs'))
      }
      if (!res.data) {
        throw new Error(t('Failed to clean logs'))
      }
      setLogCleanupTask(res.data)
      setShowConfirmDialog(false)
      toast.success(t('Log cleanup task started.'))
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('Failed to clean logs')
      toast.error(message)
    } finally {
      setIsStartingLogCleanup(false)
    }
  }

  const cleanupServerLogFiles = async () => {
    if (
      !serverLogCleanupValue ||
      Number.isNaN(serverLogCleanupValue) ||
      serverLogCleanupValue < 1
    ) {
      toast.error(t('Please enter a valid number'))
      return
    }

    setServerLogCleanupLoading(true)
    try {
      const res = await api.delete(
        `/api/performance/logs?mode=${serverLogCleanupMode}&value=${serverLogCleanupValue}`
      )
      if (res.data.success) {
        const { deleted_count, freed_bytes } = res.data.data
        toast.success(
          t('Cleaned up {{count}} log files, freed {{size}}', {
            count: deleted_count,
            size: formatBytes(freed_bytes),
          })
        )
      } else {
        toast.error(res.data.message || t('Cleanup failed'))
      }
      fetchServerLogInfo()
    } catch {
      toast.error(t('Cleanup failed'))
    } finally {
      setServerLogCleanupLoading(false)
    }
  }

  return (
    <SettingsSection title={t('Log Maintenance')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save log settings'
          />
          <FormField
            control={form.control}
            name='LogConsumeEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Record quota usage')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Track per-request consumption to power usage analytics. Keeping this on increases database writes.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
                <FormMessage />
              </SettingsSwitchItem>
            )}
          />

          <SettingsControlGroup className='flex flex-col gap-4'>
            <FormField
              control={form.control}
              name='ImageGenerationLogEnabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Record image generation logs')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Store prompts and generated images for successful image generation requests. Images use local disk space.'
                      )}
                    </FormDescription>
                    <FormDescription>
                      {t(
                        'When disabled, async image generation falls back to synchronous responses and generated images are not stored.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormMessage />
                </SettingsSwitchItem>
              )}
            />
            <FormField
              control={form.control}
              name='ImageGenerationLogRetentionDays'
              render={({ field }) => (
                <div className='flex flex-col gap-2'>
                  <FormLabel>{t('Image log retention days')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      max={3650}
                      className='max-w-40'
                      value={field.value}
                      onBlur={field.onBlur}
                      name={field.name}
                      ref={field.ref}
                      onChange={(event) =>
                        field.onChange(Number(event.target.value))
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Set to 0 to keep image generation logs indefinitely.')}
                  </FormDescription>
                  <FormMessage />
                </div>
              )}
            />
            <FormField
              control={form.control}
              name='ImageGenerationLogPollingIntervalSeconds'
              render={({ field }) => (
                <div className='flex flex-col gap-2'>
                  <FormLabel>
                    {t('Image task polling interval (seconds)')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={5}
                      max={3600}
                      className='max-w-40'
                      value={field.value}
                      onBlur={field.onBlur}
                      name={field.name}
                      ref={field.ref}
                      onChange={(event) =>
                        field.onChange(Number(event.target.value))
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Refresh unfinished image tasks at this interval. Allowed range: 5-3600 seconds.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </div>
              )}
            />
            <FormField
              control={form.control}
              name='ImageGenerationLogImageAuthWhitelistEnabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>
                      {t('Allow unauthenticated image reads from whitelist')}
                    </FormLabel>
                    <FormDescription>
                      {t(
                        'When enabled, only matching IP addresses or browser Origin/Referer domains can read generated image files without an API Key.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormMessage />
                </SettingsSwitchItem>
              )}
            />
            {imageAuthWhitelistEnabled && (
              <FormField
                control={form.control}
                name='ImageGenerationLogImageAuthWhitelist'
                render={({ field }) => (
                  <div className='flex flex-col gap-2'>
                    <FormLabel>{t('Image read whitelist')}</FormLabel>
                    <FormControl>
                      <Textarea
                        rows={5}
                        className='max-w-2xl font-mono text-sm'
                        placeholder={'example.com\n203.0.113.10'}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Enter one exact domain or IP address per line. Full HTTP/HTTPS URLs are accepted; wildcards and CIDR ranges are not supported.'
                      )}
                    </FormDescription>
                    <Alert variant='destructive'>
                      <AlertDescription>
                        {t(
                          '0.0.0.0 allows everyone to read generated image files without authentication. Task status queries and image generation still require an API Key.'
                        )}
                      </AlertDescription>
                    </Alert>
                    <FormMessage />
                  </div>
                )}
              />
            )}
            <ImageStorageSettings />
          </SettingsControlGroup>

          <SettingsControlGroup className='space-y-3'>
            <div>
              <h4 className='text-sm font-medium'>{t('Clean history logs')}</h4>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Remove all log entries created before the selected timestamp.'
                )}
              </p>
            </div>
            <DateTimePicker value={purgeDate} onChange={setPurgeDate} />
            <div className='flex flex-wrap gap-3'>
              {quickSelectOptions.map((option) => (
                <Button
                  key={option.label}
                  type='button'
                  variant='outline'
                  onClick={() => setPurgeDate(option.getValue())}
                >
                  {t(option.label)}
                </Button>
              ))}
              <Button
                type='button'
                variant='destructive'
                onClick={handleRequestCleanLogs}
                disabled={isStartingLogCleanup || logCleanupActive}
              >
                {isStartingLogCleanup || logCleanupActive
                  ? t('Cleaning...')
                  : t('Clean logs')}
              </Button>
            </div>
            {logCleanupTask && (
              <div className='rounded-md border p-3'>
                <div className='mb-2 flex items-center justify-between gap-3 text-sm'>
                  <span className='font-medium'>
                    {t('Log cleanup progress')}
                  </span>
                  <span className='text-muted-foreground tabular-nums'>
                    {logCleanupProgress}%
                  </span>
                </div>
                <Progress value={logCleanupProgress} />
                <div className='text-muted-foreground mt-2 text-xs'>
                  {t('{{processed}} of {{total}} log entries processed.', {
                    processed: logCleanupProcessed,
                    total: logCleanupTotal,
                  })}
                </div>
                {logCleanupTask.status === 'failed' && logCleanupTask.error && (
                  <div className='text-destructive mt-2 text-xs'>
                    {logCleanupTask.error}
                  </div>
                )}
              </div>
            )}
          </SettingsControlGroup>
        </SettingsForm>
      </Form>

      <Separator />

      <div className='space-y-4'>
        <div>
          <h4 className='font-medium'>{t('Server Log Management')}</h4>
          <p className='text-muted-foreground mt-1 text-xs'>
            {t(
              'Manage server log files. Log files accumulate over time; regular cleanup is recommended to free disk space.'
            )}
          </p>
        </div>

        {serverLogInfo !== null &&
          (serverLogInfo.enabled ? (
            <div className='space-y-4'>
              <div className='rounded-lg border p-4'>
                <div className='grid grid-cols-2 gap-2 text-sm md:grid-cols-4'>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Log Directory')}:
                    </span>{' '}
                    <span className='font-mono text-xs'>
                      {serverLogInfo.log_dir}
                    </span>
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Log File Count')}:
                    </span>{' '}
                    {serverLogInfo.file_count}
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Total Log Size')}:
                    </span>{' '}
                    {formatBytes(serverLogInfo.total_size)}
                  </div>
                  {serverLogInfo.oldest_time && serverLogInfo.newest_time && (
                    <div>
                      <span className='text-muted-foreground'>
                        {t('Date Range')}:
                      </span>{' '}
                      {dayjs(serverLogInfo.oldest_time).format('YYYY-MM-DD')} ~{' '}
                      {dayjs(serverLogInfo.newest_time).format('YYYY-MM-DD')}
                    </div>
                  )}
                </div>
              </div>

              <div className='flex flex-wrap items-end gap-3'>
                <div className='grid gap-1.5'>
                  <Label className='text-xs'>{t('Cleanup Mode')}</Label>
                  <Select
                    items={[
                      { value: 'by_count', label: t('Retain last N files') },
                      { value: 'by_days', label: t('Retain last N days') },
                    ]}
                    value={serverLogCleanupMode}
                    onValueChange={(value) =>
                      value !== null && setServerLogCleanupMode(value)
                    }
                  >
                    <SelectTrigger className='w-[160px]'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        <SelectItem value='by_count'>
                          {t('Retain last N files')}
                        </SelectItem>
                        <SelectItem value='by_days'>
                          {t('Retain last N days')}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </div>
                <div className='grid gap-1.5'>
                  <Label className='text-xs'>
                    {serverLogCleanupMode === 'by_count'
                      ? t('Files to Retain')
                      : t('Days to Retain')}
                  </Label>
                  <Input
                    type='number'
                    min={1}
                    max={serverLogCleanupMode === 'by_count' ? 1000 : 3650}
                    value={serverLogCleanupValue}
                    onChange={(event) =>
                      setServerLogCleanupValue(Number(event.target.value))
                    }
                    className='w-[120px]'
                  />
                </div>
                <AlertDialog>
                  <AlertDialogTrigger
                    render={
                      <Button
                        type='button'
                        variant='destructive'
                        size='sm'
                        disabled={serverLogCleanupLoading}
                      />
                    }
                  >
                    {serverLogCleanupLoading
                      ? t('Cleaning...')
                      : t('Clean Up Log Files')}
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>
                        {t('Confirm log file cleanup?')}
                      </AlertDialogTitle>
                      <AlertDialogDescription>
                        {serverLogCleanupMode === 'by_count'
                          ? t(
                              'Only the last {{value}} log files will be retained; the rest will be deleted.',
                              {
                                value: serverLogCleanupValue,
                              }
                            )
                          : t(
                              'Log files older than {{value}} days will be deleted.',
                              {
                                value: serverLogCleanupValue,
                              }
                            )}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
                      <AlertDialogAction
                        variant='destructive'
                        onClick={cleanupServerLogFiles}
                      >
                        {t('Confirm Cleanup')}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </div>
          ) : (
            <Alert>
              <AlertDescription>
                {t(
                  'Server logging is not enabled (log directory not configured)'
                )}
              </AlertDescription>
            </Alert>
          ))}
      </div>

      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Confirm log cleanup')}</AlertDialogTitle>
            <AlertDialogDescription>
              {formattedPurgeDate
                ? t(
                    'This will permanently remove all log entries created before {{date}}.',
                    { date: formattedPurgeDate }
                  )
                : t(
                    'This will permanently remove log entries before the selected timestamp.'
                  )}{' '}
              {t('This action cannot be undone.')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isStartingLogCleanup}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              onClick={handleCleanLogs}
              disabled={isStartingLogCleanup}
            >
              {isStartingLogCleanup ? t('Cleaning...') : t('Delete logs')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}
