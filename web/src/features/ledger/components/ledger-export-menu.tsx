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
import {
  CalendarDays,
  CalendarRange,
  Database,
  Download,
  List,
  Loader2,
} from 'lucide-react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import dayjs from '@/lib/dayjs'

import { downloadLedgerEntries } from '../api'
import {
  buildLedgerExportParams,
  type LedgerExportSelection,
} from '../lib/export'
import type { LedgerListParams } from '../types'

interface LedgerExportMenuProps {
  params: LedgerListParams
}

function toInputValue(timestamp?: number): string {
  return timestamp ? dayjs(timestamp * 1000).format('YYYY-MM-DDTHH:mm') : ''
}

export function LedgerExportMenu(props: LedgerExportMenuProps) {
  const { t } = useTranslation()
  const startInputId = useId()
  const endInputId = useId()
  const [exporting, setExporting] = useState(false)
  const [customOpen, setCustomOpen] = useState(false)
  const [customStart, setCustomStart] = useState('')
  const [customEnd, setCustomEnd] = useState('')

  const exportEntries = async (selection: LedgerExportSelection) => {
    setExporting(true)
    try {
      const result = await downloadLedgerEntries(
        buildLedgerExportParams(props.params, selection)
      )
      const url = URL.createObjectURL(result.blob)
      const link = document.createElement('a')
      link.href = url
      link.download = result.filename
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
      if (result.truncated) {
        toast.warning(
          t('Export limited to first {{count}} records', {
            count: result.limit,
          })
        )
      } else {
        toast.success(t('Ledger downloaded'))
      }
      return true
    } catch {
      toast.error(t('Failed to download ledger'))
      return false
    } finally {
      setExporting(false)
    }
  }

  const openCustom = () => {
    setCustomStart(toInputValue(props.params.start_timestamp))
    setCustomEnd(toInputValue(props.params.end_timestamp))
    setCustomOpen(true)
  }
  const startDate = new Date(customStart)
  const endDate = new Date(customEnd)
  const customValid =
    customStart !== '' &&
    customEnd !== '' &&
    Number.isFinite(startDate.getTime()) &&
    Number.isFinite(endDate.getTime()) &&
    startDate <= endDate

  const downloadCustom = async () => {
    if (!customValid) return
    const downloaded = await exportEntries({
      scope: 'custom',
      start: startDate,
      end: endDate,
    })
    if (downloaded) setCustomOpen(false)
  }

  return (
    <>
      <Tooltip>
        <DropdownMenu>
          <TooltipTrigger
            render={
              <DropdownMenuTrigger
                render={
                  <Button
                    type='button'
                    variant='outline'
                    size='icon'
                    disabled={exporting}
                    aria-label={t('Download CSV')}
                  />
                }
              >
                {exporting ? (
                  <Loader2 className='animate-spin' />
                ) : (
                  <Download />
                )}
              </DropdownMenuTrigger>
            }
          />
          <TooltipContent>{t('Download CSV')}</TooltipContent>
          <DropdownMenuContent align='end' className='w-44'>
            <DropdownMenuGroup>
              <DropdownMenuLabel>{t('Download CSV')}</DropdownMenuLabel>
              <DropdownMenuItem
                disabled={exporting}
                onClick={() => void exportEntries({ scope: 'page' })}
              >
                <List />
                {t('Current page')}
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={exporting}
                onClick={() => void exportEntries({ scope: 'today' })}
              >
                <CalendarDays />
                {t('Today')}
              </DropdownMenuItem>
              <DropdownMenuItem disabled={exporting} onClick={openCustom}>
                <CalendarRange />
                {t('Select time range')}
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={exporting}
                onClick={() => void exportEntries({ scope: 'all' })}
              >
                <Database />
                {t('All records')}
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </Tooltip>

      <Dialog open={customOpen} onOpenChange={setCustomOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Select time range')}</DialogTitle>
            <DialogDescription className='sr-only'>
              {t('Select time range')}
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-3 sm:grid-cols-2'>
            <div className='space-y-1.5'>
              <label htmlFor={startInputId} className='text-sm font-medium'>
                {t('Start Time')}
              </label>
              <Input
                id={startInputId}
                type='datetime-local'
                value={customStart}
                onChange={(event) => setCustomStart(event.target.value)}
              />
            </div>
            <div className='space-y-1.5'>
              <label htmlFor={endInputId} className='text-sm font-medium'>
                {t('End Time')}
              </label>
              <Input
                id={endInputId}
                type='datetime-local'
                value={customEnd}
                onChange={(event) => setCustomEnd(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => setCustomOpen(false)}
              disabled={exporting}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              onClick={() => void downloadCustom()}
              disabled={!customValid || exporting}
            >
              {exporting ? (
                <Loader2 data-icon='inline-start' className='animate-spin' />
              ) : (
                <Download data-icon='inline-start' />
              )}
              {t('Download')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
