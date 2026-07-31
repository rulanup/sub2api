import { describe, expect, it } from 'vitest'
import {
  applyTheme,
  DEFAULT_THEME_ID,
  getThemePreset,
  isValidHexColor,
  paletteFromBase,
  THEME_PRESETS,
} from '@/utils/theme'

describe('paletteFromBase', () => {
  it('derives an 11-step palette from a hex color', () => {
    const palette = paletteFromBase('#14b8a6')
    expect(palette['--color-primary-500']).toBe('20 184 166')
    for (const step of ['50', '100', '200', '300', '400', '500', '600', '700', '800', '900', '950']) {
      expect(palette[`--color-primary-${step}` as keyof typeof palette]).toMatch(/^\d+ \d+ \d+$/)
    }
  })

  it('supports 3-digit shorthand hex', () => {
    const palette = paletteFromBase('#fff')
    expect(palette['--color-primary-500']).toBe('255 255 255')
  })

  it('throws on invalid hex', () => {
    expect(() => paletteFromBase('#12')).toThrow('invalid hex color')
    expect(() => paletteFromBase('teal')).toThrow('invalid hex color')
  })

  it('produces lighter 50 and darker 950 than the base', () => {
    const palette = paletteFromBase('#6366f1')
    const base = [0x63, 0x66, 0xf1]
    const light = palette['--color-primary-50'].split(' ').map(Number)
    const dark = palette['--color-primary-950'].split(' ').map(Number)
    for (let i = 0; i < 3; i++) {
      expect(light[i]).toBeGreaterThanOrEqual(base[i])
      expect(dark[i]).toBeLessThanOrEqual(base[i])
    }
  })
})

describe('THEME_PRESETS', () => {
  it('includes solid and gradient themes with unique ids', () => {
    const ids = THEME_PRESETS.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(THEME_PRESETS.some((p) => p.kind === 'solid')).toBe(true)
    expect(THEME_PRESETS.some((p) => p.kind === 'gradient')).toBe(true)
    expect(THEME_PRESETS.some((p) => p.id === DEFAULT_THEME_ID)).toBe(true)
  })

  it('every preset has a palette and swatch', () => {
    for (const preset of THEME_PRESETS) {
      expect(preset.palette['--color-primary-500']).toMatch(/^\d+ \d+ \d+$/)
      expect(preset.swatch[0]).toMatch(/^#/)
    }
  })
})

describe('applyTheme', () => {
  it('applies a preset and sets data-theme', () => {
    document.documentElement.dataset.theme = ''
    applyTheme('blue')
    expect(document.documentElement.dataset.theme).toBe('blue')
    expect(document.documentElement.style.getPropertyValue('--color-primary-500')).not.toBe('')
  })

  it('applies a custom color and marks data-theme=custom', () => {
    applyTheme('custom', '#ff0000')
    expect(document.documentElement.dataset.theme).toBe('custom')
    expect(document.documentElement.style.getPropertyValue('--color-primary-500')).toBe('255 0 0')
  })

  it('falls back to the default theme for unknown ids', () => {
    applyTheme('not-a-theme')
    expect(document.documentElement.dataset.theme).toBe(DEFAULT_THEME_ID)
  })
})

describe('getThemePreset', () => {
  it('resolves known ids', () => {
    expect(getThemePreset('teal')?.kind).toBe('solid')
    expect(getThemePreset('ocean')?.kind).toBe('gradient')
    expect(getThemePreset('nope')).toBeUndefined()
  })
})

describe('isValidHexColor', () => {
  it('accepts 3/6-digit hex with or without #', () => {
    expect(isValidHexColor('#FFFFFF')).toBe(true)
    expect(isValidHexColor('FFFFFF')).toBe(true)
    expect(isValidHexColor('#fff')).toBe(true)
    expect(isValidHexColor('#14b8a6')).toBe(true)
  })

  it('rejects invalid formats', () => {
    expect(isValidHexColor('#GGGGGG')).toBe(false)
    expect(isValidHexColor('#12345')).toBe(false)
    expect(isValidHexColor('teal')).toBe(false)
    expect(isValidHexColor('')).toBe(false)
  })
})
