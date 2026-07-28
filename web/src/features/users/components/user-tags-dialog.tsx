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
import { Check, Pencil, Plus, Trash2, X } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { createUserTag, deleteUserTag, updateUserTag } from '../api'
import { canManageUserTag, USER_TAG_COLOR_PRESETS } from '../lib/user-tags'
import type { UserTag } from '../types'

type UserTagsDialogProps = {
  open: boolean
  tags: UserTag[]
  onOpenChange: (open: boolean) => void
  onChanged: () => Promise<void> | void
}

export function UserTagsDialog(props: UserTagsDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [color, setColor] = useState<string>(USER_TAG_COLOR_PRESETS[0])
  const [editingTagId, setEditingTagId] = useState<number | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<UserTag | null>(null)
  const [saving, setSaving] = useState(false)

  const resetEditor = () => {
    setName('')
    setColor(USER_TAG_COLOR_PRESETS[0])
    setEditingTagId(null)
  }

  const handleOpenChange = (open: boolean) => {
    if (!open) resetEditor()
    props.onOpenChange(open)
  }

  const handleEdit = (tag: UserTag) => {
    if (!canManageUserTag(tag)) return
    setEditingTagId(tag.id)
    setName(tag.name)
    setColor(tag.color)
  }

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!name.trim()) return
    setSaving(true)
    try {
      const result = editingTagId
        ? await updateUserTag(editingTagId, name.trim(), color)
        : await createUserTag(name.trim(), color)
      if (!result.success) {
        toast.error(result.message || t('Failed to save tag'))
        return
      }
      toast.success(t(editingTagId ? 'Tag updated' : 'Tag created'))
      resetEditor()
      await props.onChanged()
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to save tag')
      )
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget || !canManageUserTag(deleteTarget)) return
    try {
      const result = await deleteUserTag(deleteTarget.id)
      if (!result.success) {
        toast.error(result.message || t('Failed to delete tag'))
        return
      }
      if (editingTagId === deleteTarget.id) resetEditor()
      toast.success(t('Tag deleted'))
      await props.onChanged()
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to delete tag')
      )
    } finally {
      setDeleteTarget(null)
    }
  }

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={handleOpenChange}
        title={t('Manage tags')}
        contentHeight='min(34rem, calc(100vh - 14rem))'
        bodyClassName='space-y-5'
        footer={
          <Button variant='outline' onClick={() => handleOpenChange(false)}>
            {t('Close')}
          </Button>
        }
      >
        <form className='space-y-4' onSubmit={handleSave}>
          <div className='grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end'>
            <div className='space-y-1.5'>
              <Label htmlFor='user-tag-name'>{t('Tag name')}</Label>
              <Input
                id='user-tag-name'
                maxLength={32}
                value={name}
                onChange={(event) => setName(event.target.value)}
              />
            </div>
            <div className='flex gap-2'>
              {editingTagId && (
                <Button
                  type='button'
                  variant='outline'
                  size='icon'
                  aria-label={t('Cancel editing')}
                  onClick={resetEditor}
                >
                  <X />
                </Button>
              )}
              <Button type='submit' disabled={saving || !name.trim()}>
                {editingTagId ? <Check /> : <Plus />}
                {t(editingTagId ? 'Save tag' : 'Add tag')}
              </Button>
            </div>
          </div>

          <div className='space-y-1.5'>
            <Label htmlFor='user-tag-color'>{t('Tag color')}</Label>
            <div className='flex flex-wrap items-center gap-2'>
              {USER_TAG_COLOR_PRESETS.map((preset) => (
                <button
                  key={preset}
                  type='button'
                  className='ring-offset-background focus-visible:ring-ring size-7 rounded-full border-2 border-white shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-offset-2'
                  style={{ backgroundColor: preset }}
                  aria-label={t('Select color {{color}}', { color: preset })}
                  aria-pressed={color === preset}
                  onClick={() => setColor(preset)}
                />
              ))}
              <Input
                id='user-tag-color'
                type='color'
                className='h-8 w-12 cursor-pointer p-1'
                value={color}
                aria-label={t('Custom color')}
                onChange={(event) => setColor(event.target.value.toUpperCase())}
              />
              <span className='text-muted-foreground font-mono text-xs'>
                {color}
              </span>
            </div>
          </div>
        </form>

        <div className='border-t pt-4'>
          {props.tags.length === 0 ? (
            <div className='text-muted-foreground py-10 text-center text-sm'>
              {t('No tags')}
            </div>
          ) : (
            <div className='divide-y'>
              {props.tags.map((tag) => (
                <div
                  key={tag.id}
                  className='flex min-h-12 items-center gap-3 py-2'
                >
                  <span
                    className='h-8 w-1 shrink-0'
                    style={{ backgroundColor: tag.color }}
                  />
                  <span className='min-w-0 flex-1 truncate text-sm font-medium'>
                    {tag.built_in ? t(tag.name) : tag.name}
                  </span>
                  <span className='text-muted-foreground font-mono text-xs'>
                    {tag.color}
                  </span>
                  {!canManageUserTag(tag) ? (
                    <Badge variant='secondary'>{t('Built-in')}</Badge>
                  ) : (
                    <>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon-sm'
                              aria-label={t('Edit tag')}
                              onClick={() => handleEdit(tag)}
                            />
                          }
                        >
                          <Pencil />
                        </TooltipTrigger>
                        <TooltipContent>{t('Edit tag')}</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon-sm'
                              className='text-destructive hover:text-destructive'
                              aria-label={t('Delete tag')}
                              onClick={() => setDeleteTarget(tag)}
                            />
                          }
                        >
                          <Trash2 />
                        </TooltipTrigger>
                        <TooltipContent>{t('Delete tag')}</TooltipContent>
                      </Tooltip>
                    </>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('Delete tag')}
        desc={t('Delete tag {{name}}?', { name: deleteTarget?.name ?? '' })}
        confirmText={t('Delete')}
        destructive
        handleConfirm={handleDelete}
      />
    </>
  )
}
