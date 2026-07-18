import { apiClient } from './client'

export type ActivityState = 'disabled' | 'upcoming' | 'active' | 'exhausted' | 'ended'
export type ActivityPrizeType = 'balance' | 'exclusive_group_access'

export interface ActivityPrize {
  id: string
  type: ActivityPrizeType
  label: string
  amount?: number
  group_id?: number
  validity_days?: number
}

export interface ActivityStatus {
  enabled: boolean
  state: ActivityState
  activity_id?: string
  title?: string
  description?: string
  start_at?: string
  end_at?: string
  daily_limit: number
  daily_used: number
  daily_remaining: number
  global_limit: number
  global_used: number
  global_remaining: number
  prizes: ActivityPrize[]
}

export interface ActivityDrawRecord {
  id: number
  activity_id: string
  prize: ActivityPrize
  balance_before?: number
  balance_after?: number
  subscription_id?: number
  subscription_expires_before?: string
  subscription_expires_after?: string
  created_at: string
}

export interface ActivityDrawResponse {
  replayed: boolean
  result: ActivityDrawRecord
  daily_limit: number
  daily_used: number
  daily_remaining: number
  global_limit: number
  global_used: number
  global_remaining: number
}

export interface ActivityHistoryResponse {
  items: ActivityDrawRecord[]
}

export function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  const bytes = new Uint8Array(16)
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    crypto.getRandomValues(bytes)
  } else {
    for (let index = 0; index < bytes.length; index += 1) {
      bytes[index] = Math.floor(Math.random() * 256)
    }
  }
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const value = Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
  return `${value.slice(0, 8)}-${value.slice(8, 12)}-${value.slice(12, 16)}-${value.slice(16, 20)}-${value.slice(20)}`
}

export async function getActivityStatus(): Promise<ActivityStatus> {
  const { data } = await apiClient.get<ActivityStatus>('/activity/status')
  return data
}

export async function drawActivity(idempotencyKey = createIdempotencyKey()): Promise<ActivityDrawResponse> {
  const { data } = await apiClient.post<ActivityDrawResponse>('/activity/draw', undefined, {
    headers: { 'Idempotency-Key': idempotencyKey },
  })
  return data
}

export async function getActivityHistory(limit = 10): Promise<ActivityHistoryResponse> {
  const { data } = await apiClient.get<ActivityHistoryResponse>('/activity/history', { params: { limit } })
  return data
}

export const activityAPI = { getStatus: getActivityStatus, draw: drawActivity, getHistory: getActivityHistory }
export default activityAPI
