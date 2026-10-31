import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import { developerEndpoints, developerDocuments } from '../content'
import { searchDeveloperContent } from '../search'

describe('developer registry', () => {
  it('only documents gateway paths that exist in the backend route registry', () => {
    const source = readFileSync(
      resolve(process.cwd(), '../backend/internal/server/routes/gateway.go'),
      'utf8',
    )
    const routePatterns = [
      ...source.matchAll(/(gateway|gemini)\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)"/g),
    ].map((match) => {
      const prefix = match[1] === 'gateway' ? '/v1' : '/v1beta'
      return match[2] + ' ' + prefix + match[3]
    })
    const documented = developerEndpoints.map(
      (endpoint) => endpoint.method + ' ' + endpoint.backendGroup + endpoint.backendPath,
    )

    expect(documented.length).toBeGreaterThanOrEqual(8)
    expect(documented.filter((entry) => !routePatterns.includes(entry))).toEqual([])
  })

  it('gives every document a stable ID and explicit translation state', () => {
    const ids = developerDocuments.map((document) => document.id)

    expect(new Set(ids).size).toBe(ids.length)
    expect(developerDocuments.every((document) => document.translations.zh.status === 'current')).toBe(true)
    expect(developerDocuments.every((document) => document.translations.ja.status === 'missing')).toBe(true)
  })

  it('ranks an exact API path above prose matches', () => {
    const results = searchDeveloperContent('/v1/chat/completions', { locale: 'en' })

    expect(results[0]).toMatchObject({
      kind: 'endpoint',
      title: 'Create chat completion',
    })
  })

  it('finds endpoints by parameter name and honors category filters', () => {
    const parameterResults = searchDeveloperContent('stream', {
      locale: 'en',
      category: 'responses',
    })

    expect(parameterResults.some((result) => result.kind === 'endpoint')).toBe(true)
    expect(parameterResults.every((result) => result.category === 'responses')).toBe(true)
  })

  it('finds endpoints by generated code example content', () => {
    const results = searchDeveloperContent('API_KEY', {
      locale: 'en',
      category: 'chat',
    })

    expect(results.some((result) => result.kind === 'endpoint' && result.id === 'chat-completions')).toBe(true)
  })

  it('suggests related content when a search has no results', () => {
    const response = searchDeveloperContent('zzzz-no-such-developer-result', { locale: 'en' })

    expect(response).toEqual([])
  })
})
