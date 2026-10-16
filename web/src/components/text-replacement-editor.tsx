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
import { Code, Plus, Table, Trash2 } from 'lucide-react'
import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ComponentProps,
} from 'react'
import { useTranslation } from 'react-i18next'

import { JsonCodeEditor } from '@/components/json-code-editor'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

export type TextReplacementRule = {
  from: string
  to: string
}

export type TextReplacementConfig = {
  request?: TextReplacementRule[]
  response?: TextReplacementRule[]
}

type ReplacementDirection = 'request' | 'response'
type ReplacementRow = TextReplacementRule & { id: string }

type TextReplacementEditorProps = Omit<
  ComponentProps<'div'>,
  'name' | 'onBlur' | 'onChange'
> & {
  value: string
  onChange: (value: string) => void
  name?: string
  onBlur?: () => void
  textareaRef?: (element: HTMLTextAreaElement | null) => void
  disabled?: boolean
  placeholder?: string
  ariaLabel?: string
  'aria-describedby'?: string
  'aria-invalid'?: boolean | 'grammar' | 'false' | 'spelling' | 'true'
  'data-form-root'?: string
}

type ReplacementRows = Record<ReplacementDirection, ReplacementRow[]>

const EMPTY_ROWS: ReplacementRows = {
  request: [],
  response: [],
}

let nextRowId = 0

function createRowId(direction: ReplacementDirection): string {
  nextRowId += 1
  return `${direction}-${nextRowId}`
}

function createRow(direction: ReplacementDirection): ReplacementRow {
  return { id: createRowId(direction), from: '', to: '' }
}

function parseConfig(value: string): {
  rows: ReplacementRows
  error: string
} {
  if (!value.trim()) return { rows: EMPTY_ROWS, error: '' }

  try {
    const parsed: unknown = JSON.parse(value)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { rows: EMPTY_ROWS, error: 'JSON must be an object' }
    }

    const record = parsed as Record<string, unknown>
    const unknownKey = Object.keys(record).find(
      (key) => key !== 'request' && key !== 'response'
    )
    if (unknownKey) {
      return { rows: EMPTY_ROWS, error: 'Invalid JSON format' }
    }
    const rows = { request: [], response: [] } as ReplacementRows
    for (const direction of ['request', 'response'] as const) {
      const valueForDirection = record[direction]
      if (valueForDirection === undefined) continue
      if (!Array.isArray(valueForDirection)) {
        return {
          rows: EMPTY_ROWS,
          error: 'Invalid JSON format',
        }
      }
      for (const item of valueForDirection) {
        if (
          !item ||
          typeof item !== 'object' ||
          Array.isArray(item) ||
          typeof (item as Record<string, unknown>).from !== 'string' ||
          typeof (item as Record<string, unknown>).to !== 'string'
        ) {
          return {
            rows: EMPTY_ROWS,
            error: 'Invalid JSON format',
          }
        }
        rows[direction].push({
          id: createRowId(direction),
          from: (item as Record<string, string>).from,
          to: (item as Record<string, string>).to,
        })
      }
    }
    return { rows, error: '' }
  } catch {
    return { rows: EMPTY_ROWS, error: 'JSON format error' }
  }
}

function serializeRows(rows: ReplacementRows): string {
  const config: TextReplacementConfig = {}
  for (const direction of ['request', 'response'] as const) {
    const entries = rows[direction]
      .filter((rule) => rule.from.trim())
      .map((rule) => ({ from: rule.from.trim(), to: rule.to }))
    if (entries.length > 0) config[direction] = entries
  }
  if (!config.request && !config.response) return ''
  return JSON.stringify(config, null, 2)
}

function cloneRows(rows: ReplacementRows): ReplacementRows {
  return {
    request: rows.request.map((rule) => ({ ...rule })),
    response: rows.response.map((rule) => ({ ...rule })),
  }
}

