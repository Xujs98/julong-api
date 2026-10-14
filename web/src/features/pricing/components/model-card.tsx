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
import { ChevronRight, Zap } from 'lucide-react'
import { memo, useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { DEFAULT_TOKEN_UNIT } from '../constants'
import { useBillingTime } from '../hooks/use-billing-time'
import type { ParsedTaskTier } from '../lib/billing-expr'
import {
  getCardExamplePrice,
  getDynamicDisplayGroupRatio,
  getDynamicPriceUnitLabelKey,
  getDynamicPricingSummary,
  isUnconfiguredTaskUsageModel,
  formatTaskUsageUnitPrice,
  type DynamicPricingTier,
} from '../lib/dynamic-price'
import { parseTags } from '../lib/filters'
import { isTokenBasedModel } from '../lib/model-helpers'
import { formatPrice, formatRequestPrice } from '../lib/price'
import { getTaskNumberFields } from '../lib/task-expr'
import {
  taskPriceLabel,
  taskPricingConditions,
  taskUsageUnitLabel,
} from '../lib/task-price-display'
import type { PricingModel, PriceType, TokenUnit } from '../types'
import { ModelBillingModeBadge } from './model-billing-mode-badge'
import { ModelPerfBadge, type ModelPerfBadgeData } from './model-perf-badge'

function isTaskTier(tier: DynamicPricingTier): tier is ParsedTaskTier {
  return 'unitPrices' in tier
}

export interface ModelCardProps {
  model: PricingModel
  onClick: (modelName: string) => void
  priceRate?: number
  usdExchangeRate?: number
  tokenUnit?: TokenUnit
  showRechargePrice?: boolean
  selectedGroup?: string
  perf?: ModelPerfBadgeData
}

export const ModelCard = memo(function ModelCard(props: ModelCardProps) {
  const { t, i18n } = useTranslation()
  const tokenUnit = props.tokenUnit ?? DEFAULT_TOKEN_UNIT
  const priceRate = props.priceRate ?? 1
  const usdExchangeRate = props.usdExchangeRate ?? 1
  const showRechargePrice = props.showRechargePrice ?? false
  const isTokenBased = isTokenBasedModel(props.model)
  const tokenUnitLabel = tokenUnit === 'K' ? '1K' : '1M'
  const tags = parseTags(props.model.tags)
  const groups = props.model.enable_groups || []
  const endpoints = props.model.supported_endpoint_types || []
  const modelIconKey = props.model.icon || props.model.vendor_icon
  const modelIcon = modelIconKey ? getLobeIcon(modelIconKey, 28) : null
  const initial = props.model.model_name?.charAt(0).toUpperCase() || '?'
  const isUnconfiguredTaskUsage = isUnconfiguredTaskUsageModel(props.model)
  const billingTime = useBillingTime(props.model.billing_expr)
  const currency = useSystemConfigStore((state) => state.config.currency)
  const dynamicPriceOptions = useMemo(
    () => ({
      now: billingTime === undefined ? undefined : new Date(billingTime),
      tokenUnit,
      showRechargePrice,
      priceRate,
      usdExchangeRate,
      groupRatioMultiplier: getDynamicDisplayGroupRatio(
        props.model,
        props.selectedGroup
      ),
    }),
    [
      props.model,
      props.selectedGroup,
      billingTime,
      tokenUnit,
      showRechargePrice,
      priceRate,
      usdExchangeRate,
    ]
  )
  const dynamicSummary = useMemo(
    () => getDynamicPricingSummary(props.model, dynamicPriceOptions),
    // Currency is read indirectly by the price formatter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [props.model, dynamicPriceOptions, currency]
  )
  const cardExamplePrice = useMemo(
    () => getCardExamplePrice(props.model, dynamicPriceOptions),
    // Currency is read indirectly by the price formatter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [props.model, dynamicPriceOptions, currency]
  )
  const taskTiers = dynamicSummary?.tiers.filter(isTaskTier) ?? []
  const usageSchemas = [
    props.model.billing_usage_schema,
    ...(props.model.billing_plugin_variants?.map(
      (variant) => variant.billing_usage_schema
    ) ?? []),
  ]
  const isVideoModel =
    props.model.output_modalities?.includes('video') === true ||
    endpoints.includes('openai-video') ||
    usageSchemas.some((schema) =>
      Object.values(schema ?? {}).some((definition) => {
        const descriptions =
          typeof definition.description === 'string'
            ? [definition.description]
            : Object.values(definition.description ?? {})
        return descriptions.some((description) =>
          /\bvideo\b|视频|視頻/.test(description.toLowerCase())
        )
      })
    )
  const hasTaskPriceMatrix =
    isVideoModel &&
    dynamicSummary?.isTaskUsage === true &&
    !dynamicSummary.providerCount &&
    !dynamicSummary.isSpecialExpression &&
    taskTiers.every((tier) => !tier.conditionText) &&
    taskTiers.length > 0
  const configuredFields = getTaskNumberFields(
    props.model.billing_usage_schema
  ).filter(([field]) =>
    taskTiers.some((tier) => Number.isFinite(tier.unitPrices[field]))
  )
  const hasPaidTaskPrice = taskTiers.some(
    (tier) =>
      tier.constant > 0 ||
      Object.values(tier.unitPrices).some((price) => price > 0)
  )
  // Video presets retain both quantities even when only one is charged.
  // Preserve free tiers, but omit a quantity that is zero in every tier.
  const pricedFields = configuredFields.filter(
    ([field]) =>
      !isVideoModel ||
      !hasPaidTaskPrice ||
      taskTiers.some((tier) => tier.unitPrices[field] > 0)
  )
  const conditionFields = [
    ...new Set(
      taskTiers.flatMap((tier) => tier.conditions.map(({ field }) => field))
    ),
  ]
  const billingBasisParts = [
    ...conditionFields.map((field) =>
      taskPriceLabel(
        props.model.billing_usage_schema?.[field]?.description,
        field,
        i18n.language
      )
    ),
    ...new Set(
      pricedFields.map(([, definition]) => {
        if (definition.unit === 'second') return t('Actual duration')
        if (definition.unit === 'count') {
          if (isVideoModel) return t('Per-call')
          return taskUsageUnitLabel(definition, i18n.language, t('times'))
        }
        return taskUsageUnitLabel(
          definition,
          i18n.language,
          t(
            definition.unit === 'token' ? '1M token' : definition.unit || 'unit'
          )
        )
      })
    ),
  ]
  const videoPriceEntries =
    dynamicSummary?.primaryEntries.filter(
      (entry) =>
        (entry.unit === 'second' ||
          entry.unit === 'count' ||
          entry.unit === 'request') &&
        (!hasPaidTaskPrice || (entry.maxValue ?? entry.value) > 0)
    ) ?? []
  const videoHeadlineEntries =
    isVideoModel &&
    !dynamicSummary?.isSpecialExpression &&
    !dynamicSummary?.hasRequestRules &&
    taskTiers.every((tier) => tier.constant === 0)
      ? videoPriceEntries
      : []
  let priceSummary: ReactNode
  if (dynamicSummary) {
    const cacheReadEntries = dynamicSummary.secondaryEntries.filter(
      (entry) => entry.field === 'cacheReadPrice' && entry.unit === 'token'
    )
    if (dynamicSummary.isSpecialExpression) {
      priceSummary = (
        <div className='col-span-full min-w-0'>
          <span className='text-warning'>
            {t('Special billing expression')}
          </span>
          <code className='text-muted-foreground mt-1 line-clamp-2 block font-mono text-xs break-all'>
            {dynamicSummary.rawExpression}
          </code>
        </div>
      )
    } else if (
      dynamicSummary.primaryEntries.length > 0 ||
      cacheReadEntries.length > 0
    ) {
      priceSummary = (
        <>
          {[
            dynamicSummary.primaryEntries.slice(
              0,
              dynamicSummary.providerCount ? 2 : undefined
            ),
            cacheReadEntries,
          ]
            .flat()
            .filter(
              (entry) =>
                !isVideoModel ||
                !hasPaidTaskPrice ||
                (entry.maxValue ?? entry.value) > 0
            )
            .map((entry) => {
              const isVideoUsageUnit =
                isVideoModel &&
                (entry.unit === 'second' || entry.unit === 'count')
              const unitLabelKey = getDynamicPriceUnitLabelKey(entry)
              const unitLabel = isVideoUsageUnit
                ? t(entry.unit === 'second' ? 's' : 'times')
                : taskUsageUnitLabel(
                    entry,
                    i18n.language,
                    unitLabelKey ? t(unitLabelKey) : tokenUnitLabel
                  )
              let label: ReactNode = null
              if (isVideoUsageUnit) {
                label = null
              } else if (entry.labelKind !== 'schema') {
                label = t(entry.shortLabel)
              } else {
                label = taskPriceLabel(
                  entry.description,
                  entry.shortLabel,
                  i18n.language
                )
              }
              return (
                <div
                  key={entry.key}
                  className='bg-muted/20 flex min-w-0 flex-col gap-1 rounded-md border px-3 py-2.5'
                >
                  {label && (
                    <span className='text-muted-foreground text-xs break-words whitespace-normal'>
                      {label}
                    </span>
                  )}
                  <span className='flex flex-wrap items-baseline gap-x-1 font-mono text-sm font-semibold tabular-nums'>
                    <span>{entry.formattedRange ?? entry.formatted}</span>
                    <span className='text-muted-foreground text-xs font-normal whitespace-nowrap'>
                      {' '}
                      / {unitLabel}
                    </span>
                  </span>
                </div>
              )
            })}
          {dynamicSummary.isTimePricing && (
            <span className='text-muted-foreground col-span-full text-xs'>
              {t('Current period price')}
            </span>
          )}
          {dynamicSummary.isMixedBilling && (
            <span className='text-muted-foreground col-span-full text-xs'>
              {t('Token or per-call pricing')}
            </span>
          )}
          {cardExamplePrice && (
            <span className='text-muted-foreground col-span-full text-xs break-words'>
              {cardExamplePrice.label} ≈ {cardExamplePrice.formatted}
            </span>
          )}
          {dynamicSummary.isTaskUsage &&
            dynamicSummary.tier?.label &&
            !dynamicSummary.primaryEntries.some(
              (entry) => entry.formattedRange
            ) && (
              <span className='text-muted-foreground col-span-full text-xs break-words'>
                ({dynamicSummary.tier.label})
              </span>
            )}
        </>
      )
    } else {
      priceSummary = (
        <span className='text-muted-foreground col-span-full'>
          {dynamicSummary.hasUnconfiguredProviders
            ? t('Usage-based billing · price not configured')
            : t('Dynamic Pricing')}
        </span>
      )
    }
  } else if (isUnconfiguredTaskUsage) {
    priceSummary = (
      <span className='text-muted-foreground col-span-full'>
        {t('Usage-based billing · price not configured')}
      </span>
    )
  } else if (isTokenBased) {
    const prices: { type: PriceType; label: string }[] = [
      { type: 'input', label: t('Input') },
      { type: 'output', label: t('Output') },
      ...(props.model.cache_ratio != null
        ? [{ type: 'cache' as const, label: t('Cached') }]
        : []),
    ]
    priceSummary = prices.map((price) => (
      <div
        key={price.type}
        className='bg-muted/20 flex min-w-0 flex-col gap-1 rounded-md border px-3 py-2.5'
      >
        <span className='text-muted-foreground text-xs'>{price.label}</span>
        <span className='font-mono text-sm font-semibold tabular-nums'>
          {formatPrice(
            props.model,
            price.type,
            tokenUnit,
            showRechargePrice,
            priceRate,
            usdExchangeRate,
            props.selectedGroup
          )}
          <span className='text-muted-foreground text-xs font-normal'>
            {' '}
            / {tokenUnitLabel}
          </span>
        </span>
      </div>
    ))
  } else {
    priceSummary = (
      <div className='bg-muted/20 col-span-full flex min-w-0 flex-col gap-1 rounded-md border px-3 py-2.5'>
        <span className='font-mono text-sm font-semibold tabular-nums'>
          {formatRequestPrice(
            props.model,
            showRechargePrice,
            priceRate,
            usdExchangeRate,
            props.selectedGroup
          )}
          <span className='text-muted-foreground text-xs font-normal'>
            {' '}
            / {t('request')}
          </span>
        </span>
      </div>
    )
  }

  return (
    <Card className='hover:ring-foreground/20 h-full min-w-0 gap-3 rounded-lg transition-colors'>
      <CardHeader className='flex flex-row items-start gap-3'>
        <div
          aria-hidden
          className='bg-muted/50 flex size-10 shrink-0 items-center justify-center rounded-lg'
        >
          {modelIcon || (
            <span className='text-muted-foreground text-sm font-bold'>
              {initial}
            </span>
          )}
        </div>
        <div className='min-w-0 flex-1'>
          <h3
            className='line-clamp-2 font-mono text-[15px] leading-snug font-semibold [overflow-wrap:anywhere]'
            title={props.model.model_name}
          >
            {props.model.model_name}
          </h3>
          {videoHeadlineEntries.length > 0 && (
            <div
              role='group'
              aria-label={t('Price')}
              className='mt-1 flex flex-wrap items-baseline gap-x-1 font-mono text-base font-semibold tabular-nums'
            >
              {videoHeadlineEntries.map((entry, index) => (
                <span
                  key={entry.key}
                  className='inline-flex flex-wrap items-baseline gap-x-1'
                >
                  {index > 0 && (
                    <span className='text-muted-foreground mx-1'>{' + '}</span>
                  )}
                  <span>{entry.formattedRange ?? entry.formatted}</span>
                  <span className='text-muted-foreground text-sm font-normal'>
                    {' '}
                    / {t(entry.unit === 'second' ? 's' : 'times')}
                  </span>
                </span>
              ))}
            </div>
          )}
          {props.model.vendor_name && (
            <p
              className='text-muted-foreground mt-1 truncate text-xs'
              title={props.model.vendor_name}
            >
              {props.model.vendor_name}
            </p>
          )}
        </div>
        <div className='flex shrink-0 items-center gap-1.5'>
          <Button
            variant='outline'
            size='sm'
            className='h-8 gap-1 px-2.5'
            onClick={() => props.onClick(props.model.model_name || '')}
          >
            {t('Details')}
            <ChevronRight aria-hidden className='size-3.5' />
          </Button>
          <CopyButton
            value={props.model.model_name}
            tooltip={t('Copy model name')}
            className='size-8'
            iconClassName='size-3.5'
          />
        </div>
      </CardHeader>
      <CardContent className='flex flex-1 flex-col gap-3'>
        <div className='flex min-w-0 flex-col gap-1.5'>
          <p className='text-muted-foreground line-clamp-2 text-[13px] leading-5 break-words'>
            {props.model.description || t('No description available.')}
          </p>
          {tags.length > 0 && (
            <div
              role='group'
              aria-label={t('Tags')}
              className='text-muted-foreground flex min-w-0 items-baseline gap-1.5 text-xs'
            >
              <span className='shrink-0'>{t('Tags')}</span>
              <span className='truncate' title={tags.join(', ')}>
                {tags.slice(0, 2).join(', ')}
              </span>
              {tags.length > 2 && (
                <span className='shrink-0' title={tags.slice(2).join(', ')}>
                  +{tags.length - 2}
                </span>
              )}
            </div>
          )}
        </div>
        <div
          role='group'
          aria-label={t('Pricing')}
          className='mt-auto flex min-w-0 flex-col gap-1.5'
        >
          {isVideoModel && (
            <Badge
              variant='outline'
              className='h-7 gap-1.5 rounded-full border-blue-200 bg-blue-50 px-2.5 text-blue-700 dark:border-blue-800 dark:bg-blue-950/40 dark:text-blue-300'
            >
              <Zap aria-hidden className='text-amber-500' />
              {t('Video generation')}
            </Badge>
          )}
          {!isVideoModel && (
            <ModelBillingModeBadge model={props.model} appearance='caption' />
          )}
          {dynamicSummary?.providerCount && (
            <span className='text-muted-foreground text-xs break-words'>
              {t('{{count}} providers', {
                count: dynamicSummary.providerCount,
              })}
              {dynamicSummary.hasUnconfiguredProviders &&
                ` · ${t('Not configured for some providers')}`}
            </span>
          )}
          {hasTaskPriceMatrix ? (
            <>
              <div className='grid grid-cols-[repeat(auto-fit,minmax(110px,1fr))] gap-2'>
                {taskTiers.map((tier) => {
                  const conditionLabel = taskPricingConditions(
                    tier.conditions,
                    props.model.billing_usage_schema,
                    i18n.language,
                    t
                  )
                  const title =
                    conditionLabel ||
                    (isVideoModel && tier.label === 'base'
                      ? t('Base price')
                      : tier.label) ||
                    t('Base price')
                  const tierHasCharge =
                    tier.constant > 0 ||
                    Object.values(tier.unitPrices).some((price) => price > 0)
                  const unitPrices = pricedFields.flatMap(
                    ([field, definition]) => {
                      const value = Number(tier.unitPrices[field])
                      if (!Number.isFinite(value) || value < 0) return []
                      if (isVideoModel && tierHasCharge && value === 0) {
                        return []
                      }
                      const isVideoUsageUnit =
                        isVideoModel &&
                        (definition.unit === 'second' ||
                          definition.unit === 'count')
                      let unitFallback = t('unit')
                      if (definition.unit === 'second') {
                        unitFallback = t('s')
                      } else if (definition.unit === 'count') {
                        unitFallback = t('times')
                      } else if (definition.unit === 'token') {
                        unitFallback = t('1M token')
                      } else if (definition.unit === 'credit') {
                        unitFallback = t('credit')
                      }
                      return [
                        {
                          field,
                          label: isVideoUsageUnit
                            ? null
                            : taskPriceLabel(
                                definition.description,
                                field,
                                i18n.language
                              ),
                          price: formatTaskUsageUnitPrice(
                            value,
                            dynamicPriceOptions
                          ),
                          unit: isVideoUsageUnit
                            ? t(definition.unit === 'second' ? 's' : 'times')
                            : taskUsageUnitLabel(
                                definition,
                                i18n.language,
                                unitFallback
                              ),
                        },
                      ]
                    }
                  )
                  if (tier.constant > 0) {
                    unitPrices.push({
                      field: 'constant',
                      label: t('Additional charge'),
                      price: formatTaskUsageUnitPrice(
                        tier.constant,
                        dynamicPriceOptions
                      ),
                      unit: t('request'),
                    })
                  }
                  return (
                    <div
                      role='group'
                      aria-label={title}
                      key={JSON.stringify([
                        tier.label,
                        tier.conditions,
                        tier.unitPrices,
                        tier.constant,
                      ])}
                      className='bg-muted/20 flex min-w-0 flex-col gap-2 rounded-md border px-3 py-2.5'
                    >
                      <span className='text-sm font-semibold break-words'>
                        {title}
                      </span>
                      {unitPrices.map((entry, index) => (
                        <div
                          key={entry.field}
                          className='flex min-w-0 flex-col gap-0.5'
                        >
                          {entry.label && (
                            <span className='text-muted-foreground text-[11px] break-words'>
                              {entry.label}
                            </span>
                          )}
                          <span className='font-mono text-sm font-semibold tabular-nums'>
                            {isVideoModel && index > 0 && '+ '}
                            {entry.price}
                            <span className='text-muted-foreground ml-1 text-xs font-normal'>
                              {' '}
                              / {entry.unit}
                            </span>
                          </span>
                        </div>
                      ))}
                    </div>
                  )
                })}
              </div>
              {billingBasisParts.length > 0 && (
                <p className='text-muted-foreground text-xs break-words'>
                  {t('Billing basis')}:{' '}
                  {billingBasisParts.join(
                    isVideoModel && pricedFields.length > 1 ? ' · ' : ' × '
                  )}
                </p>
              )}
            </>
          ) : (
            <div className='grid grid-cols-[repeat(auto-fit,minmax(110px,1fr))] gap-2'>
              {priceSummary}
            </div>
          )}
        </div>
        {(groups.length > 0 || endpoints.length > 0) && (
          <dl
            className={cn(
              'grid min-w-0 grid-cols-2 gap-3 text-xs',
              (groups.length === 0 || endpoints.length === 0) && 'grid-cols-1'
            )}
          >
            {groups.length > 0 && (
              <div className='flex min-w-0 items-baseline gap-1.5'>
                <dt className='text-muted-foreground shrink-0'>
                  {t('Groups')}
                </dt>
                <dd className='flex min-w-0 items-baseline gap-1'>
                  <span className='truncate' title={groups.join(', ')}>
                    {groups[0]}
                  </span>
                  {groups.length > 1 && (
                    <span
                      className='text-muted-foreground shrink-0'
                      title={groups.slice(1).join(', ')}
                    >
                      +{groups.length - 1}
                    </span>
                  )}
                </dd>
              </div>
            )}
            {endpoints.length > 0 && (
              <div className='flex min-w-0 items-baseline gap-1.5'>
                <dt className='text-muted-foreground shrink-0'>
                  {t('Endpoints')}
                </dt>
                <dd className='flex min-w-0 items-baseline gap-1'>
                  <span className='truncate' title={endpoints.join(', ')}>
                    {endpoints.slice(0, 2).join(', ')}
                  </span>
                  {endpoints.length > 2 && (
                    <span
                      className='text-muted-foreground shrink-0'
                      title={endpoints.slice(2).join(', ')}
                    >
                      +{endpoints.length - 2}
                    </span>
                  )}
                </dd>
              </div>
            )}
          </dl>
        )}
      </CardContent>
      <CardFooter className='mt-auto border-0 bg-transparent pt-0'>
        <ModelPerfBadge
          perf={props.perf}
          className='border-border/60 border-t pt-2'
        />
      </CardFooter>
    </Card>
  )
})
