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
import { Loader2, Save } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { getCurrencyLabel, getCurrencyDisplay } from '@/lib/currency'
import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'

import {
  formatLedgerDateTimeInput,
  parseLedgerDateTimeInput,
} from '../lib/date-time'
import type { LedgerEntry, LedgerMutation, LedgerPlatform } from '../types'

const LEDGER_PLATFORMS: LedgerPlatform[] = [
  'Anthropic',
  'OpenAI',
  'Gemini',
  'Antigravity',
  'Grok',
]
const STANDARD_TYPES = ['plus', 'pro', 'k12'] as const
const MAX_LEDGER_QUOTA = 2_147_483_647

interface LedgerEntryDialogProps {
  open: boolean
  entry?: LedgerEntry
  submitting: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: LedgerMutation) => Promise<void>
}

interface LedgerFormValues {
  platform: LedgerPlatform
  account: string
  email: string
  typeMode: 'plus' | 'pro' | 'k12' | 'custom'
  customType: string
  quotaAmount: number
  costPrice: number
  quantity: number
  occurredDateTime: string
}

function getTypeDefaults(
  type?: string
): Pick<LedgerFormValues, 'typeMode' | 'customType'> {
  const normalized = type?.toLowerCase()
  if (STANDARD_TYPES.includes(normalized as (typeof STANDARD_TYPES)[number])) {
    return {
      typeMode: normalized as LedgerFormValues['typeMode'],
      customType: '',
    }
  }
  return { typeMode: 'custom', customType: type ?? '' }
}

function getFormDefaults(entry?: LedgerEntry): LedgerFormValues {
  const type = getTypeDefaults(entry?.type)
  return {
    platform: entry?.platform ?? 'OpenAI',
    account: entry?.account ?? '',
    email: entry?.email ?? '',
    typeMode: type.typeMode,
    customType: type.customType,
    quotaAmount: entry ? quotaUnitsToDollars(entry.quota) : 0,
    costPrice: entry ? Number(entry.cost_price) : 0,
    quantity: entry?.quantity ?? 1,
    occurredDateTime: formatLedgerDateTimeInput(entry?.occurred_at),
  }
}

export function LedgerEntryDialog(props: LedgerEntryDialogProps) {
  const { t } = useTranslation()
  const schema = useMemo(
    () =>
      z
        .object({
          platform: z.enum(LEDGER_PLATFORMS),
          account: z
            .string()
            .trim()
            .min(1, t('Account information is required'))
            .max(255, t('Account information is too long')),
          email: z
            .string()
            .trim()
            .refine(
              (value) => value === '' || z.email().safeParse(value).success,
              t('Invalid email address')
            ),
          typeMode: z.enum(['plus', 'pro', 'k12', 'custom']),
          customType: z.string().trim().max(64, t('Type is too long')),
          quotaAmount: z.number().positive(t('Quota must be greater than 0')),
          costPrice: z
            .number()
            .positive(t('Cost price must be greater than 0'))
            .max(1_000_000_000, t('Cost price is too large')),
          quantity: z
            .number()
            .int(t('Quantity must be an integer'))
            .min(1, t('Quantity must be at least 1'))
            .max(10000, t('Quantity must not exceed 10000')),
          occurredDateTime: z
            .string()
            .refine(
              (value) => parseLedgerDateTimeInput(value) != null,
              t('Ledger date and time is required')
            ),
        })
        .refine(
          (values) => values.typeMode !== 'custom' || values.customType !== '',
          { path: ['customType'], message: t('Custom type is required') }
        ),
    [t]
  )
  const form = useForm<LedgerFormValues>({
    resolver: zodResolver(schema),
    defaultValues: getFormDefaults(props.entry),
  })
  const typeMode = form.watch('typeMode')
  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()

  useEffect(() => {
    if (props.open) form.reset(getFormDefaults(props.entry))
  }, [form, props.entry, props.open])

  const submit = async (values: LedgerFormValues) => {
    const quota = parseQuotaFromDollars(values.quotaAmount)
    if (quota <= 0) {
      form.setError('quotaAmount', {
        message: t('Quota must be greater than 0'),
      })
      return
    }
    if (quota > MAX_LEDGER_QUOTA) {
      form.setError('quotaAmount', { message: t('Quota is too large') })
      return
    }
    const type =
      values.typeMode === 'custom' ? values.customType.trim() : values.typeMode
    const occurredAt = parseLedgerDateTimeInput(values.occurredDateTime)
    if (occurredAt == null) {
      form.setError('occurredDateTime', {
        message: t('Ledger date and time is required'),
      })
      return
    }
    await props.onSubmit({
      platform: values.platform,
      account: values.account.trim(),
      email: values.email.trim(),
      type,
      quota,
      cost_price: String(values.costPrice),
      quantity: values.quantity,
      occurred_at: occurredAt,
    })
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[min(760px,calc(100vh-2rem))] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {props.entry ? t('Edit ledger entry') : t('Add ledger entry')}
          </DialogTitle>
          <DialogDescription>
            {t('Record an upstream account quota and its operating cost.')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            className='space-y-4'
            onSubmit={(event) => void form.handleSubmit(submit)(event)}
          >
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='platform'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Platform')}</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {LEDGER_PLATFORMS.map((platform) => (
                            <SelectItem key={platform} value={platform}>
                              {platform}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='occurredDateTime'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Ledger date and time')}</FormLabel>
                    <FormControl>
                      <Input type='datetime-local' step='1' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='account'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Account information')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      placeholder={t('Account name or identifier')}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='email'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Email (optional)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='email'
                      {...field}
                      placeholder='name@example.com'
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='typeMode'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Type')}</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='plus'>Plus</SelectItem>
                          <SelectItem value='pro'>Pro</SelectItem>
                          <SelectItem value='k12'>K12</SelectItem>
                          <SelectItem value='custom'>{t('Custom')}</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {typeMode === 'custom' && (
                <FormField
                  control={form.control}
                  name='customType'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Custom type')}</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </div>
            <div className='grid gap-4 sm:grid-cols-3'>
              <FormField
                control={form.control}
                name='quotaAmount'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Quota ({{currency}})', { currency: currencyLabel })}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min='0'
                        max={quotaUnitsToDollars(MAX_LEDGER_QUOTA)}
                        step={currencyMeta.kind === 'tokens' ? '1' : '0.000001'}
                        value={field.value}
                        onBlur={field.onBlur}
                        name={field.name}
                        ref={field.ref}
                        onChange={(event) =>
                          field.onChange(event.target.valueAsNumber)
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='costPrice'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Cost price (USD)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min='0'
                        step='0.000001'
                        value={field.value}
                        onBlur={field.onBlur}
                        name={field.name}
                        ref={field.ref}
                        onChange={(event) =>
                          field.onChange(event.target.valueAsNumber)
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='quantity'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Quantity')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min='1'
                        max='10000'
                        step='1'
                        value={field.value}
                        onBlur={field.onBlur}
                        name={field.name}
                        ref={field.ref}
                        onChange={(event) =>
                          field.onChange(event.target.valueAsNumber)
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => props.onOpenChange(false)}
                disabled={props.submitting}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={props.submitting}>
                {props.submitting ? (
                  <Loader2 data-icon='inline-start' className='animate-spin' />
                ) : (
                  <Save data-icon='inline-start' />
                )}
                {t('Save')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
