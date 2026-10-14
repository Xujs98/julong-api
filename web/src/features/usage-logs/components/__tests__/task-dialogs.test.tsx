import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import {
  createEvent,
  fireEvent,
  render,
  screen,
  within,
} from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { TASK_ACTIONS, TASK_ACTION_MAPPINGS } from '../../constants'
import type { TaskLog } from '../../types'
import { useTaskLogsColumns } from '../columns/task-logs-columns'
import { TaskDetailsDialog } from '../dialogs/task-details-dialog'
import { TaskRequestCell } from '../dialogs/task-request-dialog'
import { TaskResultCell } from '../dialogs/task-result-dialog'
import { UsageLogsProvider } from '../usage-logs-provider'
import { UsageLogsTable } from '../usage-logs-table'

const log: TaskLog = {
  id: 1,
  task_id: 'task_own',
  user_id: 7,
  platform: 'private-platform',
  action: 'text_to_video',
  status: 'SUCCESS',
  channel_id: 1,
  group: 'default',
  quota: 0,
  submit_time: 1,
  request_available: true,
}

function mount(ui: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>)
}

afterEach(() => vi.restoreAllMocks())

test('request preview separates prompt from parameters and preserves false', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        task_id: log.task_id,
        truncated: false,
        request_body: {
          prompt: 'A white dog by the sea',
          seconds: 8,
          face: false,
          images: ['[MEDIA]'],
        },
      },
    },
  })
  mount(<TaskRequestCell log={log} />)
  fireEvent.click(screen.getByRole('button', { name: 'View request' }))
  const dialog = await screen.findByRole('dialog')
  expect(
    await within(dialog).findByRole('heading', { name: 'Prompt' })
  ).toBeVisible()
  expect(within(dialog).getByText('A white dog by the sea')).toBeVisible()
  expect(
    within(dialog).getByRole('heading', { name: 'Other parameters' })
  ).toBeVisible()
  expect(dialog).toHaveTextContent('"face": false')
  expect(dialog).toHaveTextContent('[MEDIA]')
  expect(dialog).not.toHaveTextContent('"prompt":')
  fireEvent.click(within(dialog).getAllByRole('button', { name: 'Close' })[0])
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
})

test('task ID opens latest result and refresh retrieves changed state', async () => {
  const get = vi
    .spyOn(api, 'get')
    .mockResolvedValueOnce({
      data: {
        success: true,
        data: { task_id: log.task_id, status: 'IN_PROGRESS' },
      },
    })
    .mockResolvedValue({
      data: {
        success: true,
        data: {
          task_id: log.task_id,
          status: 'SUCCESS',
          result_url: '/local-video',
        },
      },
    })
  mount(<TaskResultCell taskId={log.task_id} />)
  expect(get).not.toHaveBeenCalled()
  fireEvent.click(screen.getByRole('button', { name: log.task_id }))
  const dialog = await screen.findByRole('dialog')
  expect(await within(dialog).findByText(/IN_PROGRESS/)).toBeVisible()
  fireEvent.click(within(dialog).getByRole('button', { name: 'Refresh' }))
  await within(dialog).findByText(/SUCCESS/)
  expect(dialog).toHaveTextContent('"result_url": "/local-video"')
})

function PlatformColumns(props: { admin: boolean }) {
  const columns = useTaskLogsColumns(props.admin, false)
  return (
    <div>
      {columns.some(
        (column) => 'accessorKey' in column && column.accessorKey === 'platform'
      )
        ? 'Platform visible'
        : 'Platform hidden'}
    </div>
  )
}

test('ordinary users have no platform column or detail row, admins retain both', () => {
  const view = mount(
    <>
      <PlatformColumns admin={false} />
      <TaskDetailsDialog
        log={log}
        isAdmin={false}
        isRoot={false}
        open
        onOpenChange={() => undefined}
      />
    </>
  )
  expect(screen.getByText('Platform hidden')).toBeInTheDocument()
  expect(screen.queryByText('private-platform')).not.toBeInTheDocument()
  view.unmount()
  mount(
    <>
      <PlatformColumns admin />
      <TaskDetailsDialog
        log={log}
        isAdmin
        isRoot={false}
        open
        onOpenChange={() => undefined}
      />
    </>
  )
  expect(screen.getByText('Platform visible')).toBeInTheDocument()
  expect(screen.getByText('private-platform')).toBeInTheDocument()
})

test('task type badges use distinct video colors and finished output stays pinned right', () => {
  function ColumnContract() {
    const columns = useTaskLogsColumns(true, false)
    const actionColumn = columns.find(
      (column) => 'accessorKey' in column && column.accessorKey === 'action'
    )
    const outputColumn = columns.at(-1)
    return (
      <div>
        <span>{actionColumn ? 'action-present' : 'action-missing'}</span>
        <span>{outputColumn?.id}</span>
        <span>{outputColumn?.meta?.pinned}</span>
      </div>
    )
  }

  mount(<ColumnContract />)
  expect(screen.getByText('action-present')).toBeInTheDocument()
  expect(screen.getByText('artifacts')).toBeInTheDocument()
  expect(screen.getByText('right')).toBeInTheDocument()
  expect(TASK_ACTION_MAPPINGS[TASK_ACTIONS.TEXT_GENERATE].variant).toBe('blue')
  expect(TASK_ACTION_MAPPINGS[TASK_ACTIONS.GENERATE].variant).toBe('violet')
  expect(TASK_ACTION_MAPPINGS[TASK_ACTIONS.FIRST_TAIL_GENERATE].variant).toBe(
    'cyan'
  )
})

