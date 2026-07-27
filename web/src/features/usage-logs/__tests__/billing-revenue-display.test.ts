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

import { getBillingRevenueItems } from '../lib/billing-revenue.ts'

const other = {
  admin_info: {
    billing_revenue: {
      group_special_ratio: -20,
      model_token_adjustment: 0,
    },
  },
}

describe('getBillingRevenueItems', () => {
  test('returns both configured revenue values for administrators', () => {
    assert.deepEqual(getBillingRevenueItems(other, true), [
      {
        key: 'group_special_ratio',
        labelKey: 'Group special ratio revenue',
        quota: -20,
      },
      {
        key: 'model_token_adjustment',
        labelKey: 'Model ratio revenue',
        quota: 0,
      },
    ])
  })

  test('returns no revenue values for non-administrators', () => {
    assert.deepEqual(getBillingRevenueItems(other, false), [])
  })

  test('ignores absent and non-finite revenue values', () => {
    assert.deepEqual(
      getBillingRevenueItems(
        {
          admin_info: {
            billing_revenue: {
              group_special_ratio: Number.NaN,
            },
          },
        },
        true
      ),
      []
    )
  })
})
