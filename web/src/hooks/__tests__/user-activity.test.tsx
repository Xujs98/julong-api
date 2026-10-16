import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, cleanup, fireEvent, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { useUserActivity } from '../use-user-activity'

afterEach(() => {
  cleanup()
  useAuthStore.getState().auth.reset()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

function showActivity() {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false, gcTime: 0 } },
  })
  return renderHook(useUserActivity, {
    wrapper: (props: { children: ReactNode }) => (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    ),
  })
}

it('reports actual interactions, never idle time, focus, hidden tabs or unmounted pages', async () => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-10-16T10:00:00Z'))
  const visibility = vi
    .spyOn(document, 'visibilityState', 'get')
    .mockReturnValue('visible')
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  useAuthStore
    .getState()
    .auth.setUser({ id: 5, username: 'active-user', role: 1 })
  const { unmount } = showActivity()
  await act(async () => {
    vi.advanceTimersByTime(6 * 60_000)
  })
  expect(post).not.toHaveBeenCalled()
  await act(async () => {
    fireEvent.pointerDown(document.body)
  })
  expect(post).toHaveBeenCalledExactlyOnceWith('/api/user/self/activity')
  await act(async () => {
    fireEvent.wheel(document.body)
  })
  expect(post).toHaveBeenCalledTimes(1)
  await act(async () => {
    vi.advanceTimersByTime(31_000)
    fireEvent.keyDown(document.body, { key: 'a' })
  })
  expect(post).toHaveBeenCalledTimes(2)
  visibility.mockReturnValue('hidden')
  await act(async () => {
    vi.advanceTimersByTime(6 * 60_000)
    fireEvent.wheel(document.body)
  })
  visibility.mockReturnValue('visible')
  await act(async () => {
    fireEvent(window, new Event('focus'))
  })
  expect(post).toHaveBeenCalledTimes(2)
  await act(async () => {
    fireEvent.touchStart(document.body)
  })
  expect(post).toHaveBeenCalledTimes(3)
  unmount()
  await act(async () => {
    vi.advanceTimersByTime(31_000)
    fireEvent.pointerDown(document.body)
  })
  expect(post).toHaveBeenCalledTimes(3)
})

it.each([null, { id: 1, username: 'root', role: 100 }])(
  'does not report anonymous or private account activity (%s)',
  async (user) => {
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })
    if (user) useAuthStore.getState().auth.setUser(user)
    showActivity()
    await act(async () => {
      fireEvent.pointerDown(document.body)
    })
    expect(post).not.toHaveBeenCalled()
  }
)
