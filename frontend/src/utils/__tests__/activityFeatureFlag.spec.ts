import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const store = vi.hoisted(() => ({ cachedPublicSettings: null as null | { activity_enabled?: boolean } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => store }))

describe('activity feature flag', () => {
  beforeEach(() => { store.cachedPublicSettings = null })

  it('is hidden until the opt-in public setting is enabled', () => {
    expect(isFeatureFlagEnabled(FeatureFlags.activity)).toBe(false)
    store.cachedPublicSettings = { activity_enabled: true }
    expect(isFeatureFlagEnabled(FeatureFlags.activity)).toBe(true)
  })

  it('is explicitly hidden when disabled', () => {
    store.cachedPublicSettings = { activity_enabled: false }
    expect(isFeatureFlagEnabled(FeatureFlags.activity)).toBe(false)
  })
})
