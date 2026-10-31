import { describe, expect, it } from 'vitest'

import {
  exportDeveloperWorkspace,
  exportPostmanCollection,
  importDeveloperWorkspace,
  importOpenApiCollection,
  importPostmanCollection,
} from '../workspaceIO'

const unsafeRequest = {
  id: 'req-1',
  name: 'Unsafe request',
  method: 'POST',
  url: 'https://api.example.com/v1/chat/completions',
  headers: { Authorization: 'Bearer secret', 'Content-Type': 'application/json' },
  body: '{"api_key":"sk-secret","model":"gpt-4.1"}',
  createdAt: 1,
  updatedAt: 2,
}

describe('developer workspace import and export', () => {
  it('never exports authentication headers or secret body fields', () => {
    const exported = exportDeveloperWorkspace([unsafeRequest])

    expect(exported).not.toContain('Bearer secret')
    expect(exported).not.toContain('sk-secret')
    expect(JSON.parse(exported).requests[0].headers.Authorization).toBe('[REDACTED]')
  })

  it('round-trips safe request metadata through Postman v2.1', () => {
    const postman = exportPostmanCollection([unsafeRequest], 'Sub2API')
    const imported = importPostmanCollection(postman)

    expect(imported).toHaveLength(1)
    expect(imported[0]).toMatchObject({
      name: 'Unsafe request',
      method: 'POST',
      url: 'https://api.example.com/v1/chat/completions',
    })
    expect(JSON.stringify(imported)).not.toContain('Bearer secret')
    expect(JSON.stringify(imported)).not.toContain('sk-secret')
  })

  it('round-trips workspace JSON and sanitizes imported records', () => {
    const exported = exportDeveloperWorkspace([unsafeRequest])
    const imported = importDeveloperWorkspace(exported)

    expect(imported).toHaveLength(1)
    expect(imported[0].name).toBe('Unsafe request')
    expect(JSON.stringify(imported)).not.toContain('Bearer secret')
    expect(JSON.stringify(imported)).not.toContain('sk-secret')
  })

  it('imports OpenAPI operations as sanitized request drafts', () => {
    const imported = importOpenApiCollection({
      openapi: '3.1.0',
      info: { title: 'Example', version: '1.0.0' },
      servers: [{ url: 'https://api.example.com' }],
      paths: {
        '/v1/models': {
          get: {
            operationId: 'listModels',
            summary: 'List models',
            security: [{ bearerAuth: [] }],
          },
        },
      },
    })

    expect(imported).toHaveLength(1)
    expect(imported[0]).toMatchObject({
      name: 'List models',
      method: 'GET',
      url: 'https://api.example.com/v1/models',
    })
    expect(imported[0].headers.Authorization).toBe('[REDACTED]')
  })
})
