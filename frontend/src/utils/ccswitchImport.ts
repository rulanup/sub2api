import type { GroupPlatform } from '@/types'

export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-5.5'
export const GROK_CC_SWITCH_MODEL = 'grok-4.5'

export type CcSwitchClientType = 'claude' | 'codex' | 'opencode' | 'gemini' | 'grokbuild'

export const CC_SWITCH_TARGETS_BY_PLATFORM: Readonly<
  Record<GroupPlatform, readonly CcSwitchClientType[]>
> = {
  anthropic: ['claude'],
  openai: ['claude', 'codex', 'opencode'],
  gemini: ['gemini'],
  antigravity: ['claude', 'gemini'],
  grok: ['grokbuild', 'opencode'],
  composite: []
}

export interface CcSwitchImportConfig {
  app: CcSwitchClientType
  endpoint: string
  model?: string
  usageBaseUrl?: string
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  platform?: GroupPlatform | string | null
  clientType: CcSwitchClientType
  providerName: string
  apiKey: string
  usageScript: string
}

export function normalizeCcSwitchRootUrl(baseUrl: string): string {
  const trimmed = baseUrl.trim().replace(/\/+$/, '')

  try {
    const url = new URL(trimmed)
    url.pathname = url.pathname.replace(/\/+$/, '').replace(/\/v1$/i, '')
    return url.toString().replace(/\/$/, '')
  } catch {
    return trimmed.replace(/\/v1$/i, '').replace(/\/+$/, '')
  }
}

export function getCcSwitchTargets(
  platform: GroupPlatform | string | undefined | null
): readonly CcSwitchClientType[] {
  if (!platform || !(platform in CC_SWITCH_TARGETS_BY_PLATFORM)) return []
  return CC_SWITCH_TARGETS_BY_PLATFORM[platform as GroupPlatform]
}

export function resolveCcSwitchImportConfig(
  platform: GroupPlatform | string | undefined | null,
  clientType: CcSwitchClientType,
  baseUrl: string
): CcSwitchImportConfig | null {
  if (!getCcSwitchTargets(platform).includes(clientType)) return null

  const root = normalizeCcSwitchRootUrl(baseUrl)
  switch (platform) {
    case 'anthropic':
      return { app: 'claude', endpoint: root }
    case 'openai':
      if (clientType === 'claude') return { app: 'claude', endpoint: root }
      return clientType === 'opencode'
        ? {
            app: 'opencode',
            endpoint: `${root}/v1`,
            model: OPENAI_CC_SWITCH_CODEX_MODEL,
            usageBaseUrl: root
          }
        : {
            app: 'codex',
            endpoint: root,
            model: OPENAI_CC_SWITCH_CODEX_MODEL
          }
    case 'gemini':
      return { app: 'gemini', endpoint: root }
    case 'antigravity':
      return { app: clientType, endpoint: `${root}/antigravity` }
    case 'grok':
      return {
        app: clientType,
        endpoint: `${root}/v1`,
        model: GROK_CC_SWITCH_MODEL,
        usageBaseUrl: root
      }
    default:
      return null
  }
}

function encodeBase64Utf8(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
}

export const CC_SWITCH_USAGE_SCRIPT = `({
  request: {
    url: "{{baseUrl}}/v1/usage",
    method: "GET",
    headers: { "Authorization": "Bearer {{apiKey}}" }
  },
  extractor: function(response) {
    response = response || {};
    var finite = function(value) {
      return typeof value === "number" && isFinite(value);
    };
    var candidates = [];
    var addCandidate = function(remaining, total, used, reset) {
      if (finite(remaining)) {
        candidates.push({ remaining: remaining, total: total, used: used, reset: reset });
      }
    };
    var quota = response.quota || {};
    if (finite(quota.remaining)) {
      addCandidate(quota.remaining, quota.limit, quota.used, quota.reset_at);
    }
    var subscription = response.subscription || {};
    ["daily", "weekly", "monthly"].forEach(function(windowName) {
      var total = subscription[windowName + "_limit_usd"];
      var used = subscription[windowName + "_usage_usd"];
      if (finite(total) && total > 0 && finite(used)) {
        addCandidate(Math.max(0, total - used), total, used, null);
      }
    });
    (Array.isArray(response.rate_limits) ? response.rate_limits : []).forEach(function(limit) {
      if (limit) addCandidate(limit.remaining, limit.limit, limit.used, limit.reset_at);
    });
    if (candidates.length === 0) {
      addCandidate(response.remaining, response.total, response.used, response.reset_at);
      addCandidate(response.balance, response.total, response.used, response.reset_at);
    }
    candidates.sort(function(left, right) { return left.remaining - right.remaining; });
    var selected = candidates[0] || {};
    var hasQuota = finite(quota.limit);
    var hasRates = Array.isArray(response.rate_limits) && response.rate_limits.length > 0;
    var hasSubscription = response.subscription && typeof response.subscription === "object";
    var planName = response.planName;
    if (!planName) {
      if (hasQuota) planName = "API key quota";
      else if (hasRates) planName = "API key rate limits";
      else if (hasSubscription) planName = "Subscription";
      else planName = "Wallet balance";
    }
    var isValid = typeof response.isValid === "boolean"
      ? response.isValid
      : (typeof response.is_active === "boolean" ? response.is_active : true);
    var error = response.error;
    var invalidMessage = response.invalidMessage || response.message ||
      (error && (error.message || error.type)) || (!isValid ? response.status : null);
    var extraParts = [];
    if (selected.reset) extraParts.push("Resets: " + selected.reset);
    if (subscription.weekly_window_start) {
      extraParts.push("Weekly window: " + subscription.weekly_window_start);
    }
    if (response.expires_at || subscription.expires_at) {
      extraParts.push("Expires: " + (response.expires_at || subscription.expires_at));
    }
    return {
      planName: String(planName),
      remaining: finite(selected.remaining) ? selected.remaining : null,
      total: finite(selected.total) ? selected.total : null,
      used: finite(selected.used) ? selected.used : null,
      unit: response.unit || quota.unit || "USD",
      isValid: isValid,
      invalidMessage: invalidMessage ? String(invalidMessage) : null,
      extra: extraParts.length ? extraParts.join("; ") : (response.extra || null)
    };
  }
})`

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const root = normalizeCcSwitchRootUrl(input.baseUrl)
  const config = resolveCcSwitchImportConfig(input.platform, input.clientType, root)
  if (!config) {
    throw new Error(`Unsupported CC-Switch import target: ${input.platform}/${input.clientType}`)
  }

  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', config.app],
    ['name', input.providerName],
    ['homepage', root],
    ['endpoint', config.endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', encodeBase64Utf8(input.usageScript)],
    ['usageAutoInterval', '30']
  ]

  if (config.model) entries.splice(2, 0, ['model', config.model])
  if (config.usageBaseUrl) entries.push(['usageBaseUrl', config.usageBaseUrl])

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}
