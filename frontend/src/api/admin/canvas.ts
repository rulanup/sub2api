import { apiClient } from '@/api/client'

/** 无限画布管理员设置：group_ids 为画布专用 key 绑定的分组（第一个为主分组），空表示未指定。 */
export interface CanvasSettings {
  group_ids: number[]
}

type CanvasSettingsResponse = { group_ids?: unknown; group_id?: unknown } | null | undefined

function normalize(data: CanvasSettingsResponse): CanvasSettings {
  if (Array.isArray(data?.group_ids)) {
    return { group_ids: data.group_ids.map(Number).filter((id) => Number.isInteger(id) && id > 0) }
  }
  const legacy = Number(data?.group_id) || 0
  return { group_ids: legacy > 0 ? [legacy] : [] }
}

export async function getCanvasSettings(): Promise<CanvasSettings> {
  const { data } = await apiClient.get<CanvasSettingsResponse>('/admin/canvas/settings')
  return normalize(data)
}

export async function updateCanvasSettings(groupIds: number[]): Promise<CanvasSettings> {
  const { data } = await apiClient.put<CanvasSettingsResponse>('/admin/canvas/settings', { group_ids: groupIds })
  return normalize(data)
}
