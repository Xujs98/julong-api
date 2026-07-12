import { useQuery } from '@tanstack/react-query'
import { Headphones, Phone } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FaQq } from 'react-icons/fa6'
import { SiWechat } from 'react-icons/si'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  getSupportContacts,
  type SupportContact,
} from '@/features/system-settings/api'
import { useSystemConfig } from '@/hooks/use-system-config'

function ContactIcon({ type }: { type: SupportContact['type'] }) {
  if (type === 'qq') return <FaQq className='size-4 text-[#12b7f5]' />
  if (type === 'wechat') return <SiWechat className='size-4 text-[#07c160]' />
  return <Phone className='size-4 text-[#16a34a]' />
}

function contactTypeLabel(contact: SupportContact, t: (key: string) => string) {
  if (contact.label) return contact.label
  if (contact.type === 'qq') return 'QQ'
  if (contact.type === 'wechat') return t('WeChat')
  return t('Phone')
}

export function SupportContactButton() {
  const { t } = useTranslation()
  const { logo, systemName } = useSystemConfig()
  const [open, setOpen] = useState(false)
  const query = useQuery({
    queryKey: ['support-contacts'],
    queryFn: getSupportContacts,
    staleTime: 5 * 60 * 1000,
  })
  const contacts = query.data?.data || []

  return (
    <>
      <Button
        type='button'
        variant='outline'
        size='sm'
        onClick={() => setOpen(true)}
      >
        <Headphones className='size-4' />
        {t('Contact Support')}
      </Button>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title={t('Contact Support')}
        description={t('Choose a contact method below')}
        contentClassName='sm:max-w-md'
        contentHeight='auto'
      >
        <div className='flex flex-col items-center py-4 text-center'>
          <img
            src={logo}
            alt={systemName}
            className='size-20 rounded-lg border object-contain'
          />
          <h3 className='mt-3 text-base font-semibold'>{systemName}</h3>
        </div>
        <div className='space-y-2'>
          {contacts.map((contact) => (
            <div
              key={contact.id}
              className='flex min-w-0 items-center gap-3 rounded-lg border px-3 py-2.5'
            >
              <div className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-md'>
                <ContactIcon type={contact.type} />
              </div>
              <div className='min-w-0 flex-1 text-left'>
                <div className='text-muted-foreground text-xs'>
                  {contactTypeLabel(contact, t)}
                </div>
                <div className='truncate text-sm font-medium'>
                  {contact.value}
                </div>
              </div>
              <CopyButton
                value={contact.value}
                variant='ghost'
                className='size-9 shrink-0'
              />
            </div>
          ))}
          {!query.isLoading && contacts.length === 0 && (
            <div className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
              {t('No support contacts configured')}
            </div>
          )}
        </div>
      </Dialog>
    </>
  )
}
