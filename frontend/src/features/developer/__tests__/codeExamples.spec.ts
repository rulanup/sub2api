import { describe, expect, it } from 'vitest'

import { getDeveloperEndpoint } from '../content'
import { generateCodeExample } from '../codeExamples'

describe('developer code examples', () => {
  it('generates a cURL example from the registered endpoint contract', () => {
    const endpoint = getDeveloperEndpoint('chat-completions')
    expect(endpoint).toBeDefined()

    const code = generateCodeExample(endpoint!, 'curl')

    expect(code).toContain('curl --request POST')
    expect(code).toContain('/v1/chat/completions')
    expect(code).toContain('Authorization: Bearer $API_KEY')
    expect(code).toContain('"model": "your-model"')
  })

  it('keeps the same endpoint and payload in TypeScript and Go examples', () => {
    const endpoint = getDeveloperEndpoint('responses-create')
    expect(endpoint).toBeDefined()

    const typescript = generateCodeExample(endpoint!, 'typescript')
    const go = generateCodeExample(endpoint!, 'go')

    expect(typescript).toContain('/v1/responses')
    expect(typescript).toContain('method: "POST"')
    expect(typescript).toContain('"input": "Hello"')
    expect(go).toContain('/v1/responses')
    expect(go).toContain('http.MethodPost')
    expect(go).toContain('Bearer ')
  })

  it('uses a placeholder rather than embedding credentials', () => {
    const endpoint = getDeveloperEndpoint('models-list')
    const code = generateCodeExample(endpoint!, 'python')

    expect(code).toContain('API_KEY')
    expect(code).not.toContain('sk-')
  })
})
