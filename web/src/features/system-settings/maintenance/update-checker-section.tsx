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
import { useTranslation } from 'react-i18next'

import { SystemUpdateAction } from '@/features/system-update/system-update-action'
import { useStatus } from '@/hooks/use-status'
import { formatTimestamp } from '@/lib/format'

import { SettingsSection } from '../components/settings-section'

type UpdateCheckerSectionProps = {
  julongVersion?: string | null
  upstreamVersion?: string | null
  upstreamCommit?: string | null
  startTime?: number | null
}

export function UpdateCheckerSection(props: UpdateCheckerSectionProps) {
  const { t } = useTranslation()
  const [checking, setChecking] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [release, setRelease] = useState<ReleaseInfo | null>(null)

  const uptime = props.startTime
    ? formatTimestamp(props.startTime)
    : t('Unknown')
  const julongVersion = props.julongVersion || t('Unknown')
  const upstreamVersion = props.upstreamVersion || t('Unknown')
  const upstreamCommit = props.upstreamCommit || t('Unknown')

  const handleCheckUpdates = async () => {
    setChecking(true)
    try {
      const response = await fetch(
        'https://api.github.com/repos/QuantumNous/new-api/releases/latest',
        {
          headers: {
            Accept: 'application/vnd.github+json',
            'User-Agent': 'new-api-dashboard',
          },
        }
      )

      if (!response.ok) {
        throw new Error(t('Failed to contact GitHub releases API'))
      }

      const data = (await response.json()) as ReleaseInfo
      if (!data?.tag_name) {
        throw new Error(t('Unexpected release payload'))
      }

      if (props.upstreamVersion && data.tag_name === props.upstreamVersion) {
        toast.success(
          t('You are running the latest version ({{version}}).', {
            version: data.tag_name,
          })
        )
        return
      }

      setRelease(data)
      setDialogOpen(true)
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : t('Failed to check for updates')
      toast.error(message)
    } finally {
      setChecking(false)
    }
  }

  const goToRelease = () => {
    if (release?.html_url) {
      window.open(release.html_url, '_blank', 'noopener,noreferrer')
    }
  }

  return (
    <>
      <SettingsSection title={t('System maintenance')}>
        <div className='space-y-6'>
          <div className='grid gap-4 md:grid-cols-3'>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Julong version')}
              </div>
              <div className='text-lg font-semibold'>{julongVersion}</div>
            </div>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Merged New API version')}
              </div>
              <div className='text-lg font-semibold'>{upstreamVersion}</div>
              <div className='text-muted-foreground mt-1 font-mono text-xs break-all'>
                {t('Upstream commit')}: {upstreamCommit}
              </div>
            </div>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Uptime since')}
              </div>
              <div className='text-lg font-semibold'>{uptime}</div>
            </div>
            <div className='text-lg font-semibold break-all'>{version}</div>
          </div>
          <div className='rounded-lg border p-4'>
            <div className='text-muted-foreground text-sm'>
              {t('Uptime since')}
            </div>
            <div className='text-lg font-semibold'>{uptime}</div>
          </div>
        </div>
        <SystemUpdateAction compact={false} />
      </div>
    </SettingsSection>
  )
}
