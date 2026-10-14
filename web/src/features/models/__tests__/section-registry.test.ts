import { describe, expect, it } from 'vitest'

import { MODELS_SECTION_IDS } from '../section-registry'

describe('model management sections', () => {
  it('keeps document templates in the model management tab group after deployments', () => {
    expect(MODELS_SECTION_IDS).toEqual([
      'metadata',
      'vendors',
      'deployments',
      'document-templates',
    ])
  })
})
