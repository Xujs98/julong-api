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
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

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
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const schema = z.object({
  enabled: z.boolean(),
  minimumAmount: z.coerce.number().positive().max(1_000_000_000),
  unit: z.string().trim().min(1).max(16),
  processingDays: z.string().trim().min(1).max(32),
  rejectionFreezeHours: z.coerce.number().int().min(0).max(8760),
})

type Values = z.infer<typeof schema>

export function InvoiceSettingsSection(props: { defaultValues: Values }) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<Values>({
    resolver: zodResolver(schema) as Resolver<Values>,
    defaultValues: props.defaultValues,
  })
  const { isDirty, isSubmitting } = form.formState

  async function onSubmit(values: Values) {
    const updates = [
      ['invoice_setting.enabled', values.enabled, props.defaultValues.enabled],
      [
        'invoice_setting.minimum_amount',
        values.minimumAmount,
        props.defaultValues.minimumAmount,
      ],
      ['invoice_setting.unit', values.unit, props.defaultValues.unit],
      [
        'invoice_setting.processing_days',
        values.processingDays,
        props.defaultValues.processingDays,
      ],
      [
        'invoice_setting.rejection_freeze_hours',
        values.rejectionFreezeHours,
        props.defaultValues.rejectionFreezeHours,
      ],
    ].filter(([, value, previous]) => value !== previous)

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }
    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key: String(key), value: String(value) })
    }
    form.reset(values)
  }

  return (
    <SettingsSection title={t('Self-service invoice settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending || isSubmitting}
            isSaveDisabled={!isDirty}
            saveLabel='Save invoice settings'
          />
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable self-service invoices')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When disabled, users can view the page but will see that the feature is in private testing.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={updateOption.isPending || isSubmitting}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <FormField
            control={form.control}
            name='minimumAmount'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Minimum invoice amount')}</FormLabel>
                <FormControl>
                  <Input type='number' min={0.01} step='0.01' {...field} />
                </FormControl>
                <FormDescription>
                  {t('Selected eligible credits must reach this amount.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='rejectionFreezeHours'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Rejection freeze time')}</FormLabel>
                <FormControl>
                  <Input type='number' min={0} max={8760} step={1} {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Rejected credits return to the eligible list immediately and can be selected again after this many hours. Use 0 to allow immediate reuse.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='unit'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Invoice amount unit')}</FormLabel>
                <FormControl>
                  <Input placeholder='USD' maxLength={16} {...field} />
                </FormControl>
                <FormDescription>
                  {t('Displayed after invoice amounts, such as USD or CNY.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='processingDays'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Estimated working days')}</FormLabel>
                <FormControl>
                  <Input placeholder='1-3' maxLength={32} {...field} />
                </FormControl>
                <FormDescription>
                  {t('Shown to users as the estimated invoice delivery time.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
