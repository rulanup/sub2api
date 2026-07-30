import { afterEach, describe, expect, it, vi } from 'vitest'

import { testKey } from '@/api/keys'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

describe('keysAPI.test', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    post.mockReset()
  })

  it('selects a chat model and measures it through the latency endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      object: 'list',
      data: [{ id: 'text-embedding-3-small' }, { id: 'gpt-5.5' }]
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }))
    vi.stubGlobal('fetch', fetchMock)
    post.mockResolvedValue({
      data: { model: 'gpt-5.5', latency: 123, status: 'ok' }
    })

    await expect(testKey('sk-test', 7)).resolves.toEqual({
      model: 'gpt-5.5',
      latency: 123,
      status: 'ok'
    })
    expect(fetchMock).toHaveBeenCalledWith('/v1/models', expect.objectContaining({
      method: 'GET',
      credentials: 'same-origin',
      cache: 'no-store',
      headers: expect.objectContaining({ Authorization: 'Bearer sk-test' })
    }))
    expect(post).toHaveBeenCalledWith('/usage/test-model-latency', {
      model: 'gpt-5.5',
      key_id: 7
    })
  })

  it('surfaces a gateway rejection without returning the key', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { message: 'API key is inactive' }
    }), {
      status: 401,
      headers: { 'Content-Type': 'application/json' }
    })))

    await expect(testKey('sk-secret', 8)).rejects.toThrow('API key is inactive')
    expect(post).not.toHaveBeenCalled()
  })

  it('extracts a readable message from a failed model response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: [{ id: 'gpt-5.5' }]
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    })))
    post.mockResolvedValue({
      data: {
        model: 'gpt-5.5',
        latency: 245,
        status: 'error',
        error: JSON.stringify({ error: { message: 'upstream unavailable' } })
      }
    })

    await expect(testKey('sk-test', 9)).resolves.toEqual({
      model: 'gpt-5.5',
      latency: 245,
      status: 'error',
      error: 'upstream unavailable'
    })
  })
})