export function TextReplacementEditor(props: TextReplacementEditorProps) {
  const { t } = useTranslation()
  const editorId = useId()
  const latestValueRef = useRef(props.value)
  const [mode, setMode] = useState<'visual' | 'json'>('visual')
  const [rows, setRows] = useState<ReplacementRows>(
    () => parseConfig(props.value).rows
  )
  const [jsonValue, setJsonValue] = useState(props.value)
  const [jsonError, setJsonError] = useState(
    () => parseConfig(props.value).error
  )

  useEffect(() => {
    if (props.value === latestValueRef.current) return
    latestValueRef.current = props.value
    setJsonValue(props.value)
    const parsed = parseConfig(props.value)
    setRows(parsed.rows)
    setJsonError(parsed.error)
  }, [props.value])

  const directionLabels = useMemo(
    () => ({
      request: t('Request'),
      response: t('Response'),
    }),
    [t]
  )

  const emitRows = (nextRows: ReplacementRows) => {
    setRows(nextRows)
    const nextValue = serializeRows(nextRows)
    latestValueRef.current = nextValue
    setJsonValue(nextValue)
    setJsonError('')
    props.onChange(nextValue)
  }

  const handleRowChange = (
    direction: ReplacementDirection,
    index: number,
    field: keyof TextReplacementRule,
    value: string
  ) => {
    const nextRows = cloneRows(rows)
    nextRows[direction][index] = {
      ...nextRows[direction][index],
      [field]: value,
    }
    emitRows(nextRows)
  }

  const handleAddRow = (direction: ReplacementDirection) => {
    const nextRows = cloneRows(rows)
    nextRows[direction].push(createRow(direction))
    setRows(nextRows)
  }

  const handleDeleteRow = (direction: ReplacementDirection, index: number) => {
    const nextRows = cloneRows(rows)
    nextRows[direction].splice(index, 1)
    emitRows(nextRows)
  }

  const handleJsonChange = (nextValue: string) => {
    latestValueRef.current = nextValue
    setJsonValue(nextValue)
    props.onChange(nextValue)
    const parsed = parseConfig(nextValue)
    setJsonError(parsed.error)
    if (!parsed.error) setRows(parsed.rows)
  }

  const handleModeChange = (nextMode: string) => {
    if (nextMode !== 'visual' && nextMode !== 'json') return
    if (nextMode === 'json') {
      const nextValue = serializeRows(rows)
      latestValueRef.current = nextValue
      setJsonValue(nextValue)
      setJsonError('')
      props.onChange(nextValue)
      setMode('json')
      return
    }

    const parsed = parseConfig(jsonValue)
    if (parsed.error) {
      setJsonError(parsed.error)
      return
    }
    setRows(parsed.rows)
    setJsonError('')
    setMode('visual')
  }

  const renderDirection = (direction: ReplacementDirection) => (
    <section
      key={direction}
      className='space-y-3 rounded-lg border p-3'
      aria-labelledby={`${editorId}-${direction}-heading`}
    >
      <div className='flex items-center justify-between gap-2'>
        <div>
          <h3
            id={`${editorId}-${direction}-heading`}
            className='text-sm font-medium'
          >
            {directionLabels[direction]}
          </h3>
          <p className='text-muted-foreground text-xs'>
            {t(
              direction === 'request'
                ? 'Rules applied before sending the request upstream.'
                : 'Rules applied before returning the response to the client.'
            )}
          </p>
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => handleAddRow(direction)}
          disabled={props.disabled}
        >
          <Plus className='mr-2 h-4 w-4' aria-hidden='true' />
          {t('Add Row')}
        </Button>
      </div>

      {rows[direction].length > 0 ? (
        <div className='space-y-2'>
          <div className='grid grid-cols-[1fr_1fr_auto] gap-2 text-sm font-medium'>
            <div>{t('Replace text')}</div>
            <div>{t('Replace with')}</div>
            <div className='w-10' />
          </div>
          {rows[direction].map((row, index) => (
            <div key={row.id} className='grid grid-cols-[1fr_1fr_auto] gap-2'>
              <Input
                value={row.from}
                onChange={(event) =>
                  handleRowChange(direction, index, 'from', event.target.value)
                }
                onBlur={props.onBlur}
                placeholder={t('Text to replace')}
                disabled={props.disabled}
                aria-label={`${directionLabels[direction]} ${t('Replace text')}`}
              />
              <Input
                value={row.to}
                onChange={(event) =>
                  handleRowChange(direction, index, 'to', event.target.value)
                }
                onBlur={props.onBlur}
                placeholder={t('Replace with')}
                disabled={props.disabled}
                aria-label={`${directionLabels[direction]} ${t('Replace with')}`}
              />
              <Button
                type='button'
                variant='ghost'
                size='icon'
                onClick={() => handleDeleteRow(direction, index)}
                disabled={props.disabled}
                className='h-10 w-10'
                aria-label={`${directionLabels[direction]} ${t('Delete')}`}
              >
                <Trash2 className='h-4 w-4' aria-hidden='true' />
              </Button>
            </div>
          ))}
        </div>
      ) : (
        <div className='text-muted-foreground flex h-20 items-center justify-center rounded-md border border-dashed text-sm'>
          {t('No mappings configured. Click "Add Row" to get started.')}
        </div>
      )}
    </section>
  )

  return (
    <div className={cn('space-y-3', props.className)}>
      <Tabs value={mode} onValueChange={handleModeChange}>
        <TabsList>
          <TabsTrigger value='visual'>
            <Table className='h-4 w-4' aria-hidden='true' />
            {t('Visual')}
          </TabsTrigger>
          <TabsTrigger value='json'>
            <Code className='h-4 w-4' aria-hidden='true' />
            {t('JSON')}
          </TabsTrigger>
        </TabsList>

        {jsonError && (
          <Alert variant='destructive'>
            <AlertDescription>{t(jsonError)}</AlertDescription>
          </Alert>
        )}

        <TabsContent value='visual' className='space-y-3'>
          {renderDirection('request')}
          {renderDirection('response')}
        </TabsContent>
        <TabsContent value='json'>
          <JsonCodeEditor
            value={jsonValue}
            onChange={handleJsonChange}
            name={props.name}
            onBlur={props.onBlur}
            textareaRef={props.textareaRef}
            disabled={props.disabled}
            ariaLabel={props.ariaLabel ?? t('Text Replacement Rules')}
            aria-describedby={props['aria-describedby']}
            aria-invalid={props['aria-invalid'] ?? Boolean(jsonError)}
            data-form-root={props['data-form-root']}
            placeholder={
              props.placeholder ??
              t(
                '{"request":[{"from":"source","to":"upstream"}],"response":[{"from":"upstream","to":"source"}]}'
              )
            }
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}
