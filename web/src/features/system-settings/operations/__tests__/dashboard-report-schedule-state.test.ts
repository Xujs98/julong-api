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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  appendDashboardReportSchedule,
  appendDashboardReportSendTime,
  removeDashboardReportSchedule,
  removeDashboardReportSendTime,
} from '../dashboard-report-schedule-state.ts'
import type { DashboardReportEmailSchedule } from '../email-templates-api.ts'

const dailySchedule: DashboardReportEmailSchedule = {
  id: 'daily-report',
  frequency: 'daily',
  send_times: ['12:00'],
  weekday: 1,
  month_day: 1,
}

describe('dashboard report schedule state', () => {
  test('adds independent report conditions with default schedule values', () => {
    const schedules = appendDashboardReportSchedule(
      [dailySchedule],
      'weekly-report'
    )

    assert.equal(schedules.length, 2)
    assert.deepEqual(schedules[1], {
      id: 'weekly-report',
      frequency: 'daily',
      send_times: ['08:00'],
      weekday: 1,
      month_day: 1,
    })
    assert.deepEqual(schedules[0], dailySchedule)
  })

  test('adds a distinct send time to only the selected condition', () => {
    const weeklySchedule: DashboardReportEmailSchedule = {
      ...dailySchedule,
      id: 'weekly-report',
      frequency: 'weekly',
      send_times: ['08:00', '12:00'],
    }

    const schedules = appendDashboardReportSendTime(
      [dailySchedule, weeklySchedule],
      1
    )

    assert.deepEqual(schedules[0].send_times, ['12:00'])
    assert.deepEqual(schedules[1].send_times, ['08:00', '12:00', '18:00'])
  })

  test('removes only the selected condition or send time', () => {
    const secondSchedule: DashboardReportEmailSchedule = {
      ...dailySchedule,
      id: 'second-report',
      send_times: ['08:00', '23:00'],
    }

    const schedules = removeDashboardReportSendTime(
      [dailySchedule, secondSchedule],
      1,
      0
    )
    const remaining = removeDashboardReportSchedule(schedules, 0)

    assert.deepEqual(remaining, [
      {
        ...secondSchedule,
        send_times: ['23:00'],
      },
    ])
  })
})
