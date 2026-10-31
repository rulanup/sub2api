import { sanitizeRequestConfig } from './security'
import type { WorkspaceRequest } from './types'

const DATABASE_NAME = 'sub2api-developer-portal'
const DATABASE_VERSION = 1
const REQUEST_STORE = 'requests'
const HISTORY_STORE = 'history'
const DEBUG_STORE = 'debugRecords'

export interface DeveloperDebugRecord {
  requestId: string
  method: string
  url: string
  status?: number
  durationMs: number
  requestHeaders: Record<string, string>
  requestBody: string
  responseHeaders: Record<string, string>
  responseBody: string
  createdAt: number
}

export interface RequestHistoryRecord extends WorkspaceRequest {
  historyId: string
}

let databasePromise: Promise<IDBDatabase> | null = null

function openDatabase(): Promise<IDBDatabase> {
  if (typeof indexedDB === 'undefined') {
    return Promise.reject(new Error('IndexedDB is unavailable'))
  }
  if (databasePromise) return databasePromise

  databasePromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(DATABASE_NAME, DATABASE_VERSION)
    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(REQUEST_STORE)) {
        database.createObjectStore(REQUEST_STORE, { keyPath: 'id' })
      }
      if (!database.objectStoreNames.contains(HISTORY_STORE)) {
        database.createObjectStore(HISTORY_STORE, { keyPath: 'historyId' })
      }
      if (!database.objectStoreNames.contains(DEBUG_STORE)) {
        database.createObjectStore(DEBUG_STORE, { keyPath: 'requestId' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('Failed to open IndexedDB'))
  })

  return databasePromise
}

async function getAll<T>(storeName: string): Promise<T[]> {
  const database = await openDatabase()
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(storeName, 'readonly')
    const request = transaction.objectStore(storeName).getAll()
    request.onsuccess = () => resolve(request.result as T[])
    request.onerror = () => reject(request.error ?? new Error('IndexedDB read failed'))
  })
}

async function putValue(storeName: string, value: unknown): Promise<void> {
  const database = await openDatabase()
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(storeName, 'readwrite')
    transaction.objectStore(storeName).put(value)
    transaction.oncomplete = () => resolve()
    transaction.onerror = () => reject(transaction.error ?? new Error('IndexedDB write failed'))
  })
}

async function deleteValue(storeName: string, key: IDBValidKey): Promise<void> {
  const database = await openDatabase()
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(storeName, 'readwrite')
    transaction.objectStore(storeName).delete(key)
    transaction.oncomplete = () => resolve()
    transaction.onerror = () => reject(transaction.error ?? new Error('IndexedDB delete failed'))
  })
}

async function clearStore(storeName: string): Promise<void> {
  const database = await openDatabase()
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(storeName, 'readwrite')
    transaction.objectStore(storeName).clear()
    transaction.oncomplete = () => resolve()
    transaction.onerror = () => reject(transaction.error ?? new Error('IndexedDB clear failed'))
  })
}

export async function listWorkspaceRequests(): Promise<WorkspaceRequest[]> {
  const requests = await getAll<WorkspaceRequest>(REQUEST_STORE)
  return requests.sort((left, right) => right.updatedAt - left.updatedAt)
}

export async function getWorkspaceRequest(id: string): Promise<WorkspaceRequest | undefined> {
  const requests = await getAll<WorkspaceRequest>(REQUEST_STORE)
  return requests.find((request) => request.id === id)
}

export async function saveWorkspaceRequest(request: WorkspaceRequest): Promise<WorkspaceRequest> {
  const safe = sanitizeRequestConfig(request)
  await putValue(REQUEST_STORE, safe)
  return safe
}

export async function deleteWorkspaceRequest(id: string): Promise<void> {
  await deleteValue(REQUEST_STORE, id)
}

export async function addRequestHistory(request: WorkspaceRequest): Promise<void> {
  const safe = sanitizeRequestConfig(request)
  const historyRecord: RequestHistoryRecord = {
    ...safe,
    historyId: safe.id + '-' + Date.now().toString(36),
  }
  await putValue(HISTORY_STORE, historyRecord)
  const records = await listRequestHistory()
  await Promise.all(records.slice(30).map((record) => deleteValue(HISTORY_STORE, record.historyId)))
}

export async function listRequestHistory(): Promise<RequestHistoryRecord[]> {
  const records = await getAll<RequestHistoryRecord>(HISTORY_STORE)
  return records.sort((left, right) => right.updatedAt - left.updatedAt)
}

export async function addDebugRecord(record: DeveloperDebugRecord): Promise<void> {
  const safe: DeveloperDebugRecord = {
    ...record,
    requestHeaders: sanitizeRequestConfig({
      name: '',
      method: 'GET',
      url: '',
      headers: record.requestHeaders,
      body: '',
    }).headers,
    requestBody: sanitizeRequestConfig({
      name: '',
      method: 'POST',
      url: '',
      headers: {},
      body: record.requestBody,
    }).body,
    responseHeaders: sanitizeRequestConfig({
      name: '',
      method: 'GET',
      url: '',
      headers: record.responseHeaders,
      body: '',
    }).headers,
    responseBody: sanitizeRequestConfig({
      name: '',
      method: 'POST',
      url: '',
      headers: {},
      body: record.responseBody,
    }).body,
  }
  await putValue(DEBUG_STORE, safe)
}

export async function listDebugRecords(): Promise<DeveloperDebugRecord[]> {
  const records = await getAll<DeveloperDebugRecord>(DEBUG_STORE)
  return records.sort((left, right) => right.createdAt - left.createdAt)
}

export async function clearDebugRecords(): Promise<void> {
  await clearStore(DEBUG_STORE)
}

export async function clearDeveloperData(): Promise<void> {
  await Promise.all([
    clearStore(REQUEST_STORE),
    clearStore(HISTORY_STORE),
    clearStore(DEBUG_STORE),
  ])
}
