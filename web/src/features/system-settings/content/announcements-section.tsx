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
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { MultiSelect } from '@/components/multi-select'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import type {
  Announcement,
  AnnouncementCondition,
  AnnouncementConditionGroup,
  AnnouncementConditionType,
} from '@/features/announcements/types'
import { getAdminPlans } from '@/features/subscriptions/api'
import dayjs from '@/lib/dayjs'

import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

type AnnouncementsSectionProps = { enabled: boolean; data: string }

const makeKey = () => crypto.randomUUID()
const newCondition = (): AnnouncementCondition => ({
  key: makeKey(),
  type: 'subscription_plan',
  operator: 'in',
  planIds: [],
})
const newGroup = (): AnnouncementConditionGroup => ({
  key: makeKey(),
  conditions: [newCondition()],
})
const emptyAnnouncement = (): Announcement => ({
  id: 0,
  title: '',
  content: '',
  status: 'draft',
  notificationMode: 'silent',
  audienceMode: 'all',
  conditionGroups: [],
})

function normalizeAnnouncement(item: Partial<Announcement>, index: number) {
  return {
    ...emptyAnnouncement(),
    ...item,
    id: item.id || index + 1,
    title: item.title || 'System Announcement',
    status: item.status || 'active',
    notificationMode: item.notificationMode || 'silent',
    audienceMode: item.audienceMode || 'all',
    startTime: item.startTime || item.publishDate || '',
    conditionGroups: (item.conditionGroups || []).map((group) => ({
      ...group,
      key: group.key || makeKey(),
      conditions: group.conditions.map((condition) => ({
        ...condition,
        key: condition.key || makeKey(),
      })),
    })),
  } as Announcement
}

