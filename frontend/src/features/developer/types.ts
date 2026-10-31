export type PortalLocale = 'zh' | 'en'
export type TranslationStatus = 'current' | 'outdated' | 'translating' | 'missing' | 'review'
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
export type DocumentationCategory = 'start' | 'guides' | 'reference' | 'operations'
export type ApiCategory = 'models' | 'chat' | 'responses' | 'images' | 'billing' | 'gemini'

export interface LocalizedText {
  zh: string
  en: string
}

export interface TranslationRecord {
  status: TranslationStatus
  sourceHash: string
  updatedAt: string
}

export interface DeveloperDocument {
  id: string
  slug: string
  category: DocumentationCategory
  title: LocalizedText
  description: LocalizedText
  content: LocalizedText
  version: string
  updatedAt: string
  tags: string[]
  relatedDocIds: string[]
  translations: {
    zh: TranslationRecord
    en: TranslationRecord
    ja: TranslationRecord
  }
}

export interface ApiParameter {
  name: string
  location: 'path' | 'query' | 'header' | 'body'
  type: string
  required: boolean
  description: LocalizedText
}

export interface ApiError {
  status: number
  code: string
  description: LocalizedText
}

export interface DeveloperEndpoint {
  id: string
  category: ApiCategory
  method: HttpMethod
  path: string
  backendGroup: '/v1' | '/v1beta'
  backendPath: string
  summary: LocalizedText
  description: LocalizedText
  auth: 'api-key'
  version: string
  sourceHandler: string
  parameters: ApiParameter[]
  requestExample: unknown
  responseExample: unknown
  errors: ApiError[]
  rateLimit: LocalizedText
  relatedDocIds: string[]
}

export interface DeveloperSearchResult {
  kind: 'document' | 'endpoint'
  id: string
  title: string
  description: string
  category: string
  path: string
  score: number
}

export interface DeveloperSearchOptions {
  locale: PortalLocale
  siteName?: string
  category?: string
  language?: string
  version?: string
  limit?: number
}

export interface WorkspaceRequest {
  id: string
  name: string
  method: HttpMethod
  url: string
  headers: Record<string, string>
  body: string
  query?: Record<string, string>
  environment?: string
  createdAt: number
  updatedAt: number
}

export interface DebugInput {
  status?: number
  error?: unknown
}

export interface DebugDiagnosis {
  code:
    | 'rate_limit'
    | 'authentication'
    | 'permission'
    | 'not_found'
    | 'invalid_request'
    | 'upstream_error'
    | 'timeout'
    | 'network_or_cors'
    | 'unknown'
  title: LocalizedText
  summary: LocalizedText
  confidence: 'high' | 'medium' | 'low'
  steps: LocalizedText[]
  relatedDocIds: string[]
}
