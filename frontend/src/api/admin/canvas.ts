import { apiClient } from '@/api/client'

/** 无限画布管理员设置：group_id 为画布专用 key 绑定的分组，0 表示未指定。 */
export interface CanvasSettings {
  group_id: number
}

export async function getCanvasSettings(): Promise<CanvasSettings> {
  const { data } = await apiClient.get<CanvasSettings>('/admin/canvas/settings')
  return { group_id: Number(data?.group_id) || 0 }
}

export async function updateCanvasSettings(groupId: number): Promise<CanvasSettings> {
  const { data } = await apiClient.put<CanvasSettings>('/admin/canvas/settings', { group_id: groupId })
  return { group_id: Number(data?.group_id) || 0 }
}
