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
import { Minus, Plus } from 'lucide-react'
import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { cn } from '@/lib/utils'

import type { UserFormValues } from '../lib/user-form'
import {
  buildUserGroupRatioPreviews,
  getSignedUserGroupRatioAdjustment,
  type UserGroupRatioAdjustmentMode,
} from '../lib/user-ratio-adjustment'

interface UserGroupRatioAdjustmentFieldsProps {
  baseRatios: Record<string, number>
  loading: boolean
}

function formatRatio(ratio: number): string {
  return `${Number(ratio.toFixed(6))}x`
}

export function UserGroupRatioAdjustmentFields(
  props: UserGroupRatioAdjustmentFieldsProps
) {
  const { t } = useTranslation()
  const form = useFormContext<UserFormValues>()
  const enabled = form.watch('group_ratio_adjustment_enabled')
  const mode = form.watch('group_ratio_adjustment_mode')
  const value = form.watch('group_ratio_adjustment_value')
  const adjustment = getSignedUserGroupRatioAdjustment(enabled, mode, value)
  const previews = buildUserGroupRatioPreviews(
    props.baseRatios,
    enabled,
    mode,
    value
  )

  return (
    <FormField
      control={form.control}
      name='group_ratio_adjustment_enabled'
      render={({ field }) => (
        <FormItem className='space-y-4 rounded-md border p-3'>
          <div className='flex items-center justify-between gap-4'>
            <FormLabel>{t('Adjust billing ratio')}</FormLabel>
            <FormControl>
              <Switch
                checked={field.value}
                onCheckedChange={(checked) => {
                  form.clearErrors('group_ratio_adjustment_value')
                  field.onChange(checked)
                }}
              />
            </FormControl>
          </div>

          {field.value && (
            <>
              <div className='grid gap-3 sm:grid-cols-[auto_minmax(0,1fr)]'>
                <FormField
                  control={form.control}
                  name='group_ratio_adjustment_mode'
                  render={({ field: modeField }) => (
                    <FormItem>
                      <FormLabel>{t('Adjustment')}</FormLabel>
                      <ToggleGroup
                        value={[modeField.value]}
                        onValueChange={(nextValues) => {
                          const nextMode = nextValues.find(
                            (item) => item !== modeField.value
                          ) as UserGroupRatioAdjustmentMode | undefined
                          if (nextMode) {
                            form.clearErrors('group_ratio_adjustment_value')
                            modeField.onChange(nextMode)
                          }
                        }}
                        aria-label={t('Adjustment')}
                        variant='outline'
                        spacing={0}
                      >
                        <ToggleGroupItem value='increase'>
                          <Plus aria-hidden='true' />
                          {t('Increase')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='decrease'>
                          <Minus aria-hidden='true' />
                          {t('Decrease')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='group_ratio_adjustment_value'
                  render={({ field: valueField }) => (
                    <FormItem>
                      <FormLabel>{t('Adjustment value')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          min='0'
                          step='0.01'
                          value={valueField.value}
                          aria-invalid={Boolean(
                            form.formState.errors.group_ratio_adjustment_value
                          )}
                          onChange={(event) => {
                            form.clearErrors('group_ratio_adjustment_value')
                            const nextValue = Number.parseFloat(
                              event.target.value
                            )
                            valueField.onChange(
                              Number.isFinite(nextValue) ? nextValue : 0
                            )
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <div className='space-y-2 border-t pt-3'>
                <div className='text-sm font-medium'>
                  {t('Actual ratio preview')}
                </div>
                {props.loading && (
                  <div className='space-y-2' aria-busy='true'>
                    <div className='bg-muted h-8 animate-pulse rounded' />
                    <div className='bg-muted h-8 animate-pulse rounded' />
                  </div>
                )}
                {!props.loading && previews.length === 0 && (
                  <p className='text-muted-foreground text-xs'>
                    {t('No token groups selected for this user.')}
                  </p>
                )}
                {!props.loading && previews.length > 0 && (
                  <div className='divide-y rounded-md border'>
                    {previews.map((preview) => (
                      <div
                        key={preview.group}
                        className='grid grid-cols-1 gap-1 px-3 py-2 text-sm sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:gap-3'
                      >
                        <span className='truncate font-medium'>
                          {preview.group}
                        </span>
                        <span
                          className={cn(
                            'font-mono text-xs',
                            'break-all sm:text-right',
                            preview.invalid && 'text-destructive'
                          )}
                        >
                          {formatRatio(preview.baseRatio)}{' '}
                          {adjustment < 0 ? '-' : '+'}{' '}
                          {formatRatio(Math.abs(adjustment))} ={' '}
                          {formatRatio(preview.adjustedRatio)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </FormItem>
      )}
    />
  )
}
