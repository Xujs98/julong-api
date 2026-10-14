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
import { ImagePlus, Loader2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
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
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'

import {
  invoiceApplicationSchema,
  parseInvoiceInformation,
  validatePaymentProofs,
  type InvoiceApplicationForm,
} from '../lib/invoice-application'
import type { InvoiceApplicationRequest } from '../types'
import { InvoicePaymentProofFiles } from './invoice-payment-proofs'

export function ApplyInvoiceDialog(props: {
  open: boolean
  email: string
  requestIds: string[]
  applying: boolean
  onOpenChange: (open: boolean) => void
  onApply: (request: InvoiceApplicationRequest) => Promise<unknown>
}) {
  const { t } = useTranslation()
  const [pasted, setPasted] = useState('')
  const uploadRef = useRef<HTMLInputElement>(null)
  const form = useForm<InvoiceApplicationForm>({
    resolver: zodResolver(invoiceApplicationSchema),
    defaultValues: {
      invoice_type: 'ordinary',
      company_name: '',
      tax_number: '',
      email: props.email,
      notes: '',
      payment_proofs: [],
    },
  })

  useEffect(() => {
    if (props.open && !form.getFieldState('email').isDirty) {
      form.setValue('email', props.email)
    }
  }, [form, props.email, props.open])

  async function submit(values: InvoiceApplicationForm) {
    if (props.requestIds.length === 0) return
    try {
      await props.onApply({ ...values, request_ids: props.requestIds })
      form.reset({
        invoice_type: 'ordinary',
        company_name: '',
        tax_number: '',
        email: props.email,
        notes: '',
        payment_proofs: [],
      })
      setPasted('')
      props.onOpenChange(false)
    } catch (error) {
      handleServerError(error, t('Failed to submit invoice application'))
    }
  }

  function recognizeInformation() {
    const recognized = parseInvoiceInformation(pasted)
    if (
      !recognized.company_name &&
      !recognized.tax_number &&
      !recognized.notes
    ) {
      toast.error(t('No invoice information was recognized'))
      return
    }
    if (recognized.company_name) {
      form.setValue('company_name', recognized.company_name, {
        shouldDirty: true,
        shouldValidate: true,
      })
    }
    if (recognized.tax_number) {
      form.setValue('tax_number', recognized.tax_number, {
        shouldDirty: true,
        shouldValidate: true,
      })
    }
    if (recognized.notes) {
      const current = form.getValues('notes').trim()
      form.setValue(
        'notes',
        current.includes(recognized.notes)
          ? current
          : [current, recognized.notes].filter(Boolean).join('\n'),
        { shouldDirty: true, shouldValidate: true }
      )
    }
    toast.success(t('Invoice information filled in'))
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (!props.applying) props.onOpenChange(open)
      }}
      title={t('Apply for invoice')}
      description={t(
        'Enter your invoicing details and upload payment screenshots for the selected credits.'
      )}
      contentClassName='sm:max-w-3xl'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            disabled={props.applying}
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='submit'
            form='invoice-application-form'
            disabled={props.applying || props.requestIds.length === 0}
          >
            {props.applying && (
              <Loader2 aria-hidden='true' className='animate-spin' />
            )}
            {t('Submit invoice application')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id='invoice-application-form'
          noValidate
          onSubmit={form.handleSubmit(submit)}
        >
          <FieldGroup>
            <FormField
              control={form.control}
              name='invoice_type'
              render={({ field }) => (
                <FormItem>
                  <Field data-disabled={props.applying}>
                    <FormLabel>{t('Invoice type')}</FormLabel>
                    <FormControl>
                      <RadioGroup
                        value={field.value}
                        onValueChange={field.onChange}
                        disabled={props.applying}
                        className='flex flex-wrap gap-5'
                        aria-label={t('Invoice type')}
                      >
                        <Field orientation='horizontal'>
                          <RadioGroupItem
                            id='invoice-type-ordinary'
                            value='ordinary'
                          />
                          <FieldLabel htmlFor='invoice-type-ordinary'>
                            {t('VAT ordinary invoice')}
                          </FieldLabel>
                        </Field>
                        <Field orientation='horizontal'>
                          <RadioGroupItem
                            id='invoice-type-special'
                            value='special'
                          />
                          <FieldLabel htmlFor='invoice-type-special'>
                            {t('VAT special invoice')}
                          </FieldLabel>
                        </Field>
                      </RadioGroup>
                    </FormControl>
                    <FormMessage />
                  </Field>
                </FormItem>
              )}
            />
            <Field data-disabled={props.applying}>
              <FieldLabel htmlFor='invoice-pasted-information'>
                {t('Paste invoice information')}
              </FieldLabel>
              <Textarea
                id='invoice-pasted-information'
                value={pasted}
                maxLength={5000}
                rows={4}
                disabled={props.applying}
                placeholder={t(
                  'Paste company name, tax number, address, phone, and bank account details'
                )}
                onChange={(event) => setPasted(event.target.value)}
              />
              <div className='flex flex-wrap items-center justify-between gap-2'>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Recognition happens locally. Address, phone, and bank details are kept in Notes.'
                  )}
                </p>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={props.applying || !pasted.trim()}
                  onClick={recognizeInformation}
                >
                  {t('Recognize and fill')}
                </Button>
              </div>
            </Field>
            <FormField
              control={form.control}
              name='company_name'
              render={({ field, fieldState }) => (
                <FormItem>
                  <Field
                    data-invalid={Boolean(fieldState.error)}
                    data-disabled={props.applying}
                  >
                    <FormLabel>{t('Company name')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        maxLength={200}
                        disabled={props.applying}
                        placeholder={t('Full company name on the invoice')}
                        autoComplete='organization'
                      />
                    </FormControl>
                    <FormMessage />
                  </Field>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='tax_number'
              render={({ field, fieldState }) => (
                <FormItem>
                  <Field
                    data-invalid={Boolean(fieldState.error)}
                    data-disabled={props.applying}
                  >
                    <FormLabel>{t('Tax number')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        maxLength={50}
                        disabled={props.applying}
                        placeholder={t('Taxpayer identification number')}
                      />
                    </FormControl>
                    <FormMessage />
                  </Field>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='email'
              render={({ field, fieldState }) => (
                <FormItem>
                  <Field
                    data-invalid={Boolean(fieldState.error)}
                    data-disabled={props.applying}
                  >
                    <FormLabel>{t('Receiving email')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type='email'
                        maxLength={254}
                        disabled={props.applying}
                        autoComplete='email'
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Your bound email is filled in automatically. Changing it here only affects this invoice.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </Field>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='notes'
              render={({ field, fieldState }) => (
                <FormItem>
                  <Field
                    data-invalid={Boolean(fieldState.error)}
                    data-disabled={props.applying}
                  >
                    <FormLabel>{t('Notes')}</FormLabel>
                    <FormControl>
                      <Textarea
                        {...field}
                        maxLength={1000}
                        rows={3}
                        disabled={props.applying}
                      />
                    </FormControl>
                    <FormMessage />
                  </Field>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='payment_proofs'
              render={({ field, fieldState }) => (
                <FormItem>
                  <Field
                    data-invalid={Boolean(fieldState.error)}
                    data-disabled={props.applying}
                  >
                    <FormLabel>{t('Payment screenshots')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Upload 1 to 3 payment screenshots (PNG, JPG, or WebP), up to 5 MiB each, for reconciliation and tax records.'
                      )}
                    </FormDescription>
                    <FormControl>
                      <Input
                        ref={uploadRef}
                        type='file'
                        className='sr-only'
                        accept='image/png,image/jpeg,image/webp'
                        multiple
                        disabled={props.applying}
                        onChange={(event) => {
                          const added = [...(event.target.files ?? [])]
                          event.target.value = ''
                          if (added.length === 0) return
                          const files = [...field.value]
                          for (const file of added) {
                            if (
                              !files.some(
                                (existing) =>
                                  existing.name === file.name &&
                                  existing.size === file.size &&
                                  existing.lastModified === file.lastModified
                              )
                            ) {
                              files.push(file)
                            }
                          }
                          const error = validatePaymentProofs(files)
                          if (error) {
                            form.setError('payment_proofs', { message: error })
                            return
                          }
                          form.clearErrors('payment_proofs')
                          field.onChange(files)
                        }}
                      />
                    </FormControl>
                    <InvoicePaymentProofFiles
                      files={field.value}
                      disabled={props.applying}
                      onRemove={(index) => {
                        field.onChange(
                          field.value.filter(
                            (_, fileIndex) => fileIndex !== index
                          )
                        )
                        form.clearErrors('payment_proofs')
                      }}
                    />
                    <Button
                      type='button'
                      variant='outline'
                      className='w-fit'
                      disabled={props.applying || field.value.length >= 3}
                      onClick={() => uploadRef.current?.click()}
                    >
                      <ImagePlus aria-hidden='true' />
                      {t('Add screenshot')}
                    </Button>
                    <FormMessage />
                  </Field>
                </FormItem>
              )}
            />
          </FieldGroup>
        </form>
      </Form>
    </Dialog>
  )
}
