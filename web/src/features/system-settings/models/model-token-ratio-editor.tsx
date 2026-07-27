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
import { ChevronDown, ChevronUp, Plus, Trash2 } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import {
  MAX_MODEL_TOKEN_ADJUSTMENT,
  createModelTokenAdjustmentDraft,
  modelTokenAdjustmentFromDraft,
  parseModelTokenAdjustmentMap,
  validateModelTokenAdjustmentDraft,
  type ModelTokenAdjustment,
  type ModelTokenAdjustmentDraft,
  type ModelTokenAdjustmentKey,
  type ModelTokenAdjustmentMap,
  type ModelTokenAdjustmentValidationError,
} from './model-token-ratio'

type ModelTokenRatioEditorProps = {
  value: string
  groupOptions: string[]
  onChange: (value: string) => void
}

type ModelTokenRatioRow = {
  userGroup: string
  billingGroup: string
  modelName: string
  adjustment: ModelTokenAdjustment
}

type GroupSelectProps = {
  options: string[]
  value: string
  placeholder: string
  onValueChange: (value: string) => void
  className?: string
  id?: string
}

const adjustmentFields: Array<{
  key: ModelTokenAdjustmentKey
  labelKey: string
}> = [
  { key: 'input', labelKey: 'Input' },
  { key: 'output', labelKey: 'Output' },
  { key: 'cache_read', labelKey: 'Cache Read' },
  { key: 'cache_creation', labelKey: 'Cache Creation' },
]

const sectionCardClassName =
  'relative shadow-sm ring-0 before:pointer-events-none before:absolute before:inset-0 before:rounded-xl before:border before:border-border/90'
const sectionHeaderClassName = 'border-b bg-muted/20'

function formatAdjustment(value?: number) {
  return value === undefined ? '-' : `+${value}`
}

function getValidationMessage(
  error: ModelTokenAdjustmentValidationError,
  t: (key: string) => string
) {
  if (error === 'billing_group_required') {
    return t('Billing group is required')
  }
  if (error === 'model_required') return t('Model name is required')
  if (error === 'adjustment_required') {
    return t('Enable at least one adjustment.')
  }
  return t('Enter an adjustment from 0 to 100.')
}

