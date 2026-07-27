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
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import {
  appendDashboardReportSchedule,
  appendDashboardReportSendTime,
  MAX_REPORT_SCHEDULES,
  MAX_REPORT_SEND_TIMES,
  removeDashboardReportSchedule,
  removeDashboardReportSendTime,
} from './dashboard-report-schedule-state'
import type {
  DashboardReportEmailFrequency,
  DashboardReportEmailSchedule,
} from './email-templates-api'

type DashboardReportScheduleEditorProps = {
  disabled: boolean
  schedules: DashboardReportEmailSchedule[]
  onChange: (schedules: DashboardReportEmailSchedule[]) => void
}

export function DashboardReportScheduleEditor(
  props: DashboardReportScheduleEditorProps
) {
  const { t } = useTranslation()

  const updateSchedule = (
    scheduleIndex: number,
    patch: Partial<DashboardReportEmailSchedule>
  ) => {
    props.onChange(
      props.schedules.map((schedule, currentIndex) =>
        currentIndex === scheduleIndex ? { ...schedule, ...patch } : schedule
      )
    )
  }

  const updateSendTime = (
    scheduleIndex: number,
    sendTimeIndex: number,
    value: string
  ) => {
    const schedule = props.schedules[scheduleIndex]
    updateSchedule(scheduleIndex, {
      send_times: schedule.send_times.map((sendTime, currentIndex) =>
        currentIndex === sendTimeIndex ? value : sendTime
      ),
    })
  }

  return (
    <div className='grid gap-4'>
      {props.schedules.map((schedule, scheduleIndex) => (
        <div
          key={schedule.id}
          className='grid gap-4 border-t pt-4 first:border-t-0 first:pt-0'
        >
          <div className='flex min-h-7 items-center justify-between gap-3'>
            <span className='text-sm font-semibold'>
              {t('Report condition {{number}}', {
                number: scheduleIndex + 1,
              })}
            </span>
            {props.schedules.length > 1 && (
              <Button
                type='button'
                size='icon-sm'
                variant='ghost'
                className='text-destructive'
                disabled={props.disabled}
                title={t('Remove report condition')}
                aria-label={t('Remove report condition')}
                onClick={() =>
                  props.onChange(
                    removeDashboardReportSchedule(
                      props.schedules,
                      scheduleIndex
                    )
                  )
                }
              >
                <Trash2 className='size-4' />
              </Button>
            )}
          </div>

          <div className='grid gap-4 md:grid-cols-2'>
            <div className='space-y-1.5'>
              <Label htmlFor={`dashboard-report-frequency-${scheduleIndex}`}>
                {t('Report frequency')}
              </Label>
              <NativeSelect
                id={`dashboard-report-frequency-${scheduleIndex}`}
                className='w-full'
                disabled={props.disabled}
                value={schedule.frequency}
                onChange={(event) =>
                  updateSchedule(scheduleIndex, {
                    frequency: event.target
                      .value as DashboardReportEmailFrequency,
                  })
                }
              >
                <NativeSelectOption value='daily'>
                  {t('Daily')}
                </NativeSelectOption>
                <NativeSelectOption value='weekly'>
                  {t('Weekly')}
                </NativeSelectOption>
                <NativeSelectOption value='monthly'>
                  {t('Monthly')}
                </NativeSelectOption>
              </NativeSelect>
            </div>

            {schedule.frequency === 'weekly' && (
              <div className='space-y-1.5'>
                <Label htmlFor={`dashboard-report-weekday-${scheduleIndex}`}>
                  {t('Weekday')}
                </Label>
                <NativeSelect
                  id={`dashboard-report-weekday-${scheduleIndex}`}
                  className='w-full'
                  disabled={props.disabled}
                  value={String(schedule.weekday)}
                  onChange={(event) =>
                    updateSchedule(scheduleIndex, {
                      weekday: Number.parseInt(event.target.value, 10),
                    })
                  }
                >
                  <NativeSelectOption value='1'>
                    {t('Monday')}
                  </NativeSelectOption>
                  <NativeSelectOption value='2'>
                    {t('Tuesday')}
                  </NativeSelectOption>
                  <NativeSelectOption value='3'>
                    {t('Wednesday')}
                  </NativeSelectOption>
                  <NativeSelectOption value='4'>
                    {t('Thursday')}
                  </NativeSelectOption>
                  <NativeSelectOption value='5'>
                    {t('Friday')}
                  </NativeSelectOption>
                  <NativeSelectOption value='6'>
                    {t('Saturday')}
                  </NativeSelectOption>
                  <NativeSelectOption value='7'>
                    {t('Sunday')}
                  </NativeSelectOption>
                </NativeSelect>
              </div>
            )}

            {schedule.frequency === 'monthly' && (
              <div className='space-y-1.5'>
                <Label htmlFor={`dashboard-report-month-day-${scheduleIndex}`}>
                  {t('Day of month')}
                </Label>
                <Input
                  id={`dashboard-report-month-day-${scheduleIndex}`}
                  type='number'
                  min={1}
                  max={31}
                  disabled={props.disabled}
                  value={schedule.month_day}
                  onChange={(event) =>
                    updateSchedule(scheduleIndex, {
                      month_day: Math.min(
                        31,
                        Math.max(
                          1,
                          Number.parseInt(event.target.value, 10) || 1
                        )
                      ),
                    })
                  }
                />
              </div>
            )}
          </div>

          <div className='space-y-2'>
            <div className='flex items-center justify-between gap-3'>
              <Label>{t('Send times')}</Label>
              <Button
                type='button'
                size='icon-sm'
                variant='outline'
                disabled={
                  props.disabled ||
                  schedule.send_times.length >= MAX_REPORT_SEND_TIMES
                }
                title={t('Add send time')}
                aria-label={t('Add send time')}
                onClick={() =>
                  props.onChange(
                    appendDashboardReportSendTime(
                      props.schedules,
                      scheduleIndex
                    )
                  )
                }
              >
                <Plus className='size-4' />
              </Button>
            </div>
            <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-3'>
              {schedule.send_times.map((sendTime, sendTimeIndex) => (
                <div
                  key={`${schedule.id}-${sendTime}`}
                  className='flex min-w-0 items-center gap-2'
                >
                  <Input
                    type='time'
                    className='min-w-0'
                    disabled={props.disabled}
                    value={sendTime}
                    aria-label={t('Send time {{number}}', {
                      number: sendTimeIndex + 1,
                    })}
                    onChange={(event) =>
                      updateSendTime(
                        scheduleIndex,
                        sendTimeIndex,
                        event.target.value
                      )
                    }
                  />
                  {schedule.send_times.length > 1 && (
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='ghost'
                      className='text-destructive'
                      disabled={props.disabled}
                      title={t('Remove send time')}
                      aria-label={t('Remove send time')}
                      onClick={() =>
                        props.onChange(
                          removeDashboardReportSendTime(
                            props.schedules,
                            scheduleIndex,
                            sendTimeIndex
                          )
                        )
                      }
                    >
                      <Trash2 className='size-4' />
                    </Button>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      ))}

      <Button
        type='button'
        size='sm'
        variant='outline'
        className='w-fit'
        disabled={
          props.disabled || props.schedules.length >= MAX_REPORT_SCHEDULES
        }
        onClick={() =>
          props.onChange(appendDashboardReportSchedule(props.schedules))
        }
      >
        <Plus className='size-4' />
        {t('Add report condition')}
      </Button>
    </div>
  )
}
