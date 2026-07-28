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
import type { Table } from '@tanstack/react-table'
import { CopyPlus, Download, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { DataTableBulkActions as BulkActionsToolbar } from '@/components/data-table'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { downloadRedemptionCodes, formatRedemptionCodes } from '../lib/utils'
import type { Redemption } from '../types'
import { RedemptionsMultiDeleteDialog } from './redemptions-multi-delete-dialog'

type DataTableBulkActionsProps<TData> = {
  table: Table<TData>
}

export function DataTableBulkActions<TData>({
  table,
}: DataTableBulkActionsProps<TData>) {
  const { t } = useTranslation()
  const currentUser = useAuthStore((state) => state.auth.user)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const selectedRows = table.getFilteredSelectedRowModel().rows
  const selectedRedemptions = useMemo(
    () => selectedRows.map((row) => row.original as Redemption),
    [selectedRows]
  )

  const codesOnly = useMemo(
    () => formatRedemptionCodes(selectedRedemptions, false),
    [selectedRedemptions]
  )
  const namesAndCodes = useMemo(
    () => formatRedemptionCodes(selectedRedemptions, true),
    [selectedRedemptions]
  )
  const isAdmin = (currentUser?.role ?? 0) >= ROLE.ADMIN

  return (
    <>
      <BulkActionsToolbar table={table} entityName={t('redemption code')}>
        <CopyButton
          value={codesOnly}
          variant='outline'
          size='icon'
          className='size-8'
          tooltip={t('Copy codes only')}
          successTooltip={t('Codes copied!')}
          aria-label={t('Copy codes only')}
        />
        <CopyButton
          value={namesAndCodes}
          icon={<CopyPlus className='size-4' />}
          variant='outline'
          size='icon'
          className='size-8'
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
                className='size-8'
                onClick={() => downloadRedemptionCodes(selectedRedemptions)}
                aria-label={t('Download redemption codes')}
              />
            }
          >
            <Download className='size-4' />
          </TooltipTrigger>
          <TooltipContent>{t('Download redemption codes')}</TooltipContent>
        </Tooltip>
        {isAdmin && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type='button'
                  variant='destructive'
                  size='icon'
                  className='size-8'
                  onClick={() => setShowDeleteConfirm(true)}
                  aria-label={t('Delete selected redemption codes')}
                />
              }
            >
              <Trash2 className='size-4' />
            </TooltipTrigger>
            <TooltipContent>
              {t('Delete selected redemption codes')}
            </TooltipContent>
          </Tooltip>
        )}
      </BulkActionsToolbar>

      {showDeleteConfirm && (
        <RedemptionsMultiDeleteDialog
          table={table}
          onOpenChange={setShowDeleteConfirm}
        />
      )}
    </>
  )
}
