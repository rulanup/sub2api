import type { WorkspaceRequest } from './types'

export const REDACTED_VALUE = '[REDACTED]'
export const MAX_LIVE_RESPONSE_BYTES = 2 * 1024 * 1024
export const MAX_LIVE_STREAM_BYTES = 4 * 1024 * 1024
export const DEFAULT_LIVE_TIMEOUT_MS = 60_000
export const LIVE_BASE_URL_PLACEHOLDER = 'https://your-api-host'

const SENSITIVE_HEADER_NAMES = new Set([
  'authorization',
  'proxy-authorization',
  'cookie',
  'set-cookie',
  'x-api-key',
  'api-key',
  'x-auth-token',
  'x-access-token',
])

const SENSITIVE_KEY_PATTERN =
  /(^|_)(api_?key|access_?token|refresh_?token|auth_?token|password|secret|authorization|cookie)(_|$)/i

export function isSensitiveHeader(name: string): boolean {
  return SENSITIVE_HEADER_NAMES.has(name.trim().toLocaleLowerCase())
}

export function isSensitiveKey(name: string): boolean {
  return SENSITIVE_KEY_PATTERN.test(name.trim())
}

export function sanitizeHeaders(headers: Record<string, string>): Record<string, string> {
  return Object.fromEntries(
    Object.entries(headers).map(([name, value]) => [
      name,
      isSensitiveHeader(name) ? REDACTED_VALUE : value,
    ]),
  )
}

export function sanitizeJsonValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sanitizeJsonValue)
  }

  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .filter(([key]) => !isSensitiveKey(key))
        .map(([key, child]) => [key, sanitizeJsonValue(child)]),
    )
  }

  return value
}

export function sanitizeBody(body: string): string {
  if (!body.trim()) return body

  try {
    return JSON.stringify(sanitizeJsonValue(JSON.parse(body)))
  } catch {
    return body
      .replace(/((?:api_?key|access_?token|refresh_?token|auth_?token|password|secret)\s*[=:]\s*)[^\s&,}]+/gi, '$1' + REDACTED_VALUE)
      .replace(/((?:authorization|cookie)\s*:\s*)[^\r\n]+/gi, '$1' + REDACTED_VALUE)
  }
}

export interface RequestConfigInput {
  name: string
  method: WorkspaceRequest['method']
  url: string
  headers: Record<string, string>
  body: string
  query?: Record<string, string>
  environment?: string
  createdAt?: number
  updatedAt?: number
}

export function sanitizeRequestConfig<T extends RequestConfigInput>(request: T): T {
  return {
    ...request,
    headers: sanitizeHeaders(request.headers),
    body: sanitizeBody(request.body),
    query: request.query
      ? Object.fromEntries(
          Object.entries(request.query).filter(([name]) => !isSensitiveKey(name)),
        )
      : undefined,
  }
}

function isPrivateIpv4(hostname: string): boolean {
  const match = hostname.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/)
  if (!match) return false

  const octets = match.slice(1).map(Number)
  if (octets.some((octet) => octet < 0 || octet > 255)) return true

  return (
    octets[0] === 10 ||
    octets[0] === 127 ||
    (octets[0] === 169 && octets[1] === 254) ||
    (octets[0] === 172 && octets[1] >= 16 && octets[1] <= 31) ||
    (octets[0] === 192 && octets[1] === 168)
  )
}

function isPrivateHostname(hostname: string): boolean {
  const value = hostname.trim().toLocaleLowerCase().replace(/^\[|\]$/g, '')
  if (!value || value === 'localhost' || value.endsWith('.localhost') || value.endsWith('.local')) {
    return true
  }
  if (value === '::1' || value.startsWith('fc') || value.startsWith('fd') || value.startsWith('fe80:')) {
    return true
  }
  return isPrivateIpv4(value)
}

export type LiveTargetValidation =
  | { ok: true }
  | { ok: false; reason: 'invalid_url' | 'https_required' | 'embedded_credentials' | 'private_network' }

export function resolveDefaultLiveBaseUrl(origin: string): string {
  try {
    const url = new URL(origin)
    return url.protocol === 'https:' ? url.origin : LIVE_BASE_URL_PLACEHOLDER
  } catch {
    return LIVE_BASE_URL_PLACEHOLDER
  }
}

export function validateLiveTarget(target: string): LiveTargetValidation {
  let url: URL
  try {
    url = new URL(target)
  } catch {
    return { ok: false, reason: 'invalid_url' }
  }

  if (url.protocol !== 'https:') return { ok: false, reason: 'https_required' }
  if (url.username || url.password) return { ok: false, reason: 'embedded_credentials' }
  if (isPrivateHostname(url.hostname)) return { ok: false, reason: 'private_network' }

  return { ok: true }
}

export function parseJsonObject(value: string): Record<string, unknown> {
  if (!value.trim()) return {}
  const parsed: unknown = JSON.parse(value)
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error('Expected a JSON object')
  }
  return parsed as Record<string, unknown>
}
