import { apiClient } from '@/api/client'
import type { PromptErrorFilters, PromptErrorPage, PromptErrorRecord, PromptErrorDeletePreview, PromptErrorDeleteResult } from './types'

const basePath = '/admin/prompt-error-records'

function toQuery(filters: PromptErrorFilters, page?: number, pageSize?: number): Record<string, string | number> {
  const params: Record<string, string | number> = {}
  const map: Array<[keyof PromptErrorFilters, string]> = [
    ['keyword', 'keyword'],
    ['model', 'model'],
    ['error_status', 'error_status'],
    ['group_id', 'group_id'],
    ['user_id', 'user_id'],
    ['api_key_id', 'api_key_id'],
    ['request_id', 'request_id'],
    ['prompt_hash', 'prompt_hash'],
    ['start_at', 'start_at'],
    ['end_at', 'end_at'],
  ]
  for (const [key, qk] of map) {
    const v = (filters[key] || '').trim()
    if (v) params[qk] = v
  }
  if (page) params.page = page
  if (pageSize) params.page_size = pageSize
  // Convert local datetime to ISO if needed
  for (const k of ['start_at', 'end_at'] as const) {
    if (params[k]) {
      const d = new Date(String(params[k]))
      if (!Number.isNaN(d.getTime())) params[k] = d.toISOString()
    }
  }
  return params
}

export async function listRecords(filters: PromptErrorFilters, page: number, pageSize: number): Promise<PromptErrorPage> {
  const { data } = await apiClient.get<PromptErrorPage>(basePath, { params: toQuery(filters, page, pageSize) })
  return data
}

export async function getRecord(id: number): Promise<PromptErrorRecord> {
  const { data } = await apiClient.get<PromptErrorRecord>(`${basePath}/${id}`)
  return data
}

export async function deleteRecord(id: number): Promise<PromptErrorDeleteResult> {
  const { data } = await apiClient.delete<PromptErrorDeleteResult>(`${basePath}/${id}`)
  return data
}

export async function batchDeleteRecords(ids: number[]): Promise<PromptErrorDeleteResult> {
  const { data } = await apiClient.post<PromptErrorDeleteResult>(`${basePath}/batch-delete`, { ids })
  return data
}

export async function previewDelete(filters: PromptErrorFilters): Promise<PromptErrorDeletePreview> {
  const payload: Record<string, unknown> = toQuery(filters)
  // Ensure start_at/end_at are ISO strings
  const { data } = await apiClient.post<PromptErrorDeletePreview>(`${basePath}/delete-preview`, payload)
  return data
}

export async function deleteByFilter(filters: PromptErrorFilters, preview: PromptErrorDeletePreview): Promise<PromptErrorDeleteResult> {
  const { data } = await apiClient.post<PromptErrorDeleteResult>(`${basePath}/delete-by-filter`, {
    filter: toQuery(filters),
    snapshot_max_id: preview.snapshot_max_id,
    filter_hash: preview.filter_hash,
    confirmation_token: preview.confirmation_token,
    confirm: true,
    expires_at: preview.expires_at,
  })
  return data
}

export async function exportCSV(filters: PromptErrorFilters): Promise<Blob> {
  const { data } = await apiClient.get(`${basePath}/export`, {
    params: toQuery(filters),
    responseType: 'blob',
  })
  return data as Blob
}

export const promptErrorAPI = {
  listRecords,
  getRecord,
  deleteRecord,
  batchDeleteRecords,
  previewDelete,
  deleteByFilter,
  exportCSV,
}

export default promptErrorAPI
