import { apiClient } from '@/api/client'
import type { Account, CheckMixedChannelRequest, CheckMixedChannelResponse, CreateAccountRequest, PaginatedResponse, UpdateAccountRequest } from '@/types'

type AccountFilters = {
  platform?: string
  type?: string
  status?: string
  group?: string
  search?: string
  privacy_mode?: string
  lite?: string
  include_scheduler_score?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export async function list(page = 1, pageSize = 20, filters?: AccountFilters): Promise<PaginatedResponse<Account>> {
  const { data } = await apiClient.get<PaginatedResponse<Account>>('/user/accounts', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function create(payload: CreateAccountRequest): Promise<Account> {
  const { data } = await apiClient.post<Account>('/user/accounts', payload)
  return data
}

export async function getById(id: number): Promise<Account> {
  const { data } = await apiClient.get<Account>(`/user/accounts/${id}`)
  return data
}

export async function update(id: number, payload: UpdateAccountRequest): Promise<Account> {
  const { data } = await apiClient.put<Account>(`/user/accounts/${id}`, payload)
  return data
}

export async function deleteAccount(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/user/accounts/${id}`)
  return data
}

export async function checkMixedChannelRisk(_payload: CheckMixedChannelRequest): Promise<CheckMixedChannelResponse> {
  return { has_risk: false }
}

export type QuickAddAccountInput = { platform: string; base_url: string; api_key: string }

export async function previewModels(payload: QuickAddAccountInput): Promise<string[]> {
  const { data } = await apiClient.post<{ models: string[] }>('/user/accounts/models/preview', payload)
  return data.models
}

export async function quickAdd(payload: QuickAddAccountInput): Promise<{ models: string[] }> {
  const { data } = await apiClient.post<{ models: string[] }>('/user/accounts/quick-add', payload)
  return data
}

export const userAccountsAPI = { list, getById, create, update, delete: deleteAccount, checkMixedChannelRisk, previewModels, quickAdd }
