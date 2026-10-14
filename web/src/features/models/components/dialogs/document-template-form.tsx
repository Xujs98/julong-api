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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { lazy, Suspense, useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { LoadingState } from '@/components/loading-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { getServerErrorMessage } from '@/lib/server-error-message'

import {
  invalidateDocumentTemplateData,
  updateDocumentTemplate,
  uploadDocumentTemplate,
} from '../../document-templates-api'
import {
  documentTemplateFormSchema,
  documentUploadError,
  MAX_DOCUMENT_CONTENT_BYTES,
} from '../../lib/document-template-form'
import type { DocumentTemplateDialogProps } from './document-template-dialog'

const Markdown = lazy(() =>
  import('@/components/ui/markdown').then((module) => ({
    default: module.Markdown,
  }))
)

export function DocumentTemplateForm(
  props: Omit<DocumentTemplateDialogProps, 'open'> & { title: string }
) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const formId = useId()
  const [file, setFile] = useState<File | null>(null)
  const [fileError, setFileError] = useState('')
  const [section, setSection] = useState('source')
  const [confirmClose, setConfirmClose] = useState(false)
  const form = useForm({
    resolver: zodResolver(documentTemplateFormSchema),
    defaultValues: {
      name: props.document?.name ?? '',
      content: props.document?.content ?? '',
    },
  })
  const save = useMutation({
    mutationFn: (values: { name: string; content: string }) => {
      if (file) {
        return uploadDocumentTemplate(file, values.name, props.document?.id)
      }
      if (!props.document) throw new Error(t('Select a document to upload'))
      const patch = props.mode === 'rename' ? { name: values.name } : values
      return updateDocumentTemplate(props.document.id, patch)
    },
    onSuccess: async () => {
      await invalidateDocumentTemplateData(client)
      toast.success(t('Document template saved'))
      props.onOpenChange(false)
    },
    onError: (error) =>
      form.setError('root', { message: getServerErrorMessage(error) }),
  })
  const isDirty = form.formState.isDirty
  const close = (open: boolean) => {
    if (open || save.isPending) return
    if (isDirty || file) {
      setConfirmClose(true)
      return
    }
    props.onOpenChange(false)
  }
  const submit = form.handleSubmit((values) => {
    form.clearErrors('root')
    if (props.mode === 'upload' && !file) {
      setFileError(t('Select a document to upload'))
      return
    }
    if (props.mode === 'edit' && !file) {
      if (!values.content.trim()) {
        form.setError('content', {
          message: t('Document template content cannot be empty'),
        })
        return
      }
      if (
        new TextEncoder().encode(values.content).length >
        MAX_DOCUMENT_CONTENT_BYTES
      ) {
        form.setError('content', {
          message: t('Document template content exceeds 2 MB'),
        })
        return
      }
    }
    save.mutate(values)
  })
  return (
    <>
      <Dialog
        open
        onOpenChange={close}
        title={props.title}
        description={t(
          'Choose a template in a model’s endpoint settings to publish it as that model’s API example.'
        )}
        contentClassName={
          props.mode === 'edit' ? 'sm:max-w-4xl' : 'sm:max-w-xl'
        }
        footer={
          <>
            <Button
              variant='outline'
              disabled={save.isPending}
              onClick={() => close(false)}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' form={formId} disabled={save.isPending}>
              {save.isPending ? t('Saving...') : t('Save')}
            </Button>
          </>
        }
      >
        <Form {...form}>
          <form id={formId} className='space-y-4' onSubmit={submit}>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('Document name')}</FormLabel>
                  <FormControl>
                    <Input
                      maxLength={128}
                      {...field}
                      disabled={save.isPending}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {props.mode !== 'rename' && (
              <div className='space-y-2'>
                <Label htmlFor={`${formId}-file`}>
                  {props.mode === 'edit'
                    ? t('Replace document file (optional)')
                    : t('Document file')}
                </Label>
                <Input
                  id={`${formId}-file`}
                  type='file'
                  accept='.md,.markdown,.docx,text/markdown,application/vnd.openxmlformats-officedocument.wordprocessingml.document'
                  disabled={save.isPending}
                  aria-invalid={Boolean(fileError)}
                  aria-describedby={`${formId}-file-help`}
                  onChange={(event) => {
                    const selected = event.target.files?.[0]
                    event.target.value = ''
                    if (!selected) return
                    const error = documentUploadError(selected)
                    if (error) {
                      setFileError(t(error))
                      setFile(null)
                      return
                    }
                    setFileError('')
                    setFile(selected)
                    if (!form.getValues('name').trim()) {
                      form.setValue(
                        'name',
                        selected.name.replace(/\.(md|markdown|docx)$/i, ''),
                        { shouldDirty: true }
                      )
                    }
                  }}
                />
                <p
                  id={`${formId}-file-help`}
                  className='text-muted-foreground text-xs'
                >
                  {t(
                    'Markdown (.md, .markdown) or Word (.docx), up to 10 MB. Legacy .doc files must be saved as .docx. Word text is converted to Markdown; embedded images and objects are not imported.'
                  )}
                </p>
                {file && (
                  <div className='flex min-w-0 items-center gap-2 text-sm'>
                    <span className='truncate' title={file.name}>
                      {file.name}
                    </span>
                    <Button
                      type='button'
                      size='sm'
                      variant='ghost'
                      disabled={save.isPending}
                      onClick={() => setFile(null)}
                    >
                      {t('Clear file')}
                    </Button>
                  </div>
                )}
                {fileError && (
                  <p role='alert' className='text-destructive text-sm'>
                    {fileError}
                  </p>
                )}
              </div>
            )}
            {props.mode === 'edit' && !file && (
              <div className='space-y-3'>
                <Tabs value={section} onValueChange={setSection}>
                  <TabsList>
                    <TabsTrigger value='source'>
                      {t('Markdown source')}
                    </TabsTrigger>
                    <TabsTrigger value='preview'>{t('Preview')}</TabsTrigger>
                  </TabsList>
                </Tabs>
                {section === 'source' ? (
                  <FormField
                    control={form.control}
                    name='content'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Markdown content')}</FormLabel>
                        <FormControl>
                          <Textarea
                            className='min-h-72 resize-y font-mono text-xs'
                            {...field}
                            disabled={save.isPending}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ) : (
                  <div className='max-h-[55vh] min-h-40 overflow-auto rounded-lg border p-4'>
                    <Suspense fallback={<LoadingState />}>
                      <Markdown>{form.getValues('content')}</Markdown>
                    </Suspense>
                  </div>
                )}
              </div>
            )}
            {props.mode === 'edit' && file && (
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Saving will replace the template content with this file. Linked models will use the updated document.'
                )}
              </p>
            )}
            {form.formState.errors.root && (
              <Alert variant='destructive'>
                <AlertDescription>
                  {form.formState.errors.root.message}
                </AlertDescription>
              </Alert>
            )}
          </form>
        </Form>
      </Dialog>
      <ConfirmDialog
        open={confirmClose}
        onOpenChange={setConfirmClose}
        title={t('Discard unsaved changes?')}
        desc={t('Your document changes will be lost.')}
        confirmText={t('Discard')}
        handleConfirm={() => {
          setConfirmClose(false)
          props.onOpenChange(false)
        }}
      />
    </>
  )
}
