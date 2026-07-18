import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from '@/api/client'
import { createIdempotencyKey, drawActivity, getActivityStatus } from '@/api/activity'

vi.mock('@/api/client', () => ({
  apiClient: { get: vi.fn(), post: vi.fn() },
}))

describe('activity API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('sends a unique idempotency header for draws', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { replayed: false } })
    await drawActivity('draw-key-1')
    expect(apiClient.post).toHaveBeenCalledWith('/activity/draw', undefined, {
      headers: { 'Idempotency-Key': 'draw-key-1' },
    })
  })

  it('generates an allowed fallback UUID', () => {
    const key = createIdempotencyKey()
    expect(key).toMatch(/^[A-Za-z0-9._:-]{1,128}$/)
  })

  it('does not convert status errors into a disabled response', async () => {
    const error = { code: 'NETWORK_ERROR', message: 'offline' }
    vi.mocked(apiClient.get).mockRejectedValue(error)
    await expect(getActivityStatus()).rejects.toBe(error)
  })
})