export function AnnouncementsSection({
  enabled,
  data,
}: AnnouncementsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [announcements, setAnnouncements] = useState<Announcement[]>([])
  const [isEnabled, setIsEnabled] = useState(enabled)
  const [isEditorOpen, setIsEditorOpen] = useState(false)
  const [editing, setEditing] = useState<Announcement | null>(null)
  const [draft, setDraft] = useState<Announcement>(emptyAnnouncement)
  const plansQuery = useQuery({
    queryKey: ['admin-subscription-plans', 'announcement-editor'],
    queryFn: getAdminPlans,
  })
  const planOptions = (plansQuery.data?.data || []).map((record) => ({
    value: String(record.plan.id),
    label: record.plan.title,
  }))

  useEffect(() => {
    try {
      const parsed = JSON.parse(data || '[]') as Partial<Announcement>[]
      setAnnouncements(parsed.map(normalizeAnnouncement))
    } catch {
      setAnnouncements([])
    }
  }, [data])
  useEffect(() => setIsEnabled(enabled), [enabled])

  const openCreate = () => {
    setEditing(null)
    setDraft(emptyAnnouncement())
    setIsEditorOpen(true)
  }
  const openEdit = (announcement: Announcement) => {
    setEditing(announcement)
    setDraft(structuredClone(announcement))
    setIsEditorOpen(true)
  }
  const closeEditor = () => setIsEditorOpen(false)

  const setCondition = (
    groupKey: string,
    conditionKey: string,
    patch: Partial<AnnouncementCondition>
  ) => {
    setDraft((current) => ({
      ...current,
      conditionGroups: (current.conditionGroups || []).map((group) =>
        group.key === groupKey
          ? {
              ...group,
              conditions: group.conditions.map((condition) =>
                condition.key === conditionKey
                  ? { ...condition, ...patch }
                  : condition
              ),
            }
          : group
      ),
    }))
  }
  const removeCondition = (groupKey: string, conditionKey: string) => {
    setDraft((current) => ({
      ...current,
      conditionGroups: (current.conditionGroups || []).map((group) =>
        group.key === groupKey
          ? {
              ...group,
              conditions: group.conditions.filter(
                (condition) => condition.key !== conditionKey
              ),
            }
          : group
      ),
    }))
  }
  const persistAnnouncements = async (next: Announcement[]) => {
    await updateOption.mutateAsync({
      key: 'console_setting.announcements',
      value: JSON.stringify(next),
    })
    setAnnouncements(next)
  }

  const saveDraft = async () => {
    if (!draft.title.trim() || !draft.content.trim()) {
      toast.error(t('Title and content are required'))
      return
    }
    if (
      draft.endTime &&
      draft.startTime &&
      new Date(draft.endTime) <= new Date(draft.startTime)
    ) {
      toast.error(t('End time must be later than start time'))
      return
    }
    if (
      draft.audienceMode === 'conditions' &&
      !(draft.conditionGroups || []).some(
        (group) => group.conditions.length > 0
      )
    ) {
      toast.error(t('Add at least one display condition'))
      return
    }
    if (
      draft.audienceMode === 'conditions' &&
      (draft.conditionGroups || []).some((group) =>
        group.conditions.some(
          (condition) =>
            condition.type === 'subscription_plan' && !condition.planIds?.length
        )
      )
    ) {
      toast.error(t('Select at least one subscription plan'))
      return
    }
    const next = {
      ...draft,
      id:
        editing?.id || Math.max(0, ...announcements.map((item) => item.id)) + 1,
    }
    const nextAnnouncements = editing
      ? announcements.map((item) => (item.id === editing.id ? next : item))
      : [...announcements, next]
    try {
      await persistAnnouncements(nextAnnouncements)
      toast.success(t('Announcements saved successfully'))
      setEditing(null)
      setIsEditorOpen(false)
    } catch {
      toast.error(t('Failed to save announcements'))
    }
  }
  const handleToggle = async (checked: boolean) => {
    await updateOption.mutateAsync({
      key: 'console_setting.announcements_enabled',
      value: checked,
    })
    setIsEnabled(checked)
  }

  const statusMeta = {
    draft: { label: t('Draft'), variant: 'neutral' as const },
    active: { label: t('Displaying'), variant: 'success' as const },
    archived: { label: t('Archived'), variant: 'warning' as const },
  }
  const statusOptions = [
    { value: 'draft', label: t('Draft') },
    { value: 'active', label: t('Displaying') },
    { value: 'archived', label: t('Archived') },
  ]
  const notificationOptions = [
    { value: 'silent', label: t('Silent') },
    { value: 'popup', label: t('Popup') },
  ]
  const conditionTypeOptions = [
    { value: 'subscription_plan', label: t('Subscription plan') },
    { value: 'balance', label: t('Balance') },
  ]

  return (
    <SettingsSection title={t('Announcements')}>
      <div className='space-y-4'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex flex-wrap gap-2'>
            <Button size='sm' onClick={openCreate}>
              <Plus className='size-4' />
              {t('Create Announcement')}
            </Button>
          </div>
          <SettingsSwitchField
            checked={isEnabled}
            onCheckedChange={handleToggle}
            label={t('Enabled')}
            className='gap-2 py-0'
          />
        </div>
        <StaticDataTable
          data={announcements}
          getRowKey={(item) => item.id}
          emptyContent={t(
            'No announcements yet. Click "Create Announcement" to create one.'
          )}
          columns={[
            {
              id: 'title',
              header: t('Title'),
              cell: (item) => (
                <div className='max-w-sm'>
                  <div className='font-medium'>{item.title}</div>
                  <div className='text-muted-foreground truncate text-xs'>
                    {item.content}
                  </div>
                </div>
              ),
            },
            {
              id: 'status',
              header: t('Status'),
              cell: (item) => (
                <StatusBadge
                  label={statusMeta[item.status].label}
                  variant={statusMeta[item.status].variant}
                  copyable={false}
                />
              ),
            },
            {
              id: 'notification',
              header: t('Notification method'),
              cell: (item) =>
                item.notificationMode === 'popup' ? t('Popup') : t('Silent'),
            },
            {
              id: 'time',
              header: t('Effective time'),
              cell: (item) => (
                <div className='text-xs'>
                  {item.startTime
                    ? dayjs(item.startTime).format('YYYY-MM-DD HH:mm')
                    : t('Immediately')}
                  <br />
                  {item.endTime
                    ? dayjs(item.endTime).format('YYYY-MM-DD HH:mm')
                    : t('Permanent')}
                </div>
              ),
            },
            {
              id: 'audience',
              header: t('Display conditions'),
              cell: (item) =>
                item.audienceMode === 'all'
                  ? t('All users')
                  : t('By conditions'),
            },
            {
              id: 'actions',
              header: t('Actions'),
              cell: (item) => (
                <StaticRowActions
                  editLabel={t('Edit')}
                  deleteLabel={t('Delete')}
                  menuLabel={t('Open menu')}
                  onEdit={() => openEdit(item)}
                  onDelete={async () => {
                    try {
                      await persistAnnouncements(
                        announcements.filter((entry) => entry.id !== item.id)
                      )
                      toast.success(t('Deleted successfully'))
                    } catch {
                      toast.error(t('Failed to save announcements'))
                    }
                  }}
                />
              ),
            },
          ]}
        />
      </div>

      <Dialog
        open={isEditorOpen}
        onOpenChange={setIsEditorOpen}
        title={editing ? t('Edit Announcement') : t('Create Announcement')}
        contentClassName='sm:max-w-4xl'
        contentHeight='min(78vh, 52rem)'
        footer={
          <>
            <Button variant='outline' onClick={closeEditor}>
              {t('Cancel')}
            </Button>
            <Button disabled={updateOption.isPending} onClick={saveDraft}>
              {t('Save')}
            </Button>
          </>
        }
      >
        <div className='space-y-5'>
          <div className='space-y-2'>
            <Label>{t('Title')}</Label>
            <Input
              value={draft.title}
              onChange={(event) =>
                setDraft({ ...draft, title: event.target.value })
              }
            />
          </div>
          <div className='space-y-2'>
            <Label>{t('Content (supports Markdown)')}</Label>
            <Textarea
              rows={8}
              value={draft.content}
              onChange={(event) =>
                setDraft({ ...draft, content: event.target.value })
              }
            />
          </div>
          <div className='grid gap-4 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label>{t('Status')}</Label>
              <Select
                items={statusOptions}
                value={draft.status}
                onValueChange={(value) =>
                  setDraft({
                    ...draft,
                    status: value as Announcement['status'],
                  })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='draft'>{t('Draft')}</SelectItem>
                  <SelectItem value='active'>{t('Displaying')}</SelectItem>
                  <SelectItem value='archived'>{t('Archived')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className='space-y-2'>
              <Label>{t('Notification method')}</Label>
              <Select
                items={notificationOptions}
                value={draft.notificationMode}
                onValueChange={(value) =>
                  setDraft({
                    ...draft,
                    notificationMode: value as Announcement['notificationMode'],
                  })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='silent'>{t('Silent')}</SelectItem>
                  <SelectItem value='popup'>{t('Popup')}</SelectItem>
                </SelectContent>
              </Select>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Popup mode automatically shows this announcement to matching users.'
                )}
              </p>
            </div>
            <div className='space-y-2'>
              <Label>{t('Start time')}</Label>
              <DateTimePicker
                value={draft.startTime ? new Date(draft.startTime) : undefined}
                onChange={(date) =>
                  setDraft({ ...draft, startTime: date?.toISOString() || '' })
                }
              />
              <p className='text-muted-foreground text-xs'>
                {t('Leave empty to take effect immediately')}
              </p>
            </div>
            <div className='space-y-2'>
              <Label>{t('End time')}</Label>
              <DateTimePicker
                value={draft.endTime ? new Date(draft.endTime) : undefined}
                onChange={(date) =>
                  setDraft({ ...draft, endTime: date?.toISOString() || '' })
                }
              />
              <p className='text-muted-foreground text-xs'>
                {t('Leave empty to remain active permanently')}
              </p>
            </div>
          </div>
          <section className='bg-muted/20 space-y-5 rounded-md border p-4'>
            <div className='flex flex-wrap items-center justify-between gap-4'>
              <div>
                <h3 className='font-medium'>{t('Display conditions')}</h3>
                <p className='text-muted-foreground text-sm'>
                  {draft.audienceMode === 'all'
                    ? t('All users')
                    : t('By conditions')}
                </p>
              </div>
              <RadioGroup
                value={draft.audienceMode}
                onValueChange={(value) =>
                  setDraft({
                    ...draft,
                    audienceMode: value as Announcement['audienceMode'],
                    conditionGroups:
                      value === 'conditions' && !draft.conditionGroups?.length
                        ? [newGroup()]
                        : draft.conditionGroups,
                  })
                }
                className='flex gap-4'
              >
                <label className='flex items-center gap-2'>
                  <RadioGroupItem value='all' />
                  {t('All users')}
                </label>
                <label className='flex items-center gap-2'>
                  <RadioGroupItem value='conditions' />
                  {t('By conditions')}
                </label>
              </RadioGroup>
            </div>
            {draft.audienceMode === 'conditions' ? (
              <div className='space-y-4'>
                <div className='flex items-center justify-between'>
                  <span className='font-medium'>
                    OR ({draft.conditionGroups?.length || 0}/50)
                  </span>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() =>
                      setDraft({
                        ...draft,
                        conditionGroups: [
                          ...(draft.conditionGroups || []),
                          newGroup(),
                        ],
                      })
                    }
                  >
                    <Plus className='size-4' />
                    {t('Add OR condition group')}
                  </Button>
                </div>
                {(draft.conditionGroups || []).map((group, groupIndex) => (
                  <div
                    key={group.key}
                    className='bg-background space-y-3 rounded-md border p-4'
                  >
                    <div className='flex items-center justify-between'>
                      <div>
                        <span className='font-medium'>
                          {t('Condition')} #{groupIndex + 1}
                        </span>
                        <span className='text-muted-foreground ml-3 text-sm'>
                          AND ({group.conditions.length}/50)
                        </span>
                      </div>
                      <Button
                        variant='ghost'
                        size='sm'
                        onClick={() =>
                          setDraft({
                            ...draft,
                            conditionGroups: draft.conditionGroups?.filter(
                              (entry) => entry.key !== group.key
                            ),
                          })
                        }
                      >
                        <Trash2 className='size-4' />
                        {t('Delete')}
                      </Button>
                    </div>
                    {group.conditions.map((condition) => (
                      <div
                        key={condition.key}
                        className='bg-muted/30 grid items-end gap-3 rounded-md p-3 md:grid-cols-[1fr_1fr_2fr_auto]'
                      >
                        <div className='space-y-2'>
                          <Label>{t('Condition type')}</Label>
                          <Select
                            items={conditionTypeOptions}
                            value={condition.type}
                            onValueChange={(value) =>
                              setCondition(
                                group.key,
                                condition.key,
                                value === 'balance'
                                  ? {
                                      type: value as AnnouncementConditionType,
                                      operator: 'gte',
                                      value: 0,
                                      planIds: undefined,
                                    }
                                  : {
                                      type: value as AnnouncementConditionType,
                                      operator: 'in',
                                      planIds: [],
                                      value: undefined,
                                    }
                              )
                            }
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value='subscription_plan'>
                                {t('Subscription plan')}
                              </SelectItem>
                              <SelectItem value='balance'>
                                {t('Balance')}
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className='space-y-2'>
                          <Label>{t('Operator')}</Label>
                          <Select
                            items={
                              condition.type === 'subscription_plan'
                                ? [
                                    { value: 'in', label: t('Includes any') },
                                    {
                                      value: 'not_in',
                                      label: t('Excludes all'),
                                    },
                                  ]
                                : [
                                    { value: 'gte', label: '≥' },
                                    { value: 'lte', label: '≤' },
                                    { value: 'gt', label: '>' },
                                    { value: 'lt', label: '<' },
                                    { value: 'eq', label: '=' },
                                  ]
                            }
                            value={condition.operator}
                            onValueChange={(value) =>
                              setCondition(group.key, condition.key, {
                                operator:
                                  value as AnnouncementCondition['operator'],
                              })
                            }
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              {condition.type === 'subscription_plan' ? (
                                <>
                                  <SelectItem value='in'>
                                    {t('Includes any')}
                                  </SelectItem>
                                  <SelectItem value='not_in'>
                                    {t('Excludes all')}
                                  </SelectItem>
                                </>
                              ) : (
                                <>
                                  <SelectItem value='gte'>≥</SelectItem>
                                  <SelectItem value='lte'>≤</SelectItem>
                                  <SelectItem value='gt'>&gt;</SelectItem>
                                  <SelectItem value='lt'>&lt;</SelectItem>
                                  <SelectItem value='eq'>=</SelectItem>
                                </>
                              )}
                            </SelectContent>
                          </Select>
                        </div>
                        <div className='space-y-2'>
                          <Label>
                            {condition.type === 'subscription_plan'
                              ? t('Select plans')
                              : t('Balance threshold')}
                          </Label>
                          {condition.type === 'subscription_plan' ? (
                            <MultiSelect
                              options={planOptions}
                              selected={(condition.planIds || []).map(String)}
                              onChange={(values) =>
                                setCondition(group.key, condition.key, {
                                  planIds: values.map(Number),
                                })
                              }
                              placeholder={t('Select subscription plans')}
                            />
                          ) : (
                            <Input
                              type='number'
                              min={0}
                              value={condition.value || 0}
                              onChange={(event) =>
                                setCondition(group.key, condition.key, {
                                  value: Number(event.target.value),
                                })
                              }
                            />
                          )}
                        </div>
                        <Button
                          variant='ghost'
                          size='icon'
                          onClick={() =>
                            removeCondition(group.key, condition.key)
                          }
                          title={t('Delete')}
                        >
                          <Trash2 className='size-4' />
                        </Button>
                      </div>
                    ))}
                    <div className='flex justify-end'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          setDraft({
                            ...draft,
                            conditionGroups: draft.conditionGroups?.map(
                              (entry) =>
                                entry.key === group.key
                                  ? {
                                      ...entry,
                                      conditions: [
                                        ...entry.conditions,
                                        newCondition(),
                                      ],
                                    }
                                  : entry
                            ),
                          })
                        }
                      >
                        <Plus className='size-4' />
                        {t('Add AND condition')}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : null}
          </section>
        </div>
      </Dialog>
    </SettingsSection>
  )
}
