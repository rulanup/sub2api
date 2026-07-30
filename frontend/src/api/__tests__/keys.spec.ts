import { afterEach, describe, expect, it, vi } from 'vitest'

import { testKey } from '@/api/keys'

describe('keysAPI.test', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('tests the key against the same-origin models endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      object: 'list',
      data: [{ id: 'model-a' }, { id: 'model-b' }]
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(testKey('sk-test')).resolves.toEqual({ model_count: 2 })
    expect(fetchMock).toHaveBeenCalledWith('/v1/models', expect.objectContaining({
      method: 'GET',
      credentials: 'same-origin',
      cache: 'no-store',
      headers: expect.objectContaining({ Authorization: 'Bearer sk-test' })
    }))
  })

  it('surfaces a gateway rejection without returning the key', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { message: 'API key is inactive' }
    }), {
      status: 401,
      headers: { 'Content-Type': 'application/json' }
    })))

    await expect(testKey('sk-secret')).rejects.toThrow('API key is inactive')
  })
})
