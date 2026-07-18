import { describe, expect, it } from 'vitest'
import { configToForm, formToConfig, validateActivityConfig } from '../lotteryActivityConfig'
import type { AdminActivityConfig } from '@/api/admin/activity'

const config: AdminActivityConfig = {
  enabled: true,
  activity_id: 'summer-2026',
  title: 'Summer Draw',
  description: 'Rewards',
  start_at: '2026-07-17T08:30:00Z',
  end_at: '2026-07-20T09:45:00Z',
  daily_draw_limit: 2,
  global_draw_limit: 1000,
  prizes: [
    { id: 'credit', type: 'balance', label: 'Credit', weight: 10, amount: 1.25, group_id: 9, validity_days: 4 },
    { id: 'access', type: 'group', label: 'Access', weight: 2, group_id: 3, validity_days: 30, amount: 5 },
  ],
}

describe('lottery activity config payload', () => {
  it('round-trips local datetime fields as exact UTC RFC3339 values', () => {
    const form = configToForm(config)
    const payload = formToConfig(form)
    expect(payload.start_at).toBe(config.start_at)
    expect(payload.end_at).toBe(config.end_at)
  })

  it('trims fields and removes prize fields that do not apply to the selected type', () => {
    const payload = formToConfig(configToForm(config))
    expect(payload.prizes[0]).toEqual({ id: 'credit', type: 'balance', label: 'Credit', weight: 10, amount: 1.25 })
    expect(payload.prizes[1]).toEqual({ id: 'access', type: 'group', label: 'Access', weight: 2, group_id: 3, validity_days: 30 })
    expect(validateActivityConfig(payload)).toBeNull()
  })

  it.each([
    ['invalid activity ID', { activity_id: 'Bad ID' }, 'activityId'],
    ['reversed dates', { start_at: config.end_at }, 'dates'],
    ['invalid daily limit', { daily_draw_limit: 0 }, 'dailyLimit'],
    ['too few prizes', { prizes: [config.prizes[0]] }, 'prizeCount'],
  ])('rejects %s', (_name, override, expected) => {
    expect(validateActivityConfig({ ...config, ...override })).toBe(expected)
  })

  it('counts emoji as Unicode code points for display text limits', () => {
    const emojiTitle = '😀'.repeat(120)
    expect(validateActivityConfig({ ...config, title: emojiTitle })).toBeNull()
    expect(validateActivityConfig({ ...config, title: `${emojiTitle}😀` })).toBe('title')

    const emojiLabel = '🎁'.repeat(120)
    expect(validateActivityConfig({ ...config, prizes: [{ ...config.prizes[0], label: emojiLabel }, config.prizes[1]] })).toBeNull()
    expect(validateActivityConfig({ ...config, prizes: [{ ...config.prizes[0], label: `${emojiLabel}🎁` }, config.prizes[1]] })).toBe('prizeLabel')
  })
})
