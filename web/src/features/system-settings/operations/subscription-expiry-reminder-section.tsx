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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

type SubscriptionExpiryReminderSectionProps = {
  defaultEnabled: boolean
}

export function SubscriptionExpiryReminderSection({
  defaultEnabled,
}: SubscriptionExpiryReminderSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [enabled, setEnabled] = useState(defaultEnabled)

  useEffect(() => setEnabled(defaultEnabled), [defaultEnabled])

  const updateEnabled = async (checked: boolean) => {
    const previous = enabled
    setEnabled(checked)
    try {
      const response = await updateOption.mutateAsync({
        key: 'SubscriptionExpiryReminderEnabled',
        value: checked,
      })
      if (!response.success) setEnabled(previous)
    } catch {
      setEnabled(previous)
    }
  }

  return (
    <SettingsSection title={t('Subscription expiry reminders')}>
      <div className='overflow-hidden rounded-lg border'>
        <div className='border-b px-5 py-4'>
          <h3 className='text-sm font-semibold'>
            {t('Subscription expiry reminders')}
          </h3>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Control whether subscription expiry reminder emails are sent.')}
          </p>
        </div>
        <div className='px-5 py-3'>
          <SettingsSwitchField
            checked={enabled}
            disabled={updateOption.isPending}
            onCheckedChange={updateEnabled}
            label={t('Enable subscription expiry reminders')}
            description={t(
              'When enabled, reminders are sent 7, 3, and 1 days before expiry.'
            )}
          />
        </div>
      </div>
    </SettingsSection>
  )
}
