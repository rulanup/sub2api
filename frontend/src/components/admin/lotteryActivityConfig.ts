import type { AdminActivityConfig, AdminActivityPrize } from '@/api/admin/activity'

export const ACTIVITY_SLUG_PATTERN = /^[a-z0-9][a-z0-9_-]{0,63}$/

export interface LotteryActivityConfigForm extends Omit<AdminActivityConfig, 'start_at' | 'end_at'> {
  start_at: string
  end_at: string
}

function codePointLength(value: string): number {
  return Array.from(value).length
}

export function utcToLocalDateTime(value: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

export function localDateTimeToUTC(value: string): string {
  const date = new Date(value)
  if (!value || Number.isNaN(date.getTime())) return ''
  return date.toISOString().replace('.000Z', 'Z')
}

export function configToForm(config: AdminActivityConfig): LotteryActivityConfigForm {
  return {
    ...config,
    description: config.description || '',
    start_at: utcToLocalDateTime(config.start_at),
    end_at: utcToLocalDateTime(config.end_at),
    prizes: config.prizes.map(prize => ({ ...prize })),
  }
}

function normalizePrize(prize: AdminActivityPrize): AdminActivityPrize {
  const base = {
    id: prize.id.trim(),
    type: prize.type,
    label: prize.label.trim(),
    weight: Number(prize.weight),
  }
  if (prize.type === 'balance') {
    return { ...base, amount: Number(prize.amount) }
  }
  return {
    ...base,
    group_id: Number(prize.group_id),
    validity_days: Number(prize.validity_days),
  }
}

export function formToConfig(form: LotteryActivityConfigForm): AdminActivityConfig {
  return {
    enabled: form.enabled,
    activity_id: form.activity_id.trim(),
    title: form.title.trim(),
    description: form.description.trim(),
    start_at: localDateTimeToUTC(form.start_at),
    end_at: localDateTimeToUTC(form.end_at),
    daily_draw_limit: Number(form.daily_draw_limit),
    global_draw_limit: Number(form.global_draw_limit),
    prizes: form.prizes.map(normalizePrize),
  }
}

export function validateActivityConfig(config: AdminActivityConfig): string | null {
  if (!ACTIVITY_SLUG_PATTERN.test(config.activity_id)) return 'activityId'
  if (!config.title || codePointLength(config.title) > 120) return 'title'
  if (codePointLength(config.description) > 2000) return 'description'
  const start = Date.parse(config.start_at)
  const end = Date.parse(config.end_at)
  if (!config.start_at.endsWith('Z') || !config.end_at.endsWith('Z') || !Number.isFinite(start) || !Number.isFinite(end) || start >= end) return 'dates'
  if (!Number.isInteger(config.daily_draw_limit) || config.daily_draw_limit < 1 || config.daily_draw_limit > 100) return 'dailyLimit'
  if (!Number.isInteger(config.global_draw_limit) || config.global_draw_limit < 1 || config.global_draw_limit > 10_000_000) return 'globalLimit'
  if (config.prizes.length < 2 || config.prizes.length > 12) return 'prizeCount'
  const ids = new Set<string>()
  for (const prize of config.prizes) {
    if (!ACTIVITY_SLUG_PATTERN.test(prize.id) || ids.has(prize.id)) return 'prizeId'
    ids.add(prize.id)
    if (!prize.label || codePointLength(prize.label) > 120) return 'prizeLabel'
    if (!Number.isSafeInteger(prize.weight) || prize.weight < 1) return 'weight'
    if (prize.type === 'balance') {
      const amount = prize.amount
      if (typeof amount !== 'number' || !Number.isFinite(amount) || amount < 0.00000001 || amount > 1_000_000 || Math.round(amount * 100_000_000) / 100_000_000 !== amount) return 'amount'
    } else if (!Number.isInteger(prize.group_id) || Number(prize.group_id) < 1 || !Number.isInteger(prize.validity_days) || Number(prize.validity_days) < 1 || Number(prize.validity_days) > 36_500) {
      return 'group'
    }
  }
  return null
}
