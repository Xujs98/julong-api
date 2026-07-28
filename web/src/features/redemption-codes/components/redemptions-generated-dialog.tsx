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
import { CopyPlus, Download } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { SUCCESS_MESSAGES } from '../constants'
import { downloadRedemptionCodes, formatRedemptionCodes } from '../lib/utils'

type RedemptionsGeneratedDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  name: string
  codes: string[]
}

export function RedemptionsGeneratedDialog({
  open,
  onOpenChange,
  name,
  codes,
}: RedemptionsGeneratedDialogProps) {
  const { t } = useTranslation()
  const items = useMemo(
    () => codes.map((key) => ({ name, key })),
    [codes, name]
  )
  const codesOnly = useMemo(() => formatRedemptionCodes(items, false), [items])
  const namesAndCodes = useMemo(
    () => formatRedemptionCodes(items, true),
    [items]
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Redemption Codes')}</DialogTitle>
          <DialogDescription>
            {codes.length > 1
              ? t('Successfully created {{count}} redemption codes', {
                  count: codes.length,
                })
              : t(SUCCESS_MESSAGES.REDEMPTION_CREATED)}
          </DialogDescription>
        </DialogHeader>

        <div className='max-h-72 overflow-y-auto rounded-md border'>
          {codes.map((code, index) => (
            <div
              key={code}
              className='flex min-w-0 items-center gap-3 border-b px-3 py-2.5 last:border-b-0'
            >
              <span className='text-muted-foreground w-7 shrink-0 text-right text-xs tabular-nums'>
                {index + 1}
              </span>
              <code className='min-w-0 flex-1 truncate text-xs'>{code}</code>
            </div>
          ))}
        </div>

        <DialogFooter className='flex-row items-center justify-between'>
          <div className='flex items-center gap-2'>
            <CopyButton
              value={codesOnly}
              variant='outline'
              size='icon'
              className='size-9'
              tooltip={t('Copy codes only')}
              successTooltip={t('Codes copied!')}
              aria-label={t('Copy codes only')}
            />
            <CopyButton
              value={namesAndCodes}
              icon={<CopyPlus className='size-4' />}
              variant='outline'
              size='icon'
              className='size-9'
              tooltip={t('Copy names and codes')}
              successTooltip={t('Codes copied!')}
              aria-label={t('Copy names and codes')}
            />
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type='button'
                    variant='outline'
                    size='icon'
                    className='size-9'
                    onClick={() => downloadRedemptionCodes(items)}
                    aria-label={t('Download redemption codes')}
                  />
                }
              >
                <Download className='size-4' />
              </TooltipTrigger>
              <TooltipContent>{t('Download redemption codes')}</TooltipContent>
            </Tooltip>
          </div>

          <DialogClose render={<Button variant='outline' />}>
            {t('Close')}
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
