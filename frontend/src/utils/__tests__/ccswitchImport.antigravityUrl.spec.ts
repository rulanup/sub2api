import { describe, expect, it } from 'vitest'
import { buildCcSwitchImportDeeplink, normalizeCcSwitchRootUrl } from '../ccswitchImport'

describe('Antigravity CC Switch endpoint', () => {
  it.each([
    ['https://api.example.com', 'https://api.example.com/antigravity'],
    ['https://api.example.com/', 'https://api.example.com/antigravity'],
    ['https://api.example.com///', 'https://api.example.com/antigravity'],
    ['https://api.example.com/sub2api/', 'https://api.example.com/sub2api/antigravity']
  ])('joins the platform path onto %s for both clients', (baseUrl, endpoint) => {
    for (const clientType of ['claude', 'gemini'] as const) {
      const url = new URL(buildCcSwitchImportDeeplink({
        baseUrl, clientType, platform: 'antigravity', providerName: 'Sub2API',
        apiKey: 'sk-test', usageScript: 'return true'
      }))
      expect(url.searchParams.get('endpoint')).toBe(endpoint)
      expect(url.searchParams.get('app')).toBe(clientType)
      // fork 语义：homepage 使用规范化后的根地址（去除尾部斜杠与 /v1）。
      expect(url.searchParams.get('homepage')).toBe(normalizeCcSwitchRootUrl(baseUrl))
      expect(url.searchParams.has('model')).toBe(false)
    }
  })
})
