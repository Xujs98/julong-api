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
import { Ban, Loader2, MonitorSmartphone, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { sessionDevice } from '@/features/profile/components/login-session-utils'
import { formatTimestampToDate } from '@/lib/format'

import {
  getUserLoginDevices,
  getUserLoginIPs,
  updateUserLoginDevices,
  updateUserLoginIPs,
} from '../api'
import type { UserLoginDevice, UserLoginIP } from '../types'

type PendingAction = 'block' | 'unblock' | null

function LoadingRows() {
  return (
    <div className='space-y-2'>
      <Skeleton className='h-20 w-full' />
      <Skeleton className='h-20 w-full' />
    </div>
  )
}

function DeviceRow(props: {
  record: UserLoginDevice
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <div className='grid grid-cols-[auto_minmax(0,1fr)] gap-3 border-b p-3 last:border-b-0 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center'>
      <Checkbox
        checked={props.checked}
        onCheckedChange={(checked) => props.onCheckedChange(Boolean(checked))}
        aria-label={t('Select device')}
      />
      <div className='flex min-w-0 gap-3'>
        <div className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-md'>
          <MonitorSmartphone className='size-4' aria-hidden='true' />
        </div>
        <div className='min-w-0'>
          <div
            className='truncate text-sm font-medium'
            title={props.record.user_agent}
          >
            {sessionDevice(
              props.record.user_agent,
              t('Unknown device'),
              t('Browser')
            )}
          </div>
          <div className='text-muted-foreground mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs'>
            <span>{formatTimestampToDate(props.record.last_login_at)}</span>
            <span>
              {t('{{count}} logins', { count: props.record.login_count })}
            </span>
            <span className='font-mono'>
              {t('Device ID: {{id}}', {
                id: props.record.device_id.slice(0, 8),
              })}
            </span>
          </div>
          {props.record.ips.length > 0 && (
            <div className='mt-1.5 flex flex-wrap gap-1'>
              {props.record.ips.map((ip) => (
                <code
                  key={ip}
                  className='bg-muted rounded px-1.5 py-0.5 text-xs'
                >
                  {ip}
                </code>
              ))}
            </div>
          )}
        </div>
      </div>
      <div className='col-start-2 flex flex-wrap gap-1 sm:col-start-auto sm:justify-end'>
        {props.record.active_session_count > 0 && (
          <StatusBadge
            label={t('{{count}} active sessions', {
              count: props.record.active_session_count,
            })}
            variant='info'
            copyable={false}
          />
        )}
        <StatusBadge
          label={
            props.record.blocked ? t('Device blocked') : t('Device allowed')
          }
          variant={props.record.blocked ? 'danger' : 'success'}
          copyable={false}
        />
      </div>
    </div>
  )
}

function DevicePanel(props: { userId?: number }) {
  const { t } = useTranslation()
  const [selectedDeviceIDs, setSelectedDeviceIDs] = useState<string[]>([])
  const [pendingAction, setPendingAction] = useState<PendingAction>(null)
  const query = useQuery({
    queryKey: ['admin-user-login-devices', props.userId],
    enabled: Boolean(props.userId),
    queryFn: async () => {
      if (!props.userId) return []
      const result = await getUserLoginDevices(props.userId)
      if (!result.success) throw new Error(result.message || 'Load failed')
      return result.data || []
    },
  })
  const records = query.data || []
  const selectedRecords = records.filter((record) =>
    selectedDeviceIDs.includes(record.device_id)
  )
  const canBlock = selectedRecords.some((record) => !record.blocked)
  const canUnblock = selectedRecords.some((record) => record.blocked)

  const toggleDevice = (deviceID: string, checked: boolean) => {
    setSelectedDeviceIDs((current) =>
      checked
        ? [...new Set([...current, deviceID])]
        : current.filter((value) => value !== deviceID)
    )
  }

  const updateSelected = async (blocked: boolean) => {
    if (!props.userId) return
    const deviceIDs = selectedRecords
      .filter((record) => record.blocked !== blocked)
      .map((record) => record.device_id)
    if (deviceIDs.length === 0) return
    setPendingAction(blocked ? 'block' : 'unblock')
    try {
      const result = await updateUserLoginDevices(
        props.userId,
        deviceIDs,
        blocked
      )
      if (!result.success) {
        toast.error(result.message || t('Operation failed'))
        return
      }
      if (blocked) {
        toast.success(
          t('Devices blocked; {{count}} active sessions revoked', {
            count: result.data?.revoked_count || 0,
          })
        )
      } else {
        toast.success(t('Devices unblocked'))
      }
      setSelectedDeviceIDs([])
      await query.refetch()
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setPendingAction(null)
    }
  }

  if (query.isLoading) return <LoadingRows />
  if (query.isError) {
    return (
      <div className='flex h-28 flex-col items-center justify-center gap-2 rounded-lg border'>
        <span className='text-destructive text-sm'>
          {t('Failed to load login devices')}
        </span>
        <Button size='sm' variant='outline' onClick={() => query.refetch()}>
          {t('Retry')}
        </Button>
      </div>
    )
  }
  if (records.length === 0) {
    return (
      <div className='text-muted-foreground flex h-28 items-center justify-center rounded-lg border text-sm'>
        {t('No recognized devices')}
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap justify-end gap-2'>
        <Button
          size='sm'
          variant='destructive'
          disabled={!canBlock || pendingAction !== null}
          onClick={() => updateSelected(true)}
        >
          {pendingAction === 'block' ? (
            <Loader2 className='animate-spin' />
          ) : (
            <Ban />
          )}
          {t('Block devices')}
        </Button>
        <Button
          size='sm'
          variant='outline'
          disabled={!canUnblock || pendingAction !== null}
          onClick={() => updateSelected(false)}
        >
          {pendingAction === 'unblock' ? (
            <Loader2 className='animate-spin' />
          ) : (
            <ShieldCheck />
          )}
          {t('Unblock devices')}
        </Button>
      </div>
      <div className='overflow-hidden rounded-lg border'>
        {records.map((record) => (
          <DeviceRow
            key={record.device_id}
            record={record}
            checked={selectedDeviceIDs.includes(record.device_id)}
            onCheckedChange={(checked) =>
              toggleDevice(record.device_id, checked)
            }
          />
        ))}
      </div>
    </div>
  )
}

function LoginIPRow(props: {
  record: UserLoginIP
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <div className='grid grid-cols-[auto_minmax(0,1fr)] gap-3 border-b p-3 last:border-b-0 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center'>
      <Checkbox
        checked={props.checked}
        onCheckedChange={(checked) => props.onCheckedChange(Boolean(checked))}
        aria-label={t('Select IP')}
      />
      <div className='min-w-0'>
        <code className='text-sm font-medium break-all'>{props.record.ip}</code>
        <div className='text-muted-foreground mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs'>
          <span>{formatTimestampToDate(props.record.last_login_at)}</span>
          <span>
            {t('{{count}} logins', { count: props.record.login_count })}
          </span>
        </div>
      </div>
      <div className='col-start-2 flex flex-wrap gap-1 sm:col-start-auto sm:justify-end'>
        {props.record.shared_user_count > 1 && (
          <StatusBadge
            label={t('Shared IP: {{count}} users', {
              count: props.record.shared_user_count,
            })}
            variant='warning'
            copyable={false}
          />
        )}
        <StatusBadge
          label={props.record.blocked ? t('IP blocked') : t('IP allowed')}
          variant={props.record.blocked ? 'danger' : 'success'}
          copyable={false}
        />
      </div>
    </div>
  )
}

function LoginIPPanel(props: { userId?: number; onUpdated: () => void }) {
  const { t } = useTranslation()
  const [selectedIPs, setSelectedIPs] = useState<string[]>([])
  const [pendingAction, setPendingAction] = useState<PendingAction>(null)
  const query = useQuery({
    queryKey: ['admin-user-login-ips', props.userId],
    enabled: Boolean(props.userId),
    queryFn: async () => {
      if (!props.userId) return []
      const result = await getUserLoginIPs(props.userId)
      if (!result.success) throw new Error(result.message || 'Load failed')
      return result.data || []
    },
  })
  const records = query.data || []
  const selectedRecords = records.filter((record) =>
    selectedIPs.includes(record.ip)
  )
  const canBlock = selectedRecords.some((record) => !record.blocked)
  const canUnblock = selectedRecords.some((record) => record.blocked)

  const toggleIP = (ip: string, checked: boolean) => {
    setSelectedIPs((current) =>
      checked
        ? [...new Set([...current, ip])]
        : current.filter((value) => value !== ip)
    )
  }

  const updateSelected = async (blocked: boolean) => {
    if (!props.userId) return
    const ips = selectedRecords
      .filter((record) => record.blocked !== blocked)
      .map((record) => record.ip)
    if (ips.length === 0) return
    setPendingAction(blocked ? 'block' : 'unblock')
    try {
      const result = await updateUserLoginIPs(props.userId, ips, blocked)
      if (!result.success) {
        toast.error(result.message || t('Operation failed'))
        return
      }
      toast.success(
        blocked ? t('IP addresses blocked') : t('IP addresses unblocked')
      )
      setSelectedIPs([])
      await query.refetch()
      props.onUpdated()
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setPendingAction(null)
    }
  }

  if (query.isLoading) return <LoadingRows />
  if (query.isError) {
    return (
      <div className='flex h-28 flex-col items-center justify-center gap-2 rounded-lg border'>
        <span className='text-destructive text-sm'>
          {t('Failed to load login IPs')}
        </span>
        <Button size='sm' variant='outline' onClick={() => query.refetch()}>
          {t('Retry')}
        </Button>
      </div>
    )
  }
  if (records.length === 0) {
    return (
      <div className='text-muted-foreground flex h-28 items-center justify-center rounded-lg border text-sm'>
        {t('No login IP records')}
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap justify-end gap-2'>
        <Button
          size='sm'
          variant='destructive'
          disabled={!canBlock || pendingAction !== null}
          onClick={() => updateSelected(true)}
        >
          {pendingAction === 'block' ? (
            <Loader2 className='animate-spin' />
          ) : (
            <Ban />
          )}
          {t('Block selected')}
        </Button>
        <Button
          size='sm'
          variant='outline'
          disabled={!canUnblock || pendingAction !== null}
          onClick={() => updateSelected(false)}
        >
          {pendingAction === 'unblock' ? (
            <Loader2 className='animate-spin' />
          ) : (
            <ShieldCheck />
          )}
          {t('Unblock selected')}
        </Button>
      </div>
      <div className='overflow-hidden rounded-lg border'>
        {records.map((record) => (
          <LoginIPRow
            key={record.ip}
            record={record}
            checked={selectedIPs.includes(record.ip)}
            onCheckedChange={(checked) => toggleIP(record.ip, checked)}
          />
        ))}
      </div>
    </div>
  )
}

export function UserLoginAccessPanel(props: {
  userId?: number
  onUpdated: () => void
}) {
  const { t } = useTranslation()
  return (
    <Tabs defaultValue='devices' className='gap-4'>
      <TabsList variant='line'>
        <TabsTrigger value='devices'>{t('Devices')}</TabsTrigger>
        <TabsTrigger value='ips'>{t('Login IPs')}</TabsTrigger>
      </TabsList>
      <TabsContent value='devices'>
        <DevicePanel key={props.userId} userId={props.userId} />
      </TabsContent>
      <TabsContent value='ips'>
        <LoginIPPanel
          key={props.userId}
          userId={props.userId}
          onUpdated={props.onUpdated}
        />
      </TabsContent>
    </Tabs>
  )
}
