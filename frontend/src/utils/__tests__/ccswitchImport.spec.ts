import { describe, expect, it } from 'vitest'
import {
  CC_SWITCH_TARGETS_BY_PLATFORM,
  CC_SWITCH_USAGE_SCRIPT,
  GROK_CC_SWITCH_MODEL,
  OPENAI_CC_SWITCH_CODEX_MODEL,
  buildCcSwitchImportDeeplink,
  getCcSwitchTargets,
  normalizeCcSwitchRootUrl,
  resolveCcSwitchImportConfig,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'
import type { GroupPlatform } from '@/types'

function paramsFromDeeplink(deeplink: string): URLSearchParams {
  return new URLSearchParams(deeplink.split('?')[1] || '')
}

function decodeBase64Utf8(value: string): string {
  const bytes = Uint8Array.from(atob(value), (character) => character.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

function usageExtractor(): (response: Record<string, unknown>) => Record<string, unknown> {
  const definition = Function(`"use strict"; return ${CC_SWITCH_USAGE_SCRIPT}`)() as {
    extractor: (response: Record<string, unknown>) => Record<string, unknown>
  }
  return definition.extractor
}

describe('CC-Switch import targets', () => {
  it('defines the compatible target choices for every group platform', () => {
    expect(CC_SWITCH_TARGETS_BY_PLATFORM).toEqual({
      anthropic: ['claude'],
      openai: ['claude', 'codex', 'opencode'],
      gemini: ['gemini'],
      antigravity: ['claude', 'gemini'],
      grok: ['grokbuild', 'opencode']
    })
    expect(getCcSwitchTargets(null)).toEqual([])
    expect(getCcSwitchTargets('unsupported')).toEqual([])
  })

  it.each<{
    platform: GroupPlatform
    clientType: CcSwitchClientType
    app: CcSwitchClientType
    endpoint: string
    model?: string
    usageBaseUrl?: string
  }>([
    {
      platform: 'openai',
      clientType: 'claude',
      app: 'claude',
      endpoint: 'https://api.example.com/gateway'
    },
    {
      platform: 'openai',
      clientType: 'codex',
      app: 'codex',
      endpoint: 'https://api.example.com/gateway',
      model: OPENAI_CC_SWITCH_CODEX_MODEL
    },
    {
      platform: 'openai',
      clientType: 'opencode',
      app: 'opencode',
      endpoint: 'https://api.example.com/gateway/v1',
      model: OPENAI_CC_SWITCH_CODEX_MODEL,
      usageBaseUrl: 'https://api.example.com/gateway'
    },
    {
      platform: 'anthropic',
      clientType: 'claude',
      app: 'claude',
      endpoint: 'https://api.example.com/gateway'
    },
    {
      platform: 'gemini',
      clientType: 'gemini',
      app: 'gemini',
      endpoint: 'https://api.example.com/gateway'
    },
    {
      platform: 'antigravity',
      clientType: 'claude',
      app: 'claude',
      endpoint: 'https://api.example.com/gateway/antigravity'
    },
    {
      platform: 'antigravity',
      clientType: 'gemini',
      app: 'gemini',
      endpoint: 'https://api.example.com/gateway/antigravity'
    },
    {
      platform: 'grok',
      clientType: 'grokbuild',
      app: 'grokbuild',
      endpoint: 'https://api.example.com/gateway/v1',
      model: GROK_CC_SWITCH_MODEL,
      usageBaseUrl: 'https://api.example.com/gateway'
    },
    {
      platform: 'grok',
      clientType: 'opencode',
      app: 'opencode',
      endpoint: 'https://api.example.com/gateway/v1',
      model: GROK_CC_SWITCH_MODEL,
      usageBaseUrl: 'https://api.example.com/gateway'
    }
  ])(
    'builds the exact $platform/$clientType contract',
    ({ platform, clientType, app, endpoint, model, usageBaseUrl }) => {
      const baseUrl = 'https://api.example.com/gateway/v1///'
      const config = resolveCcSwitchImportConfig(platform, clientType, baseUrl)
      expect(config).toEqual({
        app,
        endpoint,
        ...(model ? { model } : {}),
        ...(usageBaseUrl ? { usageBaseUrl } : {})
      })

      const params = paramsFromDeeplink(
        buildCcSwitchImportDeeplink({
          baseUrl,
          platform,
          clientType,
          providerName: 'Sub2API',
          apiKey: 'sk-test',
          usageScript: CC_SWITCH_USAGE_SCRIPT
        })
      )
      expect(params.get('app')).toBe(app)
      expect(params.get('homepage')).toBe('https://api.example.com/gateway')
      expect(params.get('endpoint')).toBe(endpoint)
      expect(params.get('model')).toBe(model ?? null)
      expect(params.get('usageBaseUrl')).toBe(usageBaseUrl ?? null)
      expect(params.get('usageEnabled')).toBe('true')
      expect(params.get('usageAutoInterval')).toBe('30')
      expect(decodeBase64Utf8(params.get('usageScript') || '')).toBe(CC_SWITCH_USAGE_SCRIPT)
    }
  )

  it('normalizes roots and trailing v1 segments without damaging path prefixes', () => {
    expect(normalizeCcSwitchRootUrl(' https://api.example.com/prefix/v1/// ')).toBe(
      'https://api.example.com/prefix'
    )
    expect(normalizeCcSwitchRootUrl('https://api.example.com///')).toBe('https://api.example.com')
  })

  it('rejects incompatible and unsupported targets instead of falling back to Claude', () => {
    expect(resolveCcSwitchImportConfig('grok', 'claude', 'https://api.example.com')).toBeNull()
    expect(resolveCcSwitchImportConfig('openai', 'claude', 'https://api.example.com')).toEqual({
      app: 'claude',
      endpoint: 'https://api.example.com'
    })
    expect(resolveCcSwitchImportConfig('unsupported', 'claude', 'https://api.example.com')).toBeNull()
    expect(() =>
      buildCcSwitchImportDeeplink({
        baseUrl: 'https://api.example.com',
        platform: 'unsupported',
        clientType: 'claude',
        providerName: 'Sub2API',
        apiKey: 'sk-test',
        usageScript: 'return "UTF-8: \u4f59\u989d"'
      })
    ).toThrow('Unsupported CC-Switch import target')
  })
})

describe('CC-Switch usage script', () => {
  const extract = usageExtractor()

  it('returns quota totals and status fields', () => {
    expect(
      extract({
        mode: 'quota_limited',
        status: 'active',
        isValid: true,
        quota: { limit: 100, used: 35, remaining: 65, unit: 'USD' },
        expires_at: '2026-08-01T00:00:00Z'
      })
    ).toMatchObject({
      planName: 'API key quota',
      remaining: 65,
      total: 100,
      used: 35,
      unit: 'USD',
      isValid: true,
      extra: 'Expires: 2026-08-01T00:00:00Z'
    })
  })

  it('uses the minimum finite remaining value for rate-limit-only responses', () => {
    expect(
      extract({
        mode: 'quota_limited',
        isValid: true,
        rate_limits: [
          { window: '5h', limit: 20, used: 3, remaining: 17 },
          { window: '1d', limit: 10, used: 8, remaining: 2, reset_at: '2026-07-19T00:00:00Z' },
          { window: '7d', limit: 50, used: 5, remaining: Number.NaN }
        ]
      })
    ).toMatchObject({
      planName: 'API key rate limits',
      remaining: 2,
      total: 10,
      used: 8,
      extra: 'Resets: 2026-07-19T00:00:00Z'
    })
  })

  it('handles wallet balance and subscription response shapes', () => {
    expect(extract({ balance: 42.5, unit: 'USD' })).toMatchObject({
      planName: 'Wallet balance',
      remaining: 42.5
    })

    expect(
      extract({
        planName: 'Weekly plan',
        subscription: {
          daily_limit_usd: 20,
          daily_usage_usd: 5,
          weekly_limit_usd: 40,
          weekly_usage_usd: 37,
          monthly_limit_usd: null,
          monthly_usage_usd: 0,
          weekly_window_start: '2026-07-13T00:30:00+08:00'
        }
      })
    ).toMatchObject({
      planName: 'Weekly plan',
      remaining: 3,
      total: 40,
      used: 37,
      extra: 'Weekly window: 2026-07-13T00:30:00+08:00'
    })
  })

  it('returns a stable invalid result from API error fields', () => {
    expect(
      extract({ isValid: false, status: 'expired', error: { message: 'Key expired' } })
    ).toMatchObject({
      planName: 'Wallet balance',
      isValid: false,
      invalidMessage: 'Key expired'
    })
  })
})
