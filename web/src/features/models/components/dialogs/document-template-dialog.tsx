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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { getServerErrorMessage } from '@/lib/server-error-message'

import {
  documentTemplatesQueryKeys,
  getDocumentTemplate,
  type DocumentTemplate,
} from '../../document-templates-api'
import { DocumentTemplateForm } from './document-template-form'

export type DocumentTemplateDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  mode: 'upload' | 'edit' | 'rename'
  document?: DocumentTemplate
}

export function DocumentTemplateDialog(props: DocumentTemplateDialogProps) {
  const { t } = useTranslation()
  const id = props.document?.id ?? 0
  const query = useQuery({
    queryKey: documentTemplatesQueryKeys.detail(id),
    queryFn: () => getDocumentTemplate(id),
    enabled: props.open && props.mode === 'edit' && id > 0,
  })
  const titleKeys = {
    upload: 'Upload document template',
    edit: 'Edit document template',
    rename: 'Rename document template',
  }
  if (!props.open) return null
  if (props.mode === 'edit' && query.isPending) {
    return (
      <Dialog open onOpenChange={props.onOpenChange} title={t(titleKeys.edit)}>
        <LoadingState />
      </Dialog>
    )
  }
  if (props.mode === 'edit' && query.isError) {
    return (
      <Dialog open onOpenChange={props.onOpenChange} title={t(titleKeys.edit)}>
        <ErrorState
          description={getServerErrorMessage(query.error)}
          onRetry={() => void query.refetch()}
        />
      </Dialog>
    )
  }
  return (
    <DocumentTemplateForm
      key={`${props.mode}-${id}`}
      mode={props.mode}
      document={props.mode === 'edit' ? query.data : props.document}
      onOpenChange={props.onOpenChange}
      title={t(titleKeys[props.mode])}
    />
  )
}
