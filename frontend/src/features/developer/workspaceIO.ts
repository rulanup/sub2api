import { REDACTED_VALUE, sanitizeBody, sanitizeHeaders, sanitizeRequestConfig } from './security'
import type { HttpMethod, WorkspaceRequest } from './types'

const HTTP_METHODS: HttpMethod[] = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

function requestId(seed: string): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return seed + '-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 8)
}

function toWorkspaceRequest(
  request: Omit<WorkspaceRequest, 'id' | 'createdAt' | 'updatedAt'> & {
    id?: string
    createdAt?: number
    updatedAt?: number
  },
): WorkspaceRequest {
  const now = Date.now()
  const sanitized = sanitizeRequestConfig({
    ...request,
    name: request.name || request.method + ' ' + request.url,
    createdAt: request.createdAt ?? now,
    updatedAt: request.updatedAt ?? now,
  })
  return {
    ...sanitized,
    id: request.id || requestId(request.method + '-' + request.url),
  }
}

export function exportDeveloperWorkspace(requests: WorkspaceRequest[]): string {
  return JSON.stringify(
    {
      schema: 'sub2api.developer-workspace/v1',
      exportedAt: new Date().toISOString(),
      requests: requests.map((request) => toWorkspaceRequest(request)),
    },
    null,
    2,
  )
}

export function importDeveloperWorkspace(input: string | unknown): WorkspaceRequest[] {
  const parsed = parseJsonInput(input) as { schema?: string; requests?: unknown[] }
  if (!parsed || !Array.isArray(parsed.requests)) {
    throw new Error('Invalid developer workspace file')
  }

  return parsed.requests
    .filter((request): request is Record<string, unknown> => {
      return Boolean(request && typeof request === 'object')
    })
    .map((request) =>
      toWorkspaceRequest({
        id: typeof request.id === 'string' ? request.id : undefined,
        name: String(request.name || 'Imported request'),
        method: normalizeMethod(request.method),
        url: String(request.url || ''),
        headers:
          request.headers && typeof request.headers === 'object'
            ? Object.fromEntries(
                Object.entries(request.headers as Record<string, unknown>).map(([key, value]) => [
                  key,
                  String(value),
                ]),
              )
            : {},
        body: String(request.body || ''),
        query:
          request.query && typeof request.query === 'object'
            ? Object.fromEntries(
                Object.entries(request.query as Record<string, unknown>).map(([key, value]) => [
                  key,
                  String(value),
                ]),
              )
            : undefined,
        environment: typeof request.environment === 'string' ? request.environment : undefined,
        createdAt: typeof request.createdAt === 'number' ? request.createdAt : Date.now(),
        updatedAt: typeof request.updatedAt === 'number' ? request.updatedAt : Date.now(),
      }),
    )
}

export function exportPostmanCollection(requests: WorkspaceRequest[], name: string): string {
  const items = requests.map((request) => {
    const safe = toWorkspaceRequest(request)
    return {
      name: safe.name,
      request: {
        method: safe.method,
        header: Object.entries(safe.headers).map(([key, value]) => ({ key, value })),
        ...(safe.body
          ? {
              body: {
                mode: 'raw',
                raw: safe.body,
                options: { raw: { language: 'json' } },
              },
            }
          : {}),
        url: safe.url,
      },
    }
  })

  return JSON.stringify(
    {
      info: {
        name,
        schema: 'https://schema.getpostman.com/json/collection/v2.1.0/collection.json',
      },
      item: items,
    },
    null,
    2,
  )
}

function parseJsonInput(input: string | unknown): unknown {
  if (typeof input !== 'string') return input
  return JSON.parse(input)
}

function normalizeMethod(value: unknown): HttpMethod {
  const method = String(value || 'GET').toUpperCase() as HttpMethod
  return HTTP_METHODS.includes(method) ? method : 'GET'
}

function normalizeUrl(value: unknown): string {
  if (typeof value === 'string') return value
  if (value && typeof value === 'object' && typeof (value as { raw?: unknown }).raw === 'string') {
    return (value as { raw: string }).raw
  }
  return ''
}

function flattenPostmanItems(items: unknown[]): Array<Record<string, unknown>> {
  const flattened: Array<Record<string, unknown>> = []
  for (const item of items) {
    if (!item || typeof item !== 'object') continue
    const record = item as Record<string, unknown>
    if (Array.isArray(record.item)) {
      flattened.push(...flattenPostmanItems(record.item))
    }
    if (record.request && typeof record.request === 'object') {
      flattened.push(record)
    }
  }
  return flattened
}

export function importPostmanCollection(input: string | unknown): WorkspaceRequest[] {
  const parsed = parseJsonInput(input) as { item?: unknown[] }
  if (!parsed || !Array.isArray(parsed.item)) {
    throw new Error('Invalid Postman collection')
  }

  return flattenPostmanItems(parsed.item).map((item) => {
    const request = item.request as Record<string, unknown>
    const headers = Array.isArray(request.header)
      ? Object.fromEntries(
          request.header
            .filter((header): header is { key: string; value?: unknown } => {
              return Boolean(
                header &&
                  typeof header === 'object' &&
                  typeof (header as { key?: unknown }).key === 'string',
              )
            })
            .map((header) => [header.key, String(header.value ?? '')]),
        )
      : {}
    const body =
      request.body &&
      typeof request.body === 'object' &&
      typeof (request.body as { raw?: unknown }).raw === 'string'
        ? (request.body as { raw: string }).raw
        : ''

    return toWorkspaceRequest({
      name: String(item.name || request.method || 'Imported request'),
      method: normalizeMethod(request.method),
      url: normalizeUrl(request.url),
      headers,
      body,
      createdAt: Date.now(),
      updatedAt: Date.now(),
    })
  })
}

export function importOpenApiCollection(input: string | unknown): WorkspaceRequest[] {
  const parsed = parseJsonInput(input) as {
    servers?: Array<{ url?: string }>
    paths?: Record<string, Record<string, unknown>>
  }
  if (!parsed || !parsed.paths || typeof parsed.paths !== 'object') {
    throw new Error('Invalid OpenAPI document')
  }

  const baseUrl = parsed.servers?.find((server) => server.url)?.url?.replace(/\/$/, '') || ''
  const requests: WorkspaceRequest[] = []

  for (const [path, pathItem] of Object.entries(parsed.paths)) {
    for (const method of HTTP_METHODS) {
      const operation = pathItem[method.toLocaleLowerCase()]
      if (!operation || typeof operation !== 'object') continue
      const definition = operation as {
        operationId?: string
        summary?: string
        security?: unknown[]
      }
      const headers: Record<string, string> = {}
      if (Array.isArray(definition.security) && definition.security.length > 0) {
        headers.Authorization = REDACTED_VALUE
      }
      requests.push(
        toWorkspaceRequest({
          name: definition.summary || definition.operationId || method + ' ' + path,
          method,
          url: baseUrl + path,
          headers,
          body: method === 'GET' || method === 'DELETE' ? '' : '{}',
          createdAt: Date.now(),
          updatedAt: Date.now(),
        }),
      )
    }
  }

  return requests
}

export function sanitizeImportedHeaders(headers: Record<string, string>): Record<string, string> {
  return sanitizeHeaders(headers)
}

export function sanitizeImportedBody(body: string): string {
  return sanitizeBody(body)
}
