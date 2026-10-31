import { afterEach, describe, expect, it, vi } from 'vitest'

import { copyText } from '../clipboard'

describe('developer clipboard helper', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('falls back to execCommand when the Clipboard API is denied', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: vi.fn().mockRejectedValue(new Error('Permission denied')),
      },
    })
    const execCommand = vi.fn().mockReturnValue(true)
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: execCommand,
    })

    await copyText('Authorization: Bearer test')

    expect(execCommand).toHaveBeenCalledWith('copy')
  })
})
