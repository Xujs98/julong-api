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
import {
  CalendarClock,
  ChartNoAxesCombined,
  CircleGauge,
  RefreshCw,
  Save,
  Send,
  ShieldAlert,
  TriangleAlert,
  WalletCards,
  type LucideIcon,
} from 'lucide-react'
import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { DashboardReportScheduleEditor } from './dashboard-report-schedule-editor'
import { EmailCampaignUserPicker } from './email-campaign-user-picker'
import {
  getEmailSettingsConfig,
  resolveEmailSettingsRecipients,
  searchEmailSettingsRecipients,
  sendChannelAnomalyTestEmail,
  sendDashboardReportTestEmail,
  sendRiskUserTestEmail,
  updateEmailSettingsConfig,
  type EmailSettingsConfig,
  type RiskUserEmailLevel,
} from './email-templates-api'

const RISK_USER_EMAIL_LEVELS: Array<{
  value: RiskUserEmailLevel
  labelKey: string
}> = [
  { value: 'medium', labelKey: 'Medium risk' },
  { value: 'high', labelKey: 'High risk' },
]

type AlertPanelProps = {
  icon: LucideIcon
  title: string
  description: string
  toggleLabel: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  children?: ReactNode
}

function AlertPanel(props: AlertPanelProps) {
  const Icon = props.icon
  return (
    <div className='bg-card overflow-hidden rounded-lg border'>
      <div className='flex items-start gap-3 border-b px-4 py-4 sm:px-5'>
        <span className='bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md'>
          <Icon className='size-4.5' aria-hidden='true' />
        </span>
        <span className='min-w-0'>
          <span className='block text-base font-semibold'>{props.title}</span>
          <span className='text-muted-foreground mt-1 block text-sm leading-5'>
            {props.description}
          </span>
        </span>
      </div>
      <div className='px-4 py-4 sm:px-5'>
        <div className='flex min-h-9 items-center justify-between gap-4'>
          <span className='text-sm font-medium'>{props.toggleLabel}</span>
          <Switch
            checked={props.checked}
            aria-label={props.toggleLabel}
            onCheckedChange={props.onCheckedChange}
          />
        </div>
        {props.children && (
          <div className='mt-4 grid gap-4 border-t pt-4'>{props.children}</div>
        )}
      </div>
    </div>
  )
}

