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
import { useQuery } from '@tanstack/react-query'
import {
  CalendarClock,
  CircleGauge,
  Save,
  ShieldAlert,
  WalletCards,
  type LucideIcon,
} from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { EmailCampaignUserPicker } from './email-campaign-user-picker'
import {
  getEmailSettingsConfig,
  resolveEmailSettingsRecipients,
  searchEmailSettingsRecipients,
  updateEmailSettingsConfig,
  type EmailSettingsConfig,
} from './email-templates-api'

type AlertPanelProps = {
  icon: LucideIcon
  title: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  children?: ReactNode
}

function AlertPanel(props: AlertPanelProps) {
  const Icon = props.icon
  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex items-start justify-between gap-4 px-4 py-4 sm:px-5'>
        <div className='flex min-w-0 items-start gap-3'>
          <span className='bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md'>
            <Icon className='size-4.5' aria-hidden='true' />
          </span>
          <span className='min-w-0'>
            <span className='block text-sm font-semibold'>{props.title}</span>
            <span className='text-muted-foreground mt-1 block text-sm leading-5'>
              {props.description}
            </span>
          </span>
        </div>
        <Switch
          checked={props.checked}
          aria-label={props.title}
          onCheckedChange={props.onCheckedChange}
        />
      </div>
      {props.children && (
        <div className='bg-muted/15 grid gap-4 border-t px-4 py-4 sm:px-5'>
          {props.children}
        </div>
      )}
    </div>
  )
}

