import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const expectedKeys = [
  'title',
  'description',
  'enabled',
  'enabledHint',
  'minAmount',
  'minAmountHint',
  'maxAmount',
  'maxAmountHint',
  'amountRangeError',
] as const

describe('daily check-in admin locale keys', () => {
  it.each([
    ['zh', zh.admin.settings.features.checkin],
    ['en', en.admin.settings.features.checkin],
  ])('exposes the complete %s locale shape', (_locale, checkin) => {
    for (const key of expectedKeys) {
      expect(checkin[key]).toBeTypeOf('string')
      expect(checkin[key]).not.toBe('')
    }
  })
})