export function EmailAlertSettings() {
  const { t } = useTranslation()
  const [config, setConfig] = useState<EmailSettingsConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [testingChannelAnomaly, setTestingChannelAnomaly] = useState(false)
  const [testingDashboardReport, setTestingDashboardReport] = useState(false)
  const [testingRiskUser, setTestingRiskUser] = useState(false)

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setLoadError(null)
    try {
      const response = await getEmailSettingsConfig()
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to load email settings'))
      }
      setConfig(response.data)
    } catch (error) {
      setConfig(null)
      setLoadError(
        error instanceof Error
          ? error.message
          : t('Failed to load email settings')
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadConfig()
  }, [loadConfig])

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
    if (
      config.risk_user_email_enabled &&
      config.risk_user_email_levels.length === 0
    ) {
      toast.error(t('Select at least one risk level'))
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

  const testChannelAnomaly = async () => {
    if (!config) return
    setTestingChannelAnomaly(true)
    try {
      const response = await sendChannelAnomalyTestEmail(
        config.channel_anomaly_email_recipient_user_ids
      )
      if (!response.success || !response.data) throw new Error(response.message)
      toast.success(
        t('Channel anomaly test email sent to {{count}} recipient(s)', {
          count: response.data.recipient_count,
        })
      )
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to send channel anomaly test email')
      )
    } finally {
      setTestingChannelAnomaly(false)
    }
  }

  const testDashboardReport = async () => {
    if (!config) return
    setTestingDashboardReport(true)
    try {
      const response = await sendDashboardReportTestEmail(
        config.dashboard_report_email_recipient_user_ids
      )
      if (!response.success || !response.data) throw new Error(response.message)
      toast.success(
        t('Data dashboard test report sent to {{count}} recipient(s)', {
          count: response.data.recipient_count,
        })
      )
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to send data dashboard test report')
      )
    } finally {
      setTestingDashboardReport(false)
    }
  }

  const toggleRiskUserLevel = (level: RiskUserEmailLevel, checked: boolean) => {
    if (!config) return
    const selected = new Set(config.risk_user_email_levels)
    if (checked) selected.add(level)
    else selected.delete(level)
    patchConfig({
      risk_user_email_levels: RISK_USER_EMAIL_LEVELS.map(
        (option) => option.value
      ).filter((value) => selected.has(value)),
    })
  }

  const testRiskUser = async () => {
    if (!config) return
    if (config.risk_user_email_levels.length === 0) {
      toast.error(t('Select at least one risk level'))
      return
    }
    setTestingRiskUser(true)
    try {
      const response = await sendRiskUserTestEmail(
        config.risk_user_email_recipient_user_ids,
        config.risk_user_email_levels
      )
      if (!response.success || !response.data) throw new Error(response.message)
      toast.success(
        t(
          'Risk alert test email sent to {{count}} recipient(s), covering {{users}} user(s)',
          {
            count: response.data.recipient_count,
            users: response.data.risk_user_count,
          }
        )
      )
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to send risk alert test email')
      )
    } finally {
      setTestingRiskUser(false)
    }
  }

  if (loadError) {
    return (
      <div className='flex min-h-32 flex-col items-center justify-center gap-3 rounded-lg border p-5 text-center'>
        <p className='text-destructive text-sm'>{loadError}</p>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => void loadConfig()}
        >
          <RefreshCw className='size-4' />
          {t('Retry')}
        </Button>
      </div>
    )
  }

  if (loading || !config) {
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
              'Configure automatic billing, subscription, channel, and risk alerts.'
            )}
          </p>
        </div>
        <Button type='button' disabled={saving} onClick={save}>
          <Save className='size-4' />
          {t('Save settings')}
        </Button>
      </div>

      <div className='grid gap-5'>
        <AlertPanel
          icon={CalendarClock}
          title={t('Subscription expiry reminders')}
          description={t(
            'When enabled, reminders are sent 7, 3, and 1 days before expiry.'
          )}
          toggleLabel={t('Enable subscription expiry reminders')}
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
          toggleLabel={t('Enable low balance reminder')}
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
          toggleLabel={t('Enable account quota alert')}
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
          icon={ChartNoAxesCombined}
          title={t('Data dashboard report')}
          description={t(
            'Send a completed daily, weekly, or monthly data dashboard report to selected administrators.'
          )}
          toggleLabel={t('Enable scheduled data dashboard reports')}
          checked={config.dashboard_report_email_enabled}
          onCheckedChange={(checked) =>
            patchConfig({ dashboard_report_email_enabled: checked })
          }
        >
          <DashboardReportScheduleEditor
            disabled={!config.dashboard_report_email_enabled}
            schedules={config.dashboard_report_email_schedules}
            onChange={(dashboard_report_email_schedules) =>
              patchConfig({ dashboard_report_email_schedules })
            }
          />
          <div className='space-y-1.5'>
            <Label htmlFor='dashboard-report-recipients'>
              {t('Recipients')}
            </Label>
            <EmailCampaignUserPicker
              id='dashboard-report-recipients'
              labelKey='Report recipients'
              queryKeyPrefix='dashboard-report-email'
              searchUsers={searchEmailSettingsRecipients}
              resolveUsers={resolveEmailSettingsRecipients}
              selectedUserIds={config.dashboard_report_email_recipient_user_ids}
              onChange={(dashboard_report_email_recipient_user_ids) =>
                patchConfig({ dashboard_report_email_recipient_user_ids })
              }
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'Scheduled reports use completed periods. The test email uses current real dashboard data.'
              )}
            </p>
            <div className='flex justify-end'>
              <Button
                type='button'
                variant='outline'
                disabled={testingDashboardReport}
                onClick={testDashboardReport}
              >
                <Send className='size-4' />
                {testingDashboardReport
                  ? t('Sending...')
                  : t('Send real data test')}
              </Button>
            </div>
          </div>
        </AlertPanel>

        <AlertPanel
          icon={TriangleAlert}
          title={t('Risk user alert')}
          description={t(
            'Send email when risk detection identifies users at the selected levels.'
          )}
          toggleLabel={t('Enable risk user alert')}
          checked={config.risk_user_email_enabled}
          onCheckedChange={(checked) =>
            patchConfig({ risk_user_email_enabled: checked })
          }
        >
          <div className='space-y-2'>
            <Label>{t('Risk levels')}</Label>
            <div className='grid gap-2 sm:grid-cols-2'>
              {RISK_USER_EMAIL_LEVELS.map((option) => {
                const checked = config.risk_user_email_levels.includes(
                  option.value
                )
                return (
                  <label
                    key={option.value}
                    className='hover:bg-muted/50 flex min-h-10 cursor-pointer items-center gap-3 rounded-md border px-3 py-2 text-sm'
                  >
                    <Checkbox
                      checked={checked}
                      disabled={!config.risk_user_email_enabled}
                      onCheckedChange={(nextChecked) =>
                        toggleRiskUserLevel(option.value, nextChecked)
                      }
                    />
                    <span>{t(option.labelKey)}</span>
                  </label>
                )
              })}
            </div>
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='risk-user-recipients'>{t('Recipients')}</Label>
            <EmailCampaignUserPicker
              id='risk-user-recipients'
              labelKey='Alert recipients'
              queryKeyPrefix='risk-user-email'
              searchUsers={searchEmailSettingsRecipients}
              resolveUsers={resolveEmailSettingsRecipients}
              selectedUserIds={config.risk_user_email_recipient_user_ids}
              onChange={(risk_user_email_recipient_user_ids) =>
                patchConfig({ risk_user_email_recipient_user_ids })
              }
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'Only users currently marked at the selected risk levels are included. The test email uses current real risk data.'
              )}
            </p>
            <div className='flex justify-end'>
              <Button
                type='button'
                variant='outline'
                disabled={testingRiskUser}
                onClick={testRiskUser}
              >
                <Send className='size-4' />
                {testingRiskUser ? t('Sending...') : t('Send real data test')}
              </Button>
            </div>
          </div>
        </AlertPanel>

        <AlertPanel
          icon={ShieldAlert}
          title={t('Channel anomaly alert')}
          description={t(
            'Send email only after an anomaly automatically disables a channel.'
          )}
          toggleLabel={t('Enable channel anomaly alert')}
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
            <div className='flex justify-end'>
              <Button
                type='button'
                variant='outline'
                disabled={testingChannelAnomaly}
                onClick={testChannelAnomaly}
              >
                <Send className='size-4' />
                {testingChannelAnomaly ? t('Sending...') : t('Send test email')}
              </Button>
            </div>
          </div>
        </AlertPanel>
      </div>
    </div>
  )
}
