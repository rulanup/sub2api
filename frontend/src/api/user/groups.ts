import { apiClient } from '@/api/client'
import type { AdminGroup } from '@/types'

export type PrivateGroupInput = { name: string; description: string; platform: string }

async function listAll(): Promise<AdminGroup[]> {
  const { data } = await apiClient.get<AdminGroup[]>('/user/groups')
  return data
}

async function list(page = 1, pageSize = 20, filters: { platform?: string; status?: string; search?: string } = {}) {
  let items = await listAll()
  if (filters.platform) items = items.filter(item => item.platform === filters.platform)
  if (filters.status) items = items.filter(item => item.status === filters.status)
  if (filters.search) {
    const search = filters.search.toLowerCase()
    items = items.filter(item => item.name.toLowerCase().includes(search) || (item.description || '').toLowerCase().includes(search))
  }
  const total = items.length
  return { items: items.slice((page - 1) * pageSize, page * pageSize), total, page, page_size: pageSize, pages: Math.ceil(total / pageSize) }
}

async function create(payload: PrivateGroupInput): Promise<AdminGroup> {
  const { data } = await apiClient.post<AdminGroup>('/user/groups', payload)
  return data
}

async function update(id: number, payload: PrivateGroupInput): Promise<AdminGroup> {
  const { data } = await apiClient.put<AdminGroup>(`/user/groups/${id}`, payload)
  return data
}

async function deleteGroup(id: number): Promise<void> {
  await apiClient.delete(`/user/groups/${id}`)
}

export const userPrivateGroupsAPI = { list, listAll, create, update, delete: deleteGroup }
