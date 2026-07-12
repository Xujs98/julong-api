import { Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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

import {
  getSupportContacts,
  updateSupportContacts,
  type SupportContact,
  type SupportContactType,
} from '../api'
import { SettingsSection } from '../components/settings-section'

const emptyContact = (): SupportContact => ({
  id: crypto.randomUUID(),
  type: 'qq',
  label: '',
  value: '',
})

export function SupportContactsSection() {
  const { t } = useTranslation()
  const [contacts, setContacts] = useState<SupportContact[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    async function loadContacts() {
      try {
        const response = await getSupportContacts()
        setContacts(
          (response.data || []).map((contact) => ({
            ...contact,
            id: contact.id || crypto.randomUUID(),
          }))
        )
      } catch {
        toast.error(t('Failed to load support contacts'))
      } finally {
        setLoading(false)
      }
    }
    void loadContacts()
  }, [t])

  const updateContact = (index: number, patch: Partial<SupportContact>) => {
    setContacts((current) =>
      current.map((contact, itemIndex) =>
        itemIndex === index ? { ...contact, ...patch } : contact
      )
    )
  }

  const handleSave = async () => {
    if (contacts.some((contact) => !contact.value.trim())) {
      toast.error(t('Contact value is required'))
      return
    }
    setSaving(true)
    try {
      const response = await updateSupportContacts(contacts)
      if (response.success) toast.success(t('Setting updated successfully'))
    } catch {
      toast.error(t('Failed to update setting'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <SettingsSection title={t('Customer Support')}>
      <div className='space-y-4'>
        <div>
          <h3 className='text-sm font-medium'>{t('Support contacts')}</h3>
          <p className='text-muted-foreground mt-1 text-xs'>
            {t(
              'Configure multiple QQ, WeChat, and phone contacts shown to users.'
            )}
          </p>
        </div>

        <div className='space-y-3'>
          {contacts.map((contact, index) => (
            <div
              key={contact.id}
              className='grid gap-3 rounded-lg border p-3 sm:grid-cols-[140px_minmax(0,0.8fr)_minmax(0,1.2fr)_36px] sm:items-end'
            >
              <div className='space-y-1.5'>
                <Label>{t('Contact type')}</Label>
                <Select
                  value={contact.type}
                  onValueChange={(value) =>
                    value &&
                    updateContact(index, { type: value as SupportContactType })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='qq'>QQ</SelectItem>
                    <SelectItem value='wechat'>{t('WeChat')}</SelectItem>
                    <SelectItem value='phone'>{t('Phone')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-1.5'>
                <Label>{t('Display name')}</Label>
                <Input
                  value={contact.label}
                  placeholder={t('Optional')}
                  onChange={(event) =>
                    updateContact(index, { label: event.target.value })
                  }
                />
              </div>
              <div className='space-y-1.5'>
                <Label>{t('Contact information')}</Label>
                <Input
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
                onClick={() =>
                  setContacts((current) =>
                    current.filter((_, i) => i !== index)
                  )
                }
              >
                <Trash2 className='size-4' />
              </Button>
            </div>
          ))}
        </div>

        {!loading && contacts.length === 0 && (
          <div className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
            {t('No support contacts configured')}
          </div>
        )}

        <div className='flex flex-wrap justify-between gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={() =>
              setContacts((current) => [...current, emptyContact()])
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
