/**
 * Daily Check-in API endpoints
 */

import { apiClient } from './client'

export interface CheckinStatus {
  enabled: boolean
  checked_in: boolean
  amount: number
  min_amount: number
  max_amount: number
}

export interface CheckinResult {
  amount: number
  message: string
  timestamp: string
}

export async function getCheckinStatus(): Promise<CheckinStatus> {
  const { data } = await apiClient.get<CheckinStatus>('/checkin/status')
  return data
}

export async function doCheckin(): Promise<CheckinResult> {
  const { data } = await apiClient.post<CheckinResult>('/checkin')
  return data
}

export const checkinAPI = { getCheckinStatus, doCheckin }
export default checkinAPI
