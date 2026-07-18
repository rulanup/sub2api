import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const clientTypes = ['claude', 'codex', 'opencode', 'gemini', 'grokbuild'] as const

describe('CC-Switch import locale keys', () => {
  it.each([
    ['zh', zh.keys],
    ['en', en.keys],
  ])('exposes the generic selector copy in %s', (_locale, keys) => {
    expect(keys.ccsClientSelect.title).toBeTypeOf('string')
    expect(keys.ccsClientSelect.description).toBeTypeOf('string')
    expect(keys.ccsImportUnsupported).toBeTypeOf('string')

    for (const clientType of clientTypes) {
      expect(keys.ccsClientSelect.clients[clientType].label).toBeTypeOf('string')
      expect(keys.ccsClientSelect.clients[clientType].description).toBeTypeOf('string')
    }
  })
})