test('a long task ID is contained, shows its full value on hover and retains the copy control', () => {
  const taskId = `task_${'long-id'.repeat(30)}`
  mount(<TaskResultCell taskId={taskId} />)
  const button = screen.getByRole('button', { name: taskId })
  expect(button).toHaveAttribute('title', taskId)
  expect(button).toHaveClass('min-w-0', 'overflow-hidden')
  expect(button.parentElement).toHaveClass('overflow-hidden', 'w-40')
  expect(button.querySelector('span')).toHaveClass('truncate')
})

function mountTaskLogTable() {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: url.startsWith('/api/task')
        ? {
            items: [
              log,
              { ...log, id: 2, task_id: 'task_pending', status: 'IN_PROGRESS' },
            ],
            total: 2,
          }
        : {},
    },
  }))
  const root = createRootRoute()
  const auth = createRoute({ getParentRoute: () => root, id: '_authenticated' })
  const logs = createRoute({
    getParentRoute: () => auth,
    path: '/usage-logs/$section',
    component: () => (
      <UsageLogsProvider>
        <UsageLogsTable logCategory='task' />
      </UsageLogsProvider>
    ),
    validateSearch: (search: Record<string, unknown>) => search,
  })
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([logs])]),
    history: createMemoryHistory({ initialEntries: ['/usage-logs/task'] }),
  })
  mount(<RouterProvider router={router} />)
}

test('task log custom rows keep the whole finished output column pinned with visible content', async () => {
  mountTaskLogTable()
  const video = await screen.findByRole('button', { name: 'View video' })
  expect(
    screen.getByRole('columnheader', { name: 'Finished output' })
  ).toHaveClass('sticky', 'right-0')
  expect(video.closest('td')).toHaveClass('sticky', 'right-0', 'bg-background')
  expect(video.closest('td')).toHaveClass('data-table-pinned-right')
  expect(
    screen.getByRole('columnheader', { name: 'Finished output' })
  ).toHaveClass('data-table-pinned-right')
  expect(screen.getByText('No content').closest('td')).toHaveClass(
    'sticky',
    'right-0',
    'bg-background'
  )
})

test('wheel over finished output scrolls the same table horizontally and releases native scrolling at boundaries', async () => {
  mountTaskLogTable()
  const video = await screen.findByRole('button', { name: 'View video' })
  const viewport = video.closest('table')?.parentElement
  expect(viewport).not.toBeNull()
  if (!viewport) throw new Error('Expected task table viewport')
  Object.defineProperties(viewport, {
    scrollWidth: { configurable: true, value: 2000 },
    clientWidth: { configurable: true, value: 500 },
    scrollLeft: { configurable: true, writable: true, value: 100 },
  })

  const down = createEvent.wheel(video, {
    deltaY: 80,
    bubbles: true,
    cancelable: true,
  })
  fireEvent(video, down)
  expect(viewport.scrollLeft).toBe(180)
  expect(down.defaultPrevented).toBe(true)

  const header = screen.getByRole('columnheader', { name: 'Finished output' })
  fireEvent.wheel(header, { deltaY: -40, bubbles: true, cancelable: true })
  expect(viewport.scrollLeft).toBe(140)

  const ordinaryCell = screen.getByRole('button', { name: 'task_own' })
  const ordinary = createEvent.wheel(ordinaryCell, {
    deltaY: 80,
    bubbles: true,
    cancelable: true,
  })
  fireEvent(ordinaryCell, ordinary)
  expect(viewport.scrollLeft).toBe(140)
  expect(ordinary.defaultPrevented).toBe(false)

  const zoom = createEvent.wheel(video, {
    deltaY: 80,
    ctrlKey: true,
    bubbles: true,
    cancelable: true,
  })
  fireEvent(video, zoom)
  expect(viewport.scrollLeft).toBe(140)
  expect(zoom.defaultPrevented).toBe(false)

  viewport.scrollLeft = 1500
  const end = createEvent.wheel(video, {
    deltaY: 80,
    bubbles: true,
    cancelable: true,
  })
  fireEvent(video, end)
  expect(viewport.scrollLeft).toBe(1500)
  expect(end.defaultPrevented).toBe(false)

  Object.defineProperty(viewport, 'scrollWidth', {
    configurable: true,
    value: 500,
  })
  viewport.scrollLeft = 0
  const noOverflow = createEvent.wheel(video, {
    deltaY: 80,
    bubbles: true,
    cancelable: true,
  })
  fireEvent(video, noOverflow)
  expect(viewport.scrollLeft).toBe(0)
  expect(noOverflow.defaultPrevented).toBe(false)
})
