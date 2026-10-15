import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Save, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { handleServerError } from '@/lib/handle-server-error'

import {
  getSupportContacts,
  updateSupportContacts,
  type SupportContact,
  type SupportContactType,
} from '../api'
import { SettingsSection } from '../components/settings-section'
import { SupportQRCodeField } from './support-qrcode-field'

const emptyContact = (): SupportContact => ({
  id: crypto.randomUUID(),
  type: 'qq',
  label: '',
  value: '',
})

export function SupportContactsSection() {
  const { t } = useTranslation()
  const [draft, setContacts] = useState<SupportContact[] | null>(null)
  const [files, setFiles] = useState<Record<string, File>>({})
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ['support-contacts'],
    queryFn: getSupportContacts,
  })
  const initialized = !!query.data
  const contacts = draft ?? query.data?.data ?? []
  const loading = !initialized

  const updateContact = (index: number, patch: Partial<SupportContact>) => {
    setContacts((current) =>
      (current ?? contacts).map((contact, itemIndex) =>
        itemIndex === index ? { ...contact, ...patch } : contact
      )
    )
  }

  const saveMutation = useMutation({
    mutationFn: () => updateSupportContacts(contacts, files),
    onSuccess: (response) => {
      setContacts(response.data)
      setFiles({})
      queryClient.setQueryData(['support-contacts'], response)
      toast.success(t('Setting updated successfully'))
    },
    onError: (error) => handleServerError(error, t('Failed to update setting')),
  })
  const saving = saveMutation.isPending
  const handleSave = () => {
    if (contacts.some((contact) => !contact.value.trim())) {
      toast.error(t('Contact value is required'))
      return
    }
    saveMutation.mutate()
  }

  return (
    <SettingsSection title={t('Customer Support')}>
      <div className='space-y-4'>
        <div>
          <h3 className='text-sm font-medium'>{t('Support contacts')}</h3>
          <p className='text-muted-foreground mt-1 text-xs'>
            {t(
              'Configure support contacts and group QR codes shown in the header.'
            )}
          </p>
        </div>

        {query.isError && !initialized && (
          <ErrorState
            title={t('Failed to load support contacts')}
            onRetry={() => void query.refetch()}
          />
        )}
        {query.isPending && <LoadingState />}
        <fieldset disabled={saving || loading} className='min-w-0 space-y-3'>
          {contacts.map((contact, index) => (
            <div
              key={contact.id}
              className='grid gap-3 rounded-lg border p-3 sm:grid-cols-[140px_minmax(0,0.8fr)_minmax(0,1.2fr)_36px] sm:items-end'
            >
              <div className='space-y-1.5'>
                <Label htmlFor={`type-${contact.id}`}>
                  {t('Contact type')}
                </Label>
                <Select
                  value={contact.type}
                  onValueChange={(value) => {
                    if (!value) return
                    const group =
                      value === 'qq_group' || value === 'wechat_group'
                    updateContact(index, {
                      type: value as SupportContactType,
                      qr_code_url: group ? contact.qr_code_url : '',
                    })
                    if (!group) {
                      setFiles((current) => {
                        const next = { ...current }
                        delete next[contact.id]
                        return next
                      })
                    }
                  }}
                >
                  <SelectTrigger id={`type-${contact.id}`}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='qq'>QQ</SelectItem>
                    <SelectItem value='wechat'>{t('WeChat')}</SelectItem>
                    <SelectItem value='phone'>{t('Phone')}</SelectItem>
                    <SelectItem value='qq_group'>{t('QQ group')}</SelectItem>
                    <SelectItem value='wechat_group'>
                      {t('WeChat group')}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-1.5'>
                <Label htmlFor={`label-${contact.id}`}>
                  {t('Display name')}
                </Label>
                <Input
                  id={`label-${contact.id}`}
                  maxLength={64}
                  value={contact.label}
                  placeholder={t('Optional')}
                  onChange={(event) =>
                    updateContact(index, { label: event.target.value })
                  }
                />
              </div>
              <div className='space-y-1.5'>
                <Label htmlFor={`value-${contact.id}`}>
                  {t('Contact information')}
                </Label>
                <Input
                  id={`value-${contact.id}`}
                  maxLength={128}
                  value={contact.value}
                  onChange={(event) =>
                    updateContact(index, { value: event.target.value })
                  }
                />
              </div>
              <Button
                type='button'
                variant='ghost'
                size='icon'
                aria-label={t('Delete')}
                onClick={() => {
                  setContacts((current) =>
                    (current ?? contacts).filter((_, i) => i !== index)
                  )
                  setFiles((current) => {
                    const next = { ...current }
                    delete next[contact.id]
                    return next
                  })
                }}
              >
                <Trash2 className='size-4' />
              </Button>
              {(contact.type === 'qq_group' ||
                contact.type === 'wechat_group') && (
                <SupportQRCodeField
                  id={contact.id}
                  url={contact.qr_code_url}
                  file={files[contact.id]}
                  disabled={saving}
                  onChange={(file) => {
                    setFiles((current) => {
                      const next = { ...current }
                      if (file) next[contact.id] = file
                      else delete next[contact.id]
                      return next
                    })
                    if (!file) updateContact(index, { qr_code_url: '' })
                  }}
                />
              )}
            </div>
          ))}
        </fieldset>

        {!loading && contacts.length === 0 && (
          <div className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
            {t('No support contacts configured')}
          </div>
        )}

        <div className='flex flex-wrap justify-between gap-2'>
          <Button
            type='button'
            variant='outline'
            disabled={loading || saving || contacts.length >= 30}
            onClick={() =>
              setContacts((current) => [
                ...(current ?? contacts),
                emptyContact(),
              ])
            }
          >
            <Plus className='size-4' />
            {t('Add contact')}
          </Button>
          <Button
            type='button'
            disabled={saving || loading}
            onClick={handleSave}
          >
            <Save className='size-4' />
            {t('Save')}
          </Button>
        </div>
      </div>
    </SettingsSection>
  )
}
