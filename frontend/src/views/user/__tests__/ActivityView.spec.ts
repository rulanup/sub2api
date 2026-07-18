import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ActivityView from '../ActivityView.vue'
import type { ActivityStatus } from '@/api/activity'

const { getStatus, draw, getHistory, refreshUser, play, stop, createIdempotencyKey } = vi.hoisted(() => ({
  getStatus: vi.fn(),
  draw: vi.fn(),
  getHistory: vi.fn(),
  refreshUser: vi.fn(),
  play: vi.fn(),
  stop: vi.fn(),
  createIdempotencyKey: vi.fn(),
}))

vi.mock('@/api/activity', () => ({
  activityAPI: { getStatus, draw, getHistory },
  createIdempotencyKey,
}))

vi.mock('@/stores', () => ({ useAuthStore: () => ({ refreshUser }) }))

vi.mock('vue-i18n', async importOriginal => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('en'),
      t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

vi.mock('@lucky-canvas/vue', () => ({
  LuckyWheel: defineComponent({
    name: 'LuckyWheel',
    emits: ['start', 'end'],
    setup(_, { expose }) {
      expose({ play, stop })
      return () => h('div', { 'data-testid': 'lucky-wheel' })
    },
  }),
}))

const activeStatus = (override: Partial<ActivityStatus> = {}): ActivityStatus => ({
  enabled: true,
  state: 'active',
  activity_id: 'summer',
  title: 'Summer',
  description: 'Draw',
  start_at: '2026-07-01T00:00:00Z',
  end_at: '2026-08-01T00:00:00Z',
  daily_limit: 2,
  daily_used: 0,
  daily_remaining: 2,
  global_limit: 100,
  global_used: 0,
  global_remaining: 100,
  prizes: [
    { id: 'one', type: 'balance', label: 'One', amount: 1 },
    { id: 'vip', type: 'exclusive_group_access', label: 'VIP', group_id: 4, validity_days: 30 },
  ],
  ...override,
})

function mountView() {
  return mount(ActivityView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        LoadingSpinner: true,
      },
    },
  })
}

