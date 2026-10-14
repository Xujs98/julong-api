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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { DataTablePage, useDataTable } from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatTimestampToDate } from '@/lib/format'
import { getServerErrorMessage } from '@/lib/server-error-message'

import {
  deleteDocumentTemplate,
  documentTemplatesQueryKeys,
  invalidateDocumentTemplateData,
  listDocumentTemplates,
  type DocumentTemplate,
} from '../document-templates-api'
import { DocumentTemplateDialog } from './dialogs/document-template-dialog'

export function DocumentTemplatesTable() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 20 })
  const [keyword, setKeyword] = useState('')
  const [operation, setOperation] = useState<{
    mode: 'edit' | 'rename'
    document: DocumentTemplate
  } | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<DocumentTemplate | null>(
    null
  )
  const params = {
    p: pagination.pageIndex + 1,
    page_size: pagination.pageSize,
    keyword,
  }
  const query = useQuery({
    queryKey: documentTemplatesQueryKeys.list(params),
    queryFn: () => listDocumentTemplates(params),
  })
  const remove = useMutation({
    mutationFn: deleteDocumentTemplate,
    onSuccess: async () => {
      await invalidateDocumentTemplateData(client)
      toast.success(t('Document template deleted'))
      setDeleteTarget(null)
    },
    onError: () => undefined,
  })
  const columns: ColumnDef<DocumentTemplate>[] = [
    {
      accessorKey: 'created_time',
      header: t('Uploaded At'),
      size: 200,
      cell: ({ row }) => formatTimestampToDate(row.original.created_time),
    },
    {
      accessorKey: 'name',
      header: t('Document name'),
      size: 420,
      enableHiding: false,
      meta: { mobileTitle: true },
      cell: ({ row }) => (
        <div className='flex min-w-0 items-center gap-2'>
          <span
            className='min-w-0 truncate font-medium'
            title={row.original.name}
          >
            {row.original.name}
          </span>
          <Badge variant='outline' className='shrink-0'>
            {row.original.format === 'docx' ? 'Word' : 'Markdown'}
          </Badge>
        </div>
      ),
    },
    {
      id: 'actions',
      header: t('Actions'),
      size: 230,
      enableHiding: false,
      cell: ({ row }) => (
        <div className='flex flex-wrap items-center gap-1'>
          <Button
            size='sm'
            variant='ghost'
            onClick={() =>
              setOperation({ mode: 'edit', document: row.original })
            }
          >
            {t('Edit')}
          </Button>
          <Button
            size='sm'
            variant='ghost'
            onClick={() =>
              setOperation({ mode: 'rename', document: row.original })
            }
          >
            {t('Rename')}
          </Button>
          <Button
            size='sm'
            variant='ghost'
            className='text-destructive'
            onClick={() => {
              remove.reset()
              setDeleteTarget(row.original)
            }}
          >
            {t('Delete')}
          </Button>
        </div>
      ),
    },
  ]
  const { table } = useDataTable({
    data: query.data?.items ?? [],
    columns,
    totalCount: query.data?.total ?? 0,
    getRowId: (document) => String(document.id),
    pagination,
    onPaginationChange: setPagination,
    globalFilter: keyword,
    onGlobalFilterChange: (value) => {
      setKeyword(typeof value === 'function' ? value(keyword) : value)
      setPagination((current) => ({ ...current, pageIndex: 0 }))
    },
    columnFilters: [],
    manualPagination: true,
    manualFiltering: true,
    enableSorting: false,
    ensurePageInRange: (count) => {
      if (
        !query.isPending &&
        !query.isError &&
        pagination.pageIndex >= count &&
        pagination.pageIndex > 0
      ) {
        setPagination((current) => ({
          ...current,
          pageIndex: Math.max(0, count - 1),
        }))
      }
    },
  })
  if (query.isError) {
    return (
      <ErrorState
        description={getServerErrorMessage(query.error)}
        onRetry={() => void query.refetch()}
      />
    )
  }
  return (
    <>
      <DataTablePage
        table={table}
        columns={columns}
        isLoading={query.isLoading}
        isFetching={query.isFetching}
        applyHeaderSize
        emptyTitle={t('No document templates')}
        emptyDescription={t(
          'Upload a Markdown or Word document, then select it in a model’s endpoint settings.'
        )}
        toolbarProps={{
          searchPlaceholder: t('Search document names'),
          searchDebounceMs: 300,
        }}
      />
      {operation && (
        <DocumentTemplateDialog
          open
          mode={operation.mode}
          document={operation.document}
          onOpenChange={(open) => {
            if (!open) setOperation(null)
          }}
        />
      )}
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open && !remove.isPending) setDeleteTarget(null)
        }}
        title={t('Delete document template?')}
        desc={t(
          'Delete this document template? Linked models will return to their default API examples.'
        )}
        confirmText={t('Delete')}
        destructive
        isLoading={remove.isPending}
        handleConfirm={() => {
          if (deleteTarget) remove.mutate(deleteTarget.id)
        }}
      >
        {deleteTarget && (
          <p className='text-sm font-medium break-all'>{deleteTarget.name}</p>
        )}
        {remove.isError && (
          <Alert variant='destructive'>
            <AlertDescription>
              {getServerErrorMessage(remove.error)}
            </AlertDescription>
          </Alert>
        )}
      </ConfirmDialog>
    </>
  )
}
