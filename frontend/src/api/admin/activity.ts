import { apiClient } from '../client'

export type AdminActivityPrizeType = 'balance' | 'group'

export interface AdminActivityPrize {
  id: string
  type: AdminActivityPrizeType
  label: string
  weight: number
  amount?: number
  group_id?: number
  validity_days?: number
}

export interface AdminActivityConfig {
  enabled: boolean
  activity_id: string
  title: string
  description: string
  start_at: string
  end_at: string
  daily_draw_limit: number
  global_draw_limit: number
  prizes: AdminActivityPrize[]
}

export async function getConfig(): Promise<AdminActivityConfig> {
  const { data } = await apiClient.get<AdminActivityConfig>('/admin/activity/config')
  return data
}

export async function updateConfig(config: AdminActivityConfig): Promise<AdminActivityConfig> {
  const { data } = await apiClient.put<AdminActivityConfig>('/admin/activity/config', config)
  return data
}

export const adminActivityAPI = { getConfig, updateConfig }
export default adminActivityAPI