describe('ActivityView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getStatus.mockReset()
    draw.mockReset()
    getHistory.mockReset()
    refreshUser.mockReset()
    createIdempotencyKey.mockReset()
    Object.defineProperty(HTMLElement.prototype, 'clientWidth', { configurable: true, get: () => 420 })
    getStatus.mockResolvedValue(activeStatus())
    getHistory.mockResolvedValue({ items: [] })
    refreshUser.mockResolvedValue({})
    createIdempotencyKey.mockReturnValueOnce('draw-key-1').mockReturnValueOnce('draw-key-2')
  })

  it('stops on the server-selected segment and reveals only after wheel finish', async () => {
    draw.mockResolvedValue({
      replayed: false,
      result: { id: 7, activity_id: 'summer', prize: activeStatus().prizes[1], subscription_id: 8, subscription_expires_after: '2026-08-16T00:00:00Z', created_at: '2026-07-17T00:00:00Z' },
      daily_limit: 2, daily_used: 1, daily_remaining: 1, global_limit: 100, global_used: 1, global_remaining: 99,
    })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    expect(play).toHaveBeenCalledOnce()
    expect(stop).toHaveBeenCalledWith(1)
    expect(wrapper.find('[data-testid="activity-result"]').exists()).toBe(false)

    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await flushPromises()
    expect(wrapper.get('[data-testid="activity-result"]').text()).toContain('VIP')
    expect(refreshUser).toHaveBeenCalledOnce()
  })

  it('prevents a second draw while the first request is pending', async () => {
    let resolveDraw!: (value: unknown) => void
    draw.mockReturnValue(new Promise(resolve => { resolveDraw = resolve }))
    const wrapper = mountView()
    await flushPromises()
    const button = wrapper.get('[data-testid="activity-draw-button"]')
    await button.trigger('click')
    await button.trigger('click')
    expect(draw).toHaveBeenCalledOnce()
    resolveDraw({ result: { prize: activeStatus().prizes[0] }, daily_remaining: 1, daily_used: 1, global_remaining: 99, global_used: 1 })
  })

  it('reuses an ambiguous request key and rotates it after a successful retry', async () => {
    draw
      .mockRejectedValueOnce({ status: 0, message: 'Network error' })
      .mockResolvedValueOnce({
        result: { id: 8, activity_id: 'summer', prize: activeStatus().prizes[0], created_at: '2026-07-17T00:00:00Z' },
        daily_remaining: 1, daily_used: 1, global_remaining: 99, global_used: 1,
      })
      .mockResolvedValueOnce({
        result: { id: 9, activity_id: 'summer', prize: activeStatus().prizes[1], created_at: '2026-07-17T00:01:00Z' },
        daily_remaining: 0, daily_used: 2, global_remaining: 98, global_used: 2,
      })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await nextTick()
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    expect(draw.mock.calls.slice(0, 2)).toEqual([['draw-key-1'], ['draw-key-1']])

    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await flushPromises()
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    expect(draw).toHaveBeenLastCalledWith('draw-key-2')
  })

  it('reuses the pending key after a 503 response', async () => {
    draw
      .mockRejectedValueOnce({ status: 503, message: 'Service unavailable' })
      .mockResolvedValueOnce({
        result: { id: 8, activity_id: 'summer', prize: activeStatus().prizes[0], created_at: '2026-07-17T00:00:00Z' },
        daily_remaining: 1, daily_used: 1, global_remaining: 99, global_used: 1,
      })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await nextTick()
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()

    expect(draw.mock.calls).toEqual([['draw-key-1'], ['draw-key-1']])
  })

  it('rotates the pending key after a 400 response', async () => {
    draw
      .mockRejectedValueOnce({ status: 400, message: 'Invalid request' })
      .mockResolvedValueOnce({
        result: { id: 8, activity_id: 'summer', prize: activeStatus().prizes[0], created_at: '2026-07-17T00:00:00Z' },
        daily_remaining: 1, daily_used: 1, global_remaining: 99, global_used: 1,
      })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await nextTick()
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()

    expect(draw.mock.calls).toEqual([['draw-key-1'], ['draw-key-2']])
  })

  it('adds a server-selected prize missing from the old snapshot and reveals it', async () => {
    const newPrize = { id: 'new', type: 'balance' as const, label: 'New prize', amount: 5 }
    draw.mockResolvedValue({
      result: { id: 10, activity_id: 'summer', prize: newPrize, created_at: '2026-07-17T00:00:00Z' },
      daily_remaining: 1, daily_used: 1, global_remaining: 99, global_used: 1,
    })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    expect(stop).toHaveBeenCalledWith(2)
    expect(wrapper.text()).toContain('New prize')

    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await flushPromises()
    expect(wrapper.get('[data-testid="activity-result"]').text()).toContain('New prize')
  })

  it('keeps a confirmed balance reward visible when refreshes fail', async () => {
    draw.mockResolvedValue({
      result: { id: 8, activity_id: 'summer', prize: activeStatus().prizes[0], balance_after: 12, created_at: '2026-07-17T00:00:00Z' },
      daily_remaining: 1, daily_used: 1, global_remaining: 99, global_used: 1,
    })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    getStatus.mockRejectedValueOnce(new Error('refresh failed'))
    getHistory.mockRejectedValueOnce(new Error('refresh failed'))
    refreshUser.mockRejectedValueOnce(new Error('refresh failed'))
    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await flushPromises()
    expect(wrapper.get('[data-testid="activity-result"]').text()).toContain('One')
    expect(wrapper.get('[data-testid="activity-refresh-warning"]').text()).toContain('activity.refreshFailed')
    expect(wrapper.get('[data-testid="activity-history-error"]').exists()).toBe(true)
  })

  it('preserves existing history and offers retry when history refresh fails', async () => {
    getHistory.mockResolvedValueOnce({ items: [{ id: 1, activity_id: 'summer', prize: activeStatus().prizes[0], created_at: '2026-07-16T00:00:00Z' }] })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('One')

    getHistory.mockRejectedValueOnce(new Error('history failed'))
    draw.mockResolvedValue({
      result: { id: 8, activity_id: 'summer', prize: activeStatus().prizes[0], created_at: '2026-07-17T00:00:00Z' },
      daily_remaining: 1, daily_used: 1, global_remaining: 99, global_used: 1,
    })
    await wrapper.get('[data-testid="activity-draw-button"]').trigger('click')
    await flushPromises()
    wrapper.getComponent({ name: 'LuckyWheel' }).vm.$emit('end')
    await flushPromises()
    expect(wrapper.get('[data-testid="activity-history-error"]').text()).toContain('history failed')
    expect(wrapper.get('[data-testid="activity-history-list"]').text()).toContain('One')
    expect(wrapper.text()).not.toContain('activity.emptyHistory')
  })

  it.each([
    ['disabled', activeStatus({ enabled: false, state: 'disabled' }), 'activity.states.disabled'],
    ['upcoming', activeStatus({ state: 'upcoming' }), 'activity.states.upcoming'],
    ['ended', activeStatus({ state: 'ended' }), 'activity.states.ended'],
    ['global exhausted', activeStatus({ state: 'exhausted', global_remaining: 0 }), 'activity.states.exhausted'],
    ['daily exhausted', activeStatus({ daily_remaining: 0 }), 'activity.states.dailyExhausted'],
    ['no eligible prize', activeStatus({ prizes: [] }), 'activity.noEligible'],
  ])('disables drawing for %s state', async (_name, state, message) => {
    getStatus.mockResolvedValue(state)
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.get('[data-testid="activity-draw-button"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain(message)
    await nextTick()
  })
})
