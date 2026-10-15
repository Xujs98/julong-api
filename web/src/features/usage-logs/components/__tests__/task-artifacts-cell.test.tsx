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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import type { TaskLog } from '../../types'
import { TaskArtifactsCell } from '../task-artifacts'

function taskLog(overrides: Partial<TaskLog> = {}): TaskLog {
  return {
    id: 1,
    user_id: 7,
    platform: 'alibaba',
    task_id: 'task-public',
    action: 'text_to_image',
    channel_id: 357,
    group: 'default',
    quota: 25000,
    submit_time: 1,
    status: 'SUCCESS',
    admin_info: {
      task_plugin: {
        key: 'alibaba',
        name: 'Alibaba Bailian',
        version: '1.4.0',
      },
    },
    ...overrides,
  }
}

function renderCell(log: TaskLog) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <TaskArtifactsCell log={log} />
    </QueryClientProvider>
  )
}

describe('task artifacts cell', () => {
  test('given a synchronous result the gateway did not retain, a hint replaces the artifacts button', () => {
    renderCell(taskLog({ result_discarded: true }))

    expect(screen.getByText('Result not retained')).toBeVisible()
    expect(
      screen.queryByRole('button', { name: /Finished output/ })
    ).not.toBeInTheDocument()
  })

  test('given a retained successful plugin task, the artifacts button is offered', () => {
    renderCell(taskLog())

    expect(
      screen.getByRole('button', { name: /Finished output/ })
    ).toBeVisible()
    expect(screen.queryByText('Result not retained')).not.toBeInTheDocument()
  })

  test('given an unfinished task, neither the button nor the hint is shown', () => {
    renderCell(taskLog({ status: 'IN_PROGRESS', result_discarded: true }))

    expect(screen.getByText('No content')).toBeVisible()
    expect(screen.queryByText('Result not retained')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /Finished output/ })
    ).not.toBeInTheDocument()
  })

  test('given a task with no retained output, a visible placeholder occupies the finished output cell', () => {
    renderCell(taskLog({ status: 'IN_PROGRESS', result_discarded: false }))

    expect(screen.getByText('No content')).toBeVisible()
  })
})

test('successful video tasks offer a view video button', () => {
  renderCell(taskLog({ action: 'text_to_video' }))
  expect(screen.getByRole('button', { name: 'View video' })).toBeVisible()
})

test('video downloads use a random mp4 filename and download route', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        task_id: 'task-public',
        artifacts: [
          {
            key: 'video-main',
            type: 'video',
            mime_type: 'video/mp4',
            content_url: [
              'https://gateway.example/v1/tasks/task-public/artifacts/video-main/content?access=',
              'a'.repeat(43),
            ].join(''),
          },
        ],
      },
    },
  })

  renderCell(taskLog({ action: 'text_to_video' }))
  fireEvent.click(screen.getByRole('button', { name: 'View video' }))

  await screen.findByText('video/mp4')
  const download = screen.getByRole('button', { name: 'Download' })
  expect(download).toHaveAttribute('download')
  expect(download.getAttribute('download')).toMatch(/^[A-Za-z0-9-]+\.mp4$/)
  expect(download.getAttribute('href')).toContain('download=video')
  expect(download.getAttribute('href')).toContain('access=')
})

test('failed tasks show a red failure button that opens the reason', async () => {
  renderCell(
    taskLog({ status: 'FAILURE', fail_reason: 'Unsupported reference image' })
  )
  const button = screen.getByRole('button', { name: 'Failure reason' })
  expect(button).toHaveClass('text-destructive')
  fireEvent.click(button)
  expect(await screen.findByRole('dialog')).toHaveTextContent(
    'Unsupported reference image'
  )
})

test('a video playback failure replaces the view button with a red failure button', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        task_id: 'task-public',
        artifacts: [
          {
            key: 'video',
            type: 'video',
            mime_type: 'video/mp4',
            content_url: `https://gateway.example/v1/tasks/task-public/artifacts/video/content?access=${'a'.repeat(43)}`,
          },
        ],
      },
    },
  })
  renderCell(taskLog({ action: 'text_to_video' }))
  fireEvent.click(screen.getByRole('button', { name: 'View video' }))
  const video = await screen.findByText('video/mp4')
  const media = video.closest('[data-slot="card"]')?.querySelector('video')
  expect(media).not.toBeNull()
  if (!media) throw new Error('Video preview was not rendered')
  fireEvent.error(media)
  expect(
    await screen.findByRole('dialog', { name: 'Failure reason' })
  ).toHaveTextContent('Media preview failed. Please try again.')
  fireEvent.click(screen.getAllByRole('button', { name: 'Close' })[0])
  expect(screen.getByRole('button', { name: 'Failure reason' })).toHaveClass(
    'text-destructive'
  )
})
