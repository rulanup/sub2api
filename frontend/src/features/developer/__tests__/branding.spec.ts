import { describe, expect, it } from 'vitest'

import { brandMarkdown, brandText } from '../branding'

describe('developer portal branding', () => {
  it('uses the configured site name in user-facing product text', () => {
    expect(brandText('Sub2API API Key', '归离@API')).toBe('归离@API API Key')
  })

  it('does not rewrite technical host names inside documentation code fences', () => {
    const source = [
      '# Sub2API quick start',
      '',
      '~~~bash',
      'curl https://your-sub2api-host/v1/models',
      'Authorization: Bearer $SUB2API_API_KEY',
      '~~~',
      '',
      'Sub2API keeps this prose branded.',
    ].join('\n')

    const result = brandMarkdown(source, '归离@API')

    expect(result).toContain('# 归离@API quick start')
    expect(result).toContain('curl https://your-sub2api-host/v1/models')
    expect(result).toContain('$SUB2API_API_KEY')
    expect(result).toContain('归离@API keeps this prose branded.')
  })
})
