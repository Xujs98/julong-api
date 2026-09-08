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
import { Clock3, ReceiptText } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  applyForInvoice,
  cancelInvoiceApplication,
  completeInvoiceApplication,
  downloadInvoiceAttachment,
  getAdminInvoiceApplications,
  getInvoiceInfo,
  updateInvoiceApplication,
  withdrawCompletedInvoiceApplication,
} from './api'
import { ApplicationsTable } from './components/applications-table'
import { CancelInvoiceDialog } from './components/cancel-invoice-dialog'
import { EligibleCreditsTable } from './components/eligible-credits-table'
import { InvoiceAvailabilityNotice } from './components/invoice-availability-notice'
import { ManageInvoiceDialog } from './components/manage-invoice-dialog'
import type { InvoiceApplication, InvoiceCredit, InvoiceStatus } from './types'

const PAGE_SIZE = 20

export function Invoices() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const role = useAuthStore((state) => state.auth.user?.role ?? ROLE.GUEST)
  const isAdmin = role >= ROLE.ADMIN
  const [creditPage, setCreditPage] = useState(1)
  const [applicationPage, setApplicationPage] = useState(1)
  const [adminPage, setAdminPage] = useState(1)
  const [selected, setSelected] = useState(new Map<string, InvoiceCredit>())
  const [managedApplication, setManagedApplication] =
    useState<InvoiceApplication | null>(null)
  const [cancellingApplication, setCancellingApplication] =
    useState<InvoiceApplication | null>(null)

  const invoiceQuery = useQuery({
    queryKey: ['invoice', creditPage, applicationPage],
    queryFn: () => getInvoiceInfo(creditPage, applicationPage, PAGE_SIZE),
  })
  const adminQuery = useQuery({
    queryKey: ['invoice', 'admin', adminPage],
    queryFn: () => getAdminInvoiceApplications(adminPage, PAGE_SIZE),
    enabled: isAdmin,
  })
  const applyMutation = useMutation({
    mutationFn: (requestIds: string[]) => applyForInvoice(requestIds),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(
          response.message || t('Failed to submit invoice application')
        )
        return
      }
      setSelected(new Map())
      toast.success(t('Invoice application submitted'))
      await queryClient.invalidateQueries({ queryKey: ['invoice'] })
    },
  })
  const updateMutation = useMutation({
    mutationFn: (request: {
      id: number
      status: InvoiceStatus
      adminNote: string
    }) =>
      updateInvoiceApplication(request.id, request.status, request.adminNote),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(
          response.message || t('Failed to update invoice application')
        )
        return
      }
      setManagedApplication(null)
      setSelected(new Map())
      toast.success(t('Invoice application updated'))
      await queryClient.invalidateQueries({ queryKey: ['invoice'] })
    },
  })
  const completeMutation = useMutation({
    mutationFn: (request: {
      id: number
      locale: 'zh' | 'en'
      adminNote: string
      attachment: File
    }) =>
      completeInvoiceApplication(
        request.id,
        request.locale,
        request.adminNote,
        request.attachment
      ),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(
          response.message || t('Failed to complete invoice application')
        )
        return
      }
      setManagedApplication(null)
      toast.success(t('Invoice email sent and application completed'))
      await queryClient.invalidateQueries({ queryKey: ['invoice'] })
    },
  })
  const cancelMutation = useMutation({
    mutationFn: (id: number) => cancelInvoiceApplication(id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(
          response.message || t('Failed to cancel invoice application')
        )
        return
      }
      setCancellingApplication(null)
      setSelected(new Map())
      toast.success(t('Invoice application cancelled'))
      await queryClient.invalidateQueries({ queryKey: ['invoice'] })
    },
  })
  const withdrawMutation = useMutation({
    mutationFn: (id: number) => withdrawCompletedInvoiceApplication(id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(
          response.message || t('Failed to withdraw invoice application')
        )
        return
      }
      setManagedApplication(null)
      toast.success(t('Invoice application withdrawn'))
      await queryClient.invalidateQueries({ queryKey: ['invoice'] })
    },
    onError: () => toast.error(t('Failed to withdraw invoice application')),
  })
  const downloadMutation = useMutation({
    mutationFn: (application: InvoiceApplication) =>
      downloadInvoiceAttachment(application.id),
    onSuccess: (response, application) => {
      const disposition = String(response.headers['content-disposition'] || '')
      const encodedFilename = disposition.match(
        /filename\*=UTF-8''([^;]+)/i
      )?.[1]
      const plainFilename = disposition.match(/filename="?([^";]+)"?/i)?.[1]
      let filename = `invoice-${application.id}`
      try {
        filename = decodeURIComponent(
          encodedFilename || plainFilename || filename
        )
      } catch {
        filename = plainFilename || filename
      }
      const url = URL.createObjectURL(response.data)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = filename
      document.body.append(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(url)
    },
    onError: () => toast.error(t('Failed to download invoice')),
  })

  const data = invoiceQuery.data?.data
  const adminData = adminQuery.data?.data

  function toggleCredit(credit: InvoiceCredit, checked: boolean) {
    setSelected((current) => {
      const next = new Map(current)
      if (checked) next.set(credit.request_id, credit)
      else next.delete(credit.request_id)
      return next
    })
  }

  function toggleCurrentPage(checked: boolean) {
    setSelected((current) => {
      const next = new Map(current)
      const now = Math.floor(Date.now() / 1000)
      for (const credit of data?.credits ?? []) {
        if ((credit.frozen_until ?? 0) > now) continue
        if (checked) next.set(credit.request_id, credit)
        else next.delete(credit.request_id)
      }
      return next
    })
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Self-service Invoices')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='space-y-5 pb-4'>
            <section className='border-primary bg-card rounded-lg border border-l-4 px-5 py-4'>
              <div className='space-y-1'>
                <h3 className='text-lg font-semibold'>
                  {t('Self-service Invoices')}
                </h3>
                <p className='text-muted-foreground'>
                  {t(
                    'Select one or more eligible credits and submit an application after the total reaches the invoice threshold.'
                  )}
                </p>
              </div>
              <div className='text-muted-foreground mt-4 grid gap-3 text-sm md:grid-cols-2'>
                <span className='flex items-center gap-2'>
                  <Clock3 className='size-4' />
                  {t(
                    'Invoices are usually delivered within about {{days}} working days.',
                    {
                      days: data?.processing_days ?? '-',
                    }
                  )}
                </span>
                <span className='flex items-center gap-2'>
                  <ReceiptText className='size-4' />
                  {t(
                    'Eligible sources include online top-ups, redemptions, and administrator credits.'
                  )}
                </span>
              </div>
            </section>

            {invoiceQuery.isPending && <LoadingState />}
            {!invoiceQuery.isPending && !data && (
              <ErrorState
                title={t('Failed to load invoice data')}
                description={
                  invoiceQuery.data?.message || t('Please try again later')
                }
                onRetry={() => invoiceQuery.refetch()}
              />
            )}
            {data && (
              <>
                <InvoiceAvailabilityNotice enabled={data.enabled} />

                <EligibleCreditsTable
                  credits={data?.credits ?? []}
                  total={data?.credits_total ?? 0}
                  page={creditPage}
                  pageSize={data?.page_size ?? PAGE_SIZE}
                  unit={data?.unit ?? 'USD'}
                  minimumAmount={data?.minimum_amount ?? 0}
                  enabled={data?.enabled ?? false}
                  selected={selected}
                  applying={applyMutation.isPending}
                  onToggle={toggleCredit}
                  onTogglePage={toggleCurrentPage}
                  onClear={() => setSelected(new Map())}
                  onApply={() => applyMutation.mutate([...selected.keys()])}
                  onRefresh={() => invoiceQuery.refetch()}
                  onPageChange={setCreditPage}
                />

                <ApplicationsTable
                  title={t('Application history')}
                  description={t(
                    'Invoices are delivered to the email address captured when the application is submitted.'
                  )}
                  applications={data?.applications ?? []}
                  total={data?.applications_total ?? 0}
                  page={applicationPage}
                  pageSize={data?.page_size ?? PAGE_SIZE}
                  onPageChange={setApplicationPage}
                  onCancel={setCancellingApplication}
                  onDownload={(application) =>
                    downloadMutation.mutate(application)
                  }
                />

                {isAdmin && adminQuery.isPending && <LoadingState />}
                {isAdmin && !adminQuery.isPending && !adminData && (
                  <ErrorState
                    title={t('Failed to load invoice data')}
                    description={
                      adminQuery.data?.message || t('Please try again later')
                    }
                    onRetry={() => adminQuery.refetch()}
                  />
                )}
                {isAdmin && adminData && (
                  <ApplicationsTable
                    title={t('User invoice applications')}
                    description={t(
                      'Review applications from all users and update their processing status.'
                    )}
                    applications={adminData.items}
                    total={adminData.total}
                    page={adminPage}
                    pageSize={adminData.page_size ?? PAGE_SIZE}
                    admin
                    onPageChange={setAdminPage}
                    onManage={setManagedApplication}
                    onDownload={(application) =>
                      downloadMutation.mutate(application)
                    }
                  />
                )}
              </>
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <ManageInvoiceDialog
        application={managedApplication}
        saving={updateMutation.isPending || completeMutation.isPending}
        withdrawing={withdrawMutation.isPending}
        onOpenChange={(open) => {
          if (!open) setManagedApplication(null)
        }}
        onSave={(status, adminNote) => {
          if (!managedApplication) return
          updateMutation.mutate({
            id: managedApplication.id,
            status,
            adminNote,
          })
        }}
        onComplete={(locale, adminNote, attachment) => {
          if (!managedApplication) return
          completeMutation.mutate({
            id: managedApplication.id,
            locale,
            adminNote,
            attachment,
          })
        }}
        onDownload={(application) => downloadMutation.mutate(application)}
        onWithdraw={(application) => withdrawMutation.mutate(application.id)}
      />
      <CancelInvoiceDialog
        application={cancellingApplication}
        cancelling={cancelMutation.isPending}
        onOpenChange={(open) => {
          if (!open) setCancellingApplication(null)
        }}
        onConfirm={() => {
          if (!cancellingApplication) return
          cancelMutation.mutate(cancellingApplication.id)
        }}
      />
    </>
  )
}
