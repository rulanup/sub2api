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
  try {
    const { data } = await apiClient.get<CheckinStatus>('/checkin/status')
    return data as CheckinStatus
  } catch (e: any) {
    // If API returns error (e.g. 403 disabled), return disabled state
    return { enabled: false, checked_in: false, amount: 0, min_amount: 0, max_amount: 0 }
  }
}

export async function doCheckin(): Promise<CheckinResult> {
  const { data } = await apiClient.post<CheckinResult>('/checkin')
  return data as CheckinResult
}

export const checkinAPI = { getCheckinStatus, doCheckin }
export default checkinAPI
