import {
  MAX_LIVE_RESPONSE_BYTES,
  MAX_LIVE_STREAM_BYTES,
  REDACTED_VALUE,
  isSensitiveHeader,
  validateLiveTarget,
} from './security'

export interface LiveRequestOptions {
  url: string
  method: string
  headers: Record<string, string>
  query: Record<string, string>
  body: string
  apiKey: string
  signal?: AbortSignal
  timeoutMs?: number
  onChunk?: (chunk: string) => void
}

export interface LiveRequestResult {
  requestId: string
  status: number
  statusText: string
  headers: Record<string, string>
  body: string
  durationMs: number
  firstByteMs: number
  bytes: number
  streaming: boolean
}

function mergeQuery(url: URL, query: Record<string, string>): void {
  for (const [key, value] of Object.entries(query)) {
    url.searchParams.set(key, value)
  }
}

function buildHeaders(headers: Record<string, string>, apiKey: string): Headers {
  const result = new Headers()
  for (const [key, value] of Object.entries(headers)) {
    if (isSensitiveHeader(key) || value === REDACTED_VALUE) continue
    result.set(key, value)
  }
  result.set('Authorization', 'Bearer ' + apiKey)
  if (!result.has('Accept')) result.set('Accept', 'application/json')
  return result
}

function localRequestId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return 'local-' + crypto.randomUUID()
  }
  return 'local-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 8)
}

export async function executeLiveRequest(options: LiveRequestOptions): Promise<LiveRequestResult> {
  const url = new URL(options.url)
  mergeQuery(url, options.query)
  const validation = validateLiveTarget(url.toString())
  if (!validation.ok) throw new Error('Invalid live target: ' + validation.reason)

  const controller = new AbortController()
  const timeoutId = window.setTimeout(
    () => controller.abort(new DOMException('Request timeout', 'AbortError')),
    options.timeoutMs ?? 60_000,
  )
  const abortFromParent = () => controller.abort(options.signal?.reason)
  if (options.signal?.aborted) {
    abortFromParent()
  } else {
    options.signal?.addEventListener('abort', abortFromParent, { once: true })
  }

  const startedAt = performance.now()
  let firstByteAt = 0
  try {
    const response = await fetch(url, {
      method: options.method,
      headers: buildHeaders(options.headers, options.apiKey),
      body: options.method === 'GET' || options.method === 'HEAD' ? undefined : options.body || undefined,
      signal: controller.signal,
      credentials: 'omit',
      cache: 'no-store',
      redirect: 'follow',
    })
    firstByteAt = performance.now()

    const headers = Object.fromEntries(response.headers.entries())
    const requestId =
      response.headers.get('x-request-id') ||
      response.headers.get('x-sub2api-request-id') ||
      localRequestId()
    const streaming =
      response.headers.get('content-type')?.includes('text/event-stream') === true ||
      (() => {
        try {
          return JSON.parse(options.body || '{}').stream === true
        } catch {
          return false
        }
      })()
    const limit = streaming ? MAX_LIVE_STREAM_BYTES : MAX_LIVE_RESPONSE_BYTES
    const reader = response.body?.getReader()

    if (!reader) {
      const body = await response.text()
      if (new TextEncoder().encode(body).byteLength > limit) {
        throw new Error('Response exceeded the configured size limit')
      }
      options.onChunk?.(body)
      return {
        requestId,
        status: response.status,
        statusText: response.statusText,
        headers,
        body,
        durationMs: Math.round(performance.now() - startedAt),
        firstByteMs: Math.round(firstByteAt - startedAt),
        bytes: new TextEncoder().encode(body).byteLength,
        streaming,
      }
    }

    const decoder = new TextDecoder()
    let body = ''
    let bytes = 0
    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      bytes += value.byteLength
      if (bytes > limit) {
        await reader.cancel()
        throw new Error('Response exceeded the configured size limit')
      }
      const chunk = decoder.decode(value, { stream: true })
      body += chunk
      if (chunk) options.onChunk?.(chunk)
    }
    body += decoder.decode()

    return {
      requestId,
      status: response.status,
      statusText: response.statusText,
      headers,
      body,
      durationMs: Math.round(performance.now() - startedAt),
      firstByteMs: Math.round(firstByteAt - startedAt),
      bytes,
      streaming,
    }
  } finally {
    window.clearTimeout(timeoutId)
    options.signal?.removeEventListener('abort', abortFromParent)
  }
}
