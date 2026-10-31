import { describe, expect, it } from 'vitest'

import { classifyRequestFailure, createRedactedDebugReport } from '../debug'

describe('developer debug diagnostics', () => {
  it('classifies rate limiting with medium confidence and actionable docs', () => {
    const diagnosis = classifyRequestFailure({
      status: 429,
      error: new Error('Too many requests'),
    })

    expect(diagnosis.code).toBe('rate_limit')
    expect(diagnosis.confidence).toBe('high')
    expect(diagnosis.relatedDocIds).toContain('errors-rate-limits')
  })

  it('distinguishes timeout and network failures without overclaiming certainty', () => {
    expect(classifyRequestFailure({ error: new DOMException('Aborted', 'AbortError') }).code).toBe('timeout')
    expect(classifyRequestFailure({ error: new TypeError('Failed to fetch') }).code).toBe('network_or_cors')
  })

  it('redacts credentials from exported debug reports', () => {
    const report = createRedactedDebugReport({
      requestId: 'req-debug-1',
      method: 'POST',
      url: 'https://api.example.com/v1/chat/completions',
      status: 401,
      durationMs: 42,
      requestHeaders: { Authorization: 'Bearer secret' },
      requestBody: '{"api_key":"sk-secret","model":"gpt-4.1"}',
      responseHeaders: { 'Set-Cookie': 'session=secret' },
      responseBody: '{"error":"invalid token"}',
    })

    expect(report).not.toContain('Bearer secret')
    expect(report).not.toContain('sk-secret')
    expect(report).not.toContain('session=secret')
    expect(report).toContain('req-debug-1')
  })
})
