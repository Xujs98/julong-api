import { createFileRoute } from '@tanstack/react-router'
import z from 'zod'

import { SubmitErrorReport } from '@/features/error-reports/submit-error-report'

const errorReportSearchSchema = z.object({
  status: z.number().optional().catch(undefined),
  url: z.string().optional().catch(''),
})

export const Route = createFileRoute('/error-report')({
  validateSearch: errorReportSearchSchema,
  component: SubmitErrorReport,
})