function GroupSelect(props: GroupSelectProps) {
  const options = useMemo(() => {
    if (props.value && !props.options.includes(props.value)) {
      return [props.value, ...props.options]
    }
    return props.options
  }, [props.options, props.value])

  return (
    <Select
      value={props.value === '' ? null : props.value}
      onValueChange={(value) => {
        if (typeof value === 'string') props.onValueChange(value)
      }}
    >
      <SelectTrigger id={props.id} className={props.className}>
        <SelectValue placeholder={props.placeholder} />
      </SelectTrigger>
      <SelectContent alignItemWithTrigger={false}>
        <SelectGroup>
          {options.map((option) => (
            <SelectItem key={option} value={option}>
              {option}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  )
}

type UserGroupSectionProps = {
  userGroup: string
  rules: Record<string, Record<string, ModelTokenAdjustment>>
  onAdd: (userGroup: string) => void
  onEdit: (row: ModelTokenRatioRow) => void
  onDelete: (row: ModelTokenRatioRow) => void
  onDeleteGroup: (userGroup: string) => void
}

function UserGroupSection(props: UserGroupSectionProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(true)
  const rows = useMemo<ModelTokenRatioRow[]>(() => {
    const result: ModelTokenRatioRow[] = []
    for (const [billingGroup, modelRules] of Object.entries(props.rules)) {
      for (const [modelName, adjustment] of Object.entries(modelRules)) {
        result.push({
          userGroup: props.userGroup,
          billingGroup,
          modelName,
          adjustment,
        })
      }
    }
    return result.sort((left, right) => {
      const groupOrder = left.billingGroup.localeCompare(right.billingGroup)
      return groupOrder || left.modelName.localeCompare(right.modelName)
    })
  }, [props.rules, props.userGroup])

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div className='rounded-md border'>
        <div className='flex flex-wrap items-center justify-between gap-2 p-3'>
          <div className='flex min-w-0 items-center gap-2'>
            <CollapsibleTrigger
              render={
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  className='h-7 w-7'
                  aria-label={props.userGroup}
                />
              }
            >
              {open ? (
                <ChevronUp className='h-4 w-4' />
              ) : (
                <ChevronDown className='h-4 w-4' />
              )}
            </CollapsibleTrigger>
            <span className='truncate font-semibold'>{props.userGroup}</span>
            <StatusBadge variant='neutral' copyable={false}>
              {rows.length} {t('rules')}
            </StatusBadge>
          </div>
          <div className='flex items-center gap-1'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => props.onAdd(props.userGroup)}
            >
              <Plus className='mr-2 h-4 w-4' />
              {t('Add ratio')}
            </Button>
            <Button
              type='button'
              variant='ghost'
              size='icon'
              className='text-destructive h-8 w-8'
              aria-label={`${t('Delete')} ${props.userGroup}`}
              onClick={() => props.onDeleteGroup(props.userGroup)}
            >
              <Trash2 className='h-4 w-4' />
            </Button>
          </div>
        </div>

        <CollapsibleContent>
          <div className='border-t p-3'>
            <StaticDataTable
              data={rows}
              getRowKey={(row) => `${row.billingGroup}:${row.modelName}`}
              emptyContent={t('No model token adjustments configured.')}
              columns={[
                {
                  id: 'billing-group',
                  header: t('Billing group'),
                  cellClassName: 'font-medium',
                  cell: (row) => row.billingGroup,
                },
                {
                  id: 'model',
                  header: t('Model name'),
                  cellClassName: 'font-medium',
                  cell: (row) => row.modelName,
                },
                ...adjustmentFields.map((field) => ({
                  id: field.key,
                  header: t(field.labelKey),
                  cell: (row: ModelTokenRatioRow) => (
                    <span className='font-mono text-xs tabular-nums'>
                      {formatAdjustment(row.adjustment[field.key])}
                    </span>
                  ),
                })),
                {
                  id: 'actions',
                  header: t('Actions'),
                  className: 'text-right',
                  cellClassName: 'text-right',
                  cell: (row) => (
                    <StaticRowActions
                      editLabel={t('Edit')}
                      deleteLabel={t('Delete')}
                      menuLabel={t('Open menu')}
                      onEdit={() => props.onEdit(row)}
                      onDelete={() => props.onDelete(row)}
                    />
                  ),
                },
              ]}
            />
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}

export function ModelTokenRatioEditor(props: ModelTokenRatioEditorProps) {
  const { t } = useTranslation()
  const onChange = props.onChange
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<ModelTokenRatioRow | null>(
    null
  )
  const [userGroup, setUserGroup] = useState('')
  const [billingGroup, setBillingGroup] = useState('')
  const [modelName, setModelName] = useState('')
  const [newUserGroup, setNewUserGroup] = useState('')
  const [draft, setDraft] = useState<ModelTokenAdjustmentDraft>(() =>
    createModelTokenAdjustmentDraft()
  )
  const [validationError, setValidationError] =
    useState<ModelTokenAdjustmentValidationError | null>(null)

  const ratioMap = useMemo(
    () => parseModelTokenAdjustmentMap(props.value),
    [props.value]
  )
  const userGroups = useMemo(
    () =>
      Object.entries(ratioMap).sort(([left], [right]) =>
        left.localeCompare(right)
      ),
    [ratioMap]
  )
  const newUserGroupOptions = useMemo(() => {
    const configuredGroups = new Set(Object.keys(ratioMap))
    return props.groupOptions.filter((group) => !configuredGroups.has(group))
  }, [props.groupOptions, ratioMap])

  const emitChange = useCallback(
    (map: ModelTokenAdjustmentMap) => {
      onChange(JSON.stringify(map, null, 2))
    },
    [onChange]
  )

  const addUserGroup = useCallback(() => {
    if (!newUserGroup || ratioMap[newUserGroup]) return
    emitChange({ ...ratioMap, [newUserGroup]: {} })
    setNewUserGroup('')
  }, [emitChange, newUserGroup, ratioMap])

  const deleteUserGroup = useCallback(
    (group: string) => {
      const map = parseModelTokenAdjustmentMap(props.value)
      delete map[group]
      emitChange(map)
    },
    [emitChange, props.value]
  )

  const openCreateDialog = useCallback((group: string) => {
    setEditingRule(null)
    setUserGroup(group)
    setBillingGroup('')
    setModelName('')
    setDraft(createModelTokenAdjustmentDraft())
    setValidationError(null)
    setDialogOpen(true)
  }, [])

  const openEditDialog = useCallback((row: ModelTokenRatioRow) => {
    setEditingRule(row)
    setUserGroup(row.userGroup)
    setBillingGroup(row.billingGroup)
    setModelName(row.modelName)
    setDraft(createModelTokenAdjustmentDraft(row.adjustment))
    setValidationError(null)
    setDialogOpen(true)
  }, [])

  const deleteRule = useCallback(
    (row: ModelTokenRatioRow) => {
      const map = parseModelTokenAdjustmentMap(props.value)
      const modelRules = map[row.userGroup]?.[row.billingGroup]
      if (!modelRules) return
      delete modelRules[row.modelName]
      if (Object.keys(modelRules).length === 0) {
        delete map[row.userGroup][row.billingGroup]
      }
      emitChange(map)
    },
    [emitChange, props.value]
  )

  const saveRule = useCallback(() => {
    const error = validateModelTokenAdjustmentDraft(
      billingGroup,
      modelName,
      draft
    )
    if (error) {
      setValidationError(error)
      return
    }

    const normalizedBillingGroup = billingGroup.trim()
    const normalizedModelName = modelName.trim()
    const map = parseModelTokenAdjustmentMap(props.value)
    if (editingRule) {
      const previousModels =
        map[editingRule.userGroup]?.[editingRule.billingGroup]
      if (previousModels) {
        delete previousModels[editingRule.modelName]
        if (Object.keys(previousModels).length === 0) {
          delete map[editingRule.userGroup][editingRule.billingGroup]
        }
      }
    }
    map[userGroup] ??= {}
    map[userGroup][normalizedBillingGroup] ??= {}
    map[userGroup][normalizedBillingGroup][normalizedModelName] =
      modelTokenAdjustmentFromDraft(draft)
    emitChange(map)
    setDialogOpen(false)
  }, [
    billingGroup,
    draft,
    editingRule,
    emitChange,
    modelName,
    props.value,
    userGroup,
  ])

  const updateDraft = useCallback(
    (
      key: ModelTokenAdjustmentKey,
      patch: Partial<ModelTokenAdjustmentDraft[ModelTokenAdjustmentKey]>
    ) => {
      setDraft((current) => ({
        ...current,
        [key]: { ...current[key], ...patch },
      }))
      setValidationError(null)
    },
    []
  )

  return (
    <Card className={sectionCardClassName}>
      <CardHeader className={sectionHeaderClassName}>
        <CardTitle>{t('Model special ratio rules')}</CardTitle>
        <CardDescription>
          {t(
            'Rules apply only when the user group, final billing group, and original model all match.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <p className='text-muted-foreground text-xs'>
            {t('For example, 0.1 bills 100 tokens as 110.')}
          </p>
          <div className='flex flex-wrap items-center gap-2'>
            <GroupSelect
              className='w-[200px]'
              options={newUserGroupOptions}
              value={newUserGroup}
              placeholder={t('Select a group')}
              onValueChange={setNewUserGroup}
            />
            <Button
              type='button'
              size='sm'
              disabled={!newUserGroup}
              onClick={addUserGroup}
            >
              <Plus className='mr-2 h-4 w-4' />
              {t('Add user group')}
            </Button>
          </div>
        </div>

        <div className='space-y-3'>
          {userGroups.length === 0 ? (
            <p className='text-muted-foreground py-6 text-center text-sm'>
              {t('No model token adjustments configured.')}
            </p>
          ) : (
            userGroups.map(([group, rules]) => (
              <UserGroupSection
                key={group}
                userGroup={group}
                rules={rules}
                onAdd={openCreateDialog}
                onEdit={openEditDialog}
                onDelete={deleteRule}
                onDeleteGroup={deleteUserGroup}
              />
            ))
          )}
        </div>
      </CardContent>

      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) setValidationError(null)
        }}
        title={editingRule ? t('Edit ratio') : t('Add ratio')}
        description={t(
          'Disabled adjustments are not applied. Enabled values are added to the matching token count.'
        )}
        contentHeight='auto'
        bodyClassName='space-y-4'
        footer={
          <>
            <Button variant='outline' onClick={() => setDialogOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button onClick={saveRule}>
              {editingRule ? t('Update') : t('Add')}
            </Button>
          </>
        }
      >
        <div className='space-y-4 py-2'>
          <div className='grid gap-4 sm:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='model-token-ratio-user-group'>
                {t('User group')}
              </Label>
              <Input
                id='model-token-ratio-user-group'
                value={userGroup}
                readOnly
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='model-token-ratio-billing-group'>
                {t('Billing group')}
              </Label>
              <GroupSelect
                id='model-token-ratio-billing-group'
                options={props.groupOptions}
                value={billingGroup}
                placeholder={t('Select billing group')}
                onValueChange={(value) => {
                  setBillingGroup(value)
                  setValidationError(null)
                }}
              />
            </div>
          </div>

          <div className='space-y-2'>
            <Label htmlFor='model-token-ratio-model'>{t('Model name')}</Label>
            <Input
              id='model-token-ratio-model'
              value={modelName}
              onChange={(event) => {
                setModelName(event.target.value)
                setValidationError(null)
              }}
              placeholder='gpt-5.6'
              autoComplete='off'
            />
          </div>

          <div className='divide-y rounded-md border'>
            {adjustmentFields.map((field) => {
              const switchId = `model-token-ratio-${field.key}`
              return (
                <div
                  key={field.key}
                  className='grid min-h-14 grid-cols-[minmax(0,1fr)_120px] items-center gap-4 px-3 py-2'
                >
                  <div className='flex min-w-0 items-center gap-2'>
                    <Switch
                      id={switchId}
                      checked={draft[field.key].enabled}
                      onCheckedChange={(enabled) =>
                        updateDraft(field.key, {
                          enabled,
                          value:
                            enabled && draft[field.key].value === ''
                              ? '0'
                              : draft[field.key].value,
                        })
                      }
                    />
                    <Label htmlFor={switchId}>{t(field.labelKey)}</Label>
                  </div>
                  <Input
                    type='number'
                    min={0}
                    max={MAX_MODEL_TOKEN_ADJUSTMENT}
                    step='0.01'
                    value={draft[field.key].value}
                    disabled={!draft[field.key].enabled}
                    onChange={(event) =>
                      updateDraft(field.key, { value: event.target.value })
                    }
                    aria-label={t(field.labelKey)}
                  />
                </div>
              )
            })}
          </div>

          {validationError && (
            <p className='text-destructive text-sm' role='alert'>
              {getValidationMessage(validationError, t)}
            </p>
          )}
        </div>
      </Dialog>
    </Card>
  )
}
