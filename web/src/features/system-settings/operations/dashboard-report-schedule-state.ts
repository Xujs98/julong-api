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
import type { DashboardReportEmailSchedule } from './email-templates-api'

export const MAX_REPORT_SCHEDULES = 20
export const MAX_REPORT_SEND_TIMES = 12

export function createDashboardReportSchedule(
  id: string = crypto.randomUUID()
): DashboardReportEmailSchedule {
  return {
    id,
    frequency: 'daily',
    send_times: ['08:00'],
    weekday: 1,
    month_day: 1,
  }
}

export function getNextReportSendTime(sendTimes: string[]) {
  for (const candidate of ['08:00', '12:00', '18:00', '23:00']) {
    if (!sendTimes.includes(candidate)) return candidate
  }

  const lastTime = sendTimes.at(-1) ?? '08:00'
  const hour = Number.parseInt(lastTime.slice(0, 2), 10)
  for (let offset = 1; offset <= 24; offset++) {
    const candidate = `${String((hour + offset) % 24).padStart(2, '0')}:00`
    if (!sendTimes.includes(candidate)) return candidate
  }

  return '00:00'
}

export function appendDashboardReportSchedule(
  schedules: DashboardReportEmailSchedule[],
  id: string = crypto.randomUUID()
) {
  return [...schedules, createDashboardReportSchedule(id)]
}

export function removeDashboardReportSchedule(
  schedules: DashboardReportEmailSchedule[],
  scheduleIndex: number
) {
  return schedules.filter((_, currentIndex) => currentIndex !== scheduleIndex)
}

export function appendDashboardReportSendTime(
  schedules: DashboardReportEmailSchedule[],
  scheduleIndex: number
) {
  return schedules.map((schedule, currentIndex) =>
    currentIndex === scheduleIndex
      ? {
          ...schedule,
          send_times: [
            ...schedule.send_times,
            getNextReportSendTime(schedule.send_times),
          ],
        }
      : schedule
  )
}

export function removeDashboardReportSendTime(
  schedules: DashboardReportEmailSchedule[],
  scheduleIndex: number,
  sendTimeIndex: number
) {
  return schedules.map((schedule, currentIndex) =>
    currentIndex === scheduleIndex
      ? {
          ...schedule,
          send_times: schedule.send_times.filter(
            (_, currentSendTimeIndex) => currentSendTimeIndex !== sendTimeIndex
          ),
        }
      : schedule
  )
}
