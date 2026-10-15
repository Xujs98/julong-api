import { useQuery } from '@tanstack/react-query'
import { ExternalLink, Headphones, Phone } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FaQq } from 'react-icons/fa6'
import { SiWechat } from 'react-icons/si'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import {
  getSupportContacts,
  type SupportContact,
} from '@/features/system-settings/api'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useAuthStore } from '@/stores/auth-store'

function ContactIcon({ type }: { type: SupportContact['type'] }) {
  if (type === 'qq' || type === 'qq_group') {
    return <FaQq className='size-4 text-[#12b7f5]' />
  }
  if (type === 'wechat' || type === 'wechat_group') {
    return <SiWechat className='size-4 text-[#07c160]' />
  }
  return <Phone className='size-4 text-[#16a34a]' />
}

function contactTypeLabel(contact: SupportContact, t: (key: string) => string) {
  if (contact.label) return contact.label
  if (contact.type === 'qq_group') return t('QQ group')
  if (contact.type === 'wechat_group') return t('WeChat group')
  if (contact.type === 'qq') return 'QQ'
  if (contact.type === 'wechat') return t('WeChat')
  return t('Phone')
}

function qqChatUrl(value: string) {
  const qq = encodeURIComponent(value.trim())
  return `tencent://message/?uin=${qq}&Site=&Menu=yes`
}

export function SupportContactButton(props: { compact?: boolean }) {
  const { t } = useTranslation()
  const { logo, systemName } = useSystemConfig()
  const [open, setOpen] = useState(false)
  const [preview, setPreview] = useState<SupportContact | null>(null)
  const user = useAuthStore((state) => state.auth.user)
  const query = useQuery({
    queryKey: ['support-contacts'],
    queryFn: getSupportContacts,
    enabled: open && !!user,
    staleTime: 0,
  })
  const contacts = query.data?.data || []

  if (!user) return null

  return (
    <>
      <Button
        type='button'
        variant={props.compact ? 'ghost' : 'outline'}
        size={props.compact ? 'icon' : 'sm'}
        aria-label={t('Contact Support')}
        title={t('Contact Support')}
        aria-expanded={open}
        aria-haspopup='dialog'
        onClick={() => setOpen(true)}
      >
        <Headphones className='size-4' />
        {!props.compact && t('Contact Support')}
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
        {query.isLoading && <LoadingState />}
        {query.isError && (
          <ErrorState
            title={t('Failed to load support contacts')}
            onRetry={() => void query.refetch()}
          />
        )}
        <div className='space-y-2'>
          {contacts.map((contact) => (
            <div
              key={contact.id}
              className='min-w-0 space-y-3 rounded-lg border px-3 py-2.5'
            >
              <div className='flex min-w-0 items-center gap-3'>
                {contact.type === 'qq' ? (
                  <a
                    href={qqChatUrl(contact.value)}
                    className='focus-visible:ring-ring flex min-w-0 flex-1 items-center gap-3 rounded-md outline-none focus-visible:ring-2'
                    aria-label={`${t('Contact Support')}: QQ ${contact.value}`}
                  >
                    <div className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-md'>
                      <ContactIcon type={contact.type} />
                    </div>
                    <div className='min-w-0 flex-1 text-left'>
                      <div className='text-muted-foreground text-xs break-words'>
                        {contactTypeLabel(contact, t)}
                      </div>
                      <div className='text-sm font-medium break-all'>
                        {contact.value}
                      </div>
                    </div>
                    <ExternalLink className='text-muted-foreground size-4 shrink-0' />
                  </a>
                ) : (
                  <>
                    <div className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-md'>
                      <ContactIcon type={contact.type} />
                    </div>
                    <div className='min-w-0 flex-1 text-left'>
                      <div className='text-muted-foreground text-xs break-words'>
                        {contactTypeLabel(contact, t)}
                      </div>
                      <div className='text-sm font-medium break-all'>
                        {contact.value}
                      </div>
                    </div>
                  </>
                )}
                <CopyButton
                  value={contact.value}
                  variant='ghost'
                  className='size-9 shrink-0'
                />
              </div>
              {(contact.type === 'qq_group' ||
                contact.type === 'wechat_group') &&
                contact.qr_code_url && (
                  <div className='flex flex-col items-center gap-2 border-t pt-3'>
                    <img
                      src={contact.qr_code_url}
                      alt={t('Group QR code')}
                      className='size-48 max-w-full rounded bg-white object-contain p-2'
                      loading='lazy'
                    />
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => setPreview(contact)}
                    >
                      {t('View QR code')}
                    </Button>
                  </div>
                )}
            </div>
          ))}
          {!query.isLoading && !query.isError && contacts.length === 0 && (
            <div className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
              {t('No support contacts configured')}
            </div>
          )}
        </div>
      </Dialog>
      <Dialog
        open={!!preview}
        onOpenChange={(value) => {
          if (!value) setPreview(null)
        }}
        title={preview?.label || t('Group QR code')}
        contentClassName='sm:max-w-md'
      >
        {preview?.qr_code_url && (
          <img
            src={preview.qr_code_url}
            alt={t('Group QR code')}
            className='mx-auto max-h-[60dvh] max-w-full bg-white object-contain p-3'
          />
        )}
      </Dialog>
    </>
  )
}
