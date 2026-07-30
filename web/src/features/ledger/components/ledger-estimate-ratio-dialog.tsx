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

interface LedgerEstimateRatioDialogProps {
  open: boolean
  estimateRatio: number
  submitting: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (estimateRatio: number) => Promise<void>
}

interface EstimateRatioFormValues {
  estimateRatio: number
}

export function LedgerEstimateRatioDialog(
  props: LedgerEstimateRatioDialogProps
) {
  const { t } = useTranslation()
  const schema = useMemo(
    () =>
      z.object({
        estimateRatio: z
          .number()
          .positive(t('Estimate ratio must be greater than 0'))
          .max(1000, t('Estimate ratio must not exceed 1000')),
      }),
    [t]
  )
  const form = useForm<EstimateRatioFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { estimateRatio: props.estimateRatio },
  })

  useEffect(() => {
    if (props.open) {
      form.reset({ estimateRatio: props.estimateRatio })
    }
  }, [form, props.estimateRatio, props.open])

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{t('Configure estimate ratio')}</DialogTitle>
          <DialogDescription>
            {t(
              'Estimated consumption quota equals real quota divided by estimate ratio.'
            )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            className='space-y-4'
            onSubmit={(event) =>
              void form.handleSubmit(async (values) => {
                await props.onSubmit(values.estimateRatio)
              })(event)
            }
          >
            <FormField
              control={form.control}
              name='estimateRatio'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Estimate ratio')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min='0.000001'
                      max='1000'
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
