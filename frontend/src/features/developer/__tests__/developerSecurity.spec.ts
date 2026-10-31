import { describe, expect, it } from 'vitest'

import {
  REDACTED_VALUE,
  resolveDefaultLiveBaseUrl,
  sanitizeHeaders,
  sanitizeRequestConfig,
  validateLiveTarget,
} from '../security'

describe('developer security helpers', () => {
  it('uses the production origin for same-origin Live requests', () => {
    expect(resolveDefaultLiveBaseUrl('https://api.guili.xyz')).toBe('https://api.guili.xyz')
    expect(resolveDefaultLiveBaseUrl('http://192.168.1.159:3001')).toBe(
      'https://your-api-host',
    )
  })

  it('redacts credential headers case-insensitively', () => {
    expect(
      sanitizeHeaders({
        Authorization: 'Bearer secret',
        COOKIE: 'session=secret',
        'x-api-key': 'sk-secret',
        'Content-Type': 'application/json',
      }),
    ).toEqual({
      Authorization: REDACTED_VALUE,
      COOKIE: REDACTED_VALUE,
      'x-api-key': REDACTED_VALUE,
      'Content-Type': 'application/json',
    })
  })

  it('removes nested credentials from persisted request bodies', () => {
    const result = sanitizeRequestConfig({
      name: 'Chat request',
      method: 'POST',
      url: 'https://api.example.com/v1/chat/completions',
      headers: { Authorization: 'Bearer secret', Accept: 'application/json' },
      body: JSON.stringify({
        model: 'gpt-4.1',
        api_key: 'sk-secret',
        nested: { access_token: 'token', keep: 'value' },
      }),
    })

    expect(result.headers.Authorization).toBe(REDACTED_VALUE)
    expect(result.body).toContain('"model":"gpt-4.1"')
    expect(result.body).toContain('"keep":"value"')
    expect(result.body).not.toContain('sk-secret')
    expect(result.body).not.toContain('access_token')
  })

  it('only accepts HTTPS public targets without embedded credentials', () => {
    expect(validateLiveTarget('https://api.example.com/v1/models')).toEqual({ ok: true })
    expect(validateLiveTarget('http://api.example.com/v1/models').ok).toBe(false)
    expect(validateLiveTarget('https://user:pass@api.example.com/v1/models').ok).toBe(false)
    expect(validateLiveTarget('https://127.0.0.1/v1/models').ok).toBe(false)
    expect(validateLiveTarget('https://192.168.1.20/v1/models').ok).toBe(false)
  })
})
