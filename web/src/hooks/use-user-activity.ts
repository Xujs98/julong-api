import { useMutation } from '@tanstack/react-query'
import { useEffect } from 'react'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { requireServerSuccess } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

const activityEvents = [
  'pointerdown',
  'keydown',
  'wheel',
  'touchstart',
  'touchmove',
] as const

// Installed once at the site root. Polling, focus and idle tabs never send activity.
export function useUserActivity() {
  const user = useAuthStore((state) => state.auth.user)
  const userID = user?.id
  const role = user?.role
  const { mutate } = useMutation({
    mutationKey: ['user-activity', userID],
    mutationFn: async () => {
      const response = await api.post<{ success: boolean }>(
        '/api/user/self/activity'
      )
      requireServerSuccess(response.data)
    },
    retry: false,
    meta: { errorToast: false },
  })

  useEffect(() => {
    if (!userID || role === ROLE.SUPER_ADMIN) return
    let lastReportedAt = 0
    const reportActivity = () => {
      if (document.visibilityState !== 'visible') return
      const now = Date.now()
      if (now - lastReportedAt < 30_000) return
      lastReportedAt = now
      mutate()
    }
    for (const event of activityEvents) {
      document.addEventListener(event, reportActivity, {
        capture: true,
        passive: true,
      })
    }
    return () => {
      for (const event of activityEvents) {
        document.removeEventListener(event, reportActivity, true)
      }
    }
  }, [userID, role, mutate])
}