export function EmailAlertSettings() {
  const { t } = useTranslation()
  const configQuery = useQuery({
    queryKey: ['email-settings-config'],
    queryFn: async () => {
      const response = await getEmailSettingsConfig()
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load email settings'))
      }
      return response.data
    },
  })
  const [config, setConfig] = useState<EmailSettingsConfig | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (configQuery.data) setConfig(configQuery.data)
  }, [configQuery.data])

  const patchConfig = (patch: Partial<EmailSettingsConfig>) => {
    setConfig((current) => (current ? { ...current, ...patch } : current))
  }

  const save = async () => {
    if (!config) return
    if (config.low_balance_email_threshold < 0) {
      toast.error(t('Low balance threshold cannot be negative'))
      return
    }
    if (config.account_quota_email_threshold < 0) {
      toast.error(t('Account quota threshold cannot be negative'))
      return
    }
    setSaving(true)
    try {
      const response = await updateEmailSettingsConfig(config)
      if (!response.success || !response.data) throw new Error(response.message)
      setConfig(response.data)
      toast.success(t('Email settings saved'))
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save email settings')
      )
    } finally {
      setSaving(false)
    }
  }

  if (configQuery.isLoading || !config) {
    return (
      <div className='text-muted-foreground flex min-h-28 items-center justify-center rounded-lg border text-sm'>
        {t('Loading...')}
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
        <div>
          <h3 className='text-base font-semibold'>{t('Email reminders')}</h3>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t(
              'Configure automatic billing, subscription, and channel alerts.'
            )}
          </p>
        </div>
        <Button type='button' disabled={saving} onClick={save}>
          <Save className='size-4' />
          {t('Save settings')}
        </Button>
      </div>

      <div className='grid gap-3'>
        <AlertPanel
          icon={CalendarClock}
          title={t('Subscription expiry reminders')}
          description={t(
            'When enabled, reminders are sent 7, 3, and 1 days before expiry.'
          )}
          checked={config.subscription_expiry_reminder_enabled}
          onCheckedChange={(checked) =>
            patchConfig({ subscription_expiry_reminder_enabled: checked })
          }
        />

        <AlertPanel
          icon={WalletCards}
          title={t('Low balance reminder')}
          description={t(
            "Send an email when a user's wallet or subscription quota falls below the warning threshold."
          )}
          checked={config.low_balance_email_enabled}
          onCheckedChange={(checked) =>
            patchConfig({ low_balance_email_enabled: checked })
          }
        >
          <div className='grid gap-4 lg:grid-cols-2'>
            <div className='space-y-1.5'>
              <Label htmlFor='low-balance-threshold'>
                {t('Default warning threshold')}
              </Label>
              <Input
                id='low-balance-threshold'
                type='number'
                min={0}
                step={1}
                disabled={!config.low_balance_email_enabled}
                value={config.low_balance_email_threshold}
                onChange={(event) =>
                  patchConfig({
                    low_balance_email_threshold: Math.max(
                      0,
                      Number.parseInt(event.target.value, 10) || 0
                    ),
                  })
                }
              />
              <p className='text-muted-foreground text-xs'>
                {t('A user-specific notification threshold takes priority.')}
              </p>
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='low-balance-recharge-url'>
                {t('Recharge page URL')}
              </Label>
              <Input
                id='low-balance-recharge-url'
                type='url'
                placeholder='https://example.com/wallet'
                disabled={!config.low_balance_email_enabled}
                value={config.low_balance_email_recharge_url}
                onChange={(event) =>
                  patchConfig({
                    low_balance_email_recharge_url: event.target.value,
                  })
                }
              />
              <p className='text-muted-foreground text-xs'>
                {t('Leave blank to use the built-in wallet page.')}
              </p>
            </div>
          </div>
        </AlertPanel>

        <AlertPanel
          icon={CircleGauge}
          title={t('Account quota alert')}
          description={t(
            'Notify selected administrators when a channel account balance falls below the threshold.'
          )}
          checked={config.account_quota_email_enabled}
          onCheckedChange={(checked) =>
            patchConfig({ account_quota_email_enabled: checked })
          }
        >
          <div className='grid gap-4 lg:grid-cols-2'>
            <div className='space-y-1.5'>
              <Label htmlFor='account-quota-threshold'>
                {t('Account balance threshold (USD)')}
              </Label>
              <Input
                id='account-quota-threshold'
                type='number'
                min={0}
                step='0.01'
                disabled={!config.account_quota_email_enabled}
                value={config.account_quota_email_threshold}
                onChange={(event) =>
                  patchConfig({
                    account_quota_email_threshold:
                      Number.parseFloat(event.target.value) || 0,
                  })
                }
              />
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Triggered after a supported channel balance check crosses the threshold.'
                )}
              </p>
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='account-quota-recipients'>
                {t('Recipients')}
              </Label>
              <EmailCampaignUserPicker
                id='account-quota-recipients'
                labelKey='Alert recipients'
                queryKeyPrefix='account-quota-email'
                searchUsers={searchEmailSettingsRecipients}
                resolveUsers={resolveEmailSettingsRecipients}
                selectedUserIds={config.account_quota_email_recipient_user_ids}
                onChange={(account_quota_email_recipient_user_ids) =>
                  patchConfig({ account_quota_email_recipient_user_ids })
                }
              />
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Only active administrators and root users with email are available.'
                )}
              </p>
            </div>
          </div>
        </AlertPanel>

        <AlertPanel
          icon={ShieldAlert}
          title={t('Channel anomaly alert')}
          description={t(
            'Send email only after an anomaly automatically disables a channel.'
          )}
          checked={config.channel_anomaly_email_enabled}
          onCheckedChange={(checked) =>
            patchConfig({ channel_anomaly_email_enabled: checked })
          }
        >
          <div className='space-y-1.5'>
            <Label htmlFor='channel-anomaly-recipients'>
              {t('Recipients')}
            </Label>
            <EmailCampaignUserPicker
              id='channel-anomaly-recipients'
              labelKey='Alert recipients'
              queryKeyPrefix='channel-anomaly-email'
              searchUsers={searchEmailSettingsRecipients}
              resolveUsers={resolveEmailSettingsRecipients}
              selectedUserIds={config.channel_anomaly_email_recipient_user_ids}
              onChange={(channel_anomaly_email_recipient_user_ids) =>
                patchConfig({ channel_anomaly_email_recipient_user_ids })
              }
            />
            <p className='text-muted-foreground text-xs'>
              {t('Manual channel shutdowns never send this alert.')}
            </p>
          </div>
        </AlertPanel>
      </div>
    </div>
  )
}
