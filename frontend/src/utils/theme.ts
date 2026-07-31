/**
 * Theme engine: runtime themable primary palette.
 *
 * The Tailwind `primary` palette (50..950) is backed by CSS variables so a
 * theme can be switched at runtime without a rebuild. Themes are either
 * built-in presets (solid color / gradient pairs) or a user-entered base
 * hex color (e.g. `#ffffff`) from which the full palette is derived.
 */

export type ThemeMode = 'light' | 'dark'

export interface ThemePalette {
  '--color-primary-50': string
  '--color-primary-100': string
  '--color-primary-200': string
  '--color-primary-300': string
  '--color-primary-400': string
  '--color-primary-500': string
  '--color-primary-600': string
  '--color-primary-700': string
  '--color-primary-800': string
  '--color-primary-900': string
  '--color-primary-950': string
}

export type ThemeKind = 'solid' | 'gradient'

export interface ThemePreset {
  id: string
  kind: ThemeKind
  label: string
  /** Palette applied via CSS variables (RGB triplets for Tailwind alpha). */
  palette: ThemePalette
  /** Preview swatch colors (hex, for the picker UI). */
  swatch: [string, string?]
}

const THEME_STORAGE_KEY = 'appTheme'

/** Derives an 11-step Tailwind-style palette from a 500-level base color. */
export function paletteFromBase(baseHex: string): ThemePalette {
  const normalize = (hex: string): string => {
    let value = hex.trim().replace(/^#/, '')
    if (/^[0-9a-fA-F]{3}$/.test(value)) {
      value = value
        .split('')
        .map((c) => c + c)
        .join('')
    }
    if (!/^[0-9a-fA-F]{6}$/.test(value)) {
      throw new Error(`invalid hex color: ${hex}`)
    }
    return value.toLowerCase()
  }

  const hex = normalize(baseHex)
  const mix = (other: string, weight: number): string => {
    const a = parseInt(hex, 16)
    const b = parseInt(other.replace(/^#/, ''), 16)
    const mixChannel = (shift: number) => {
      const ca = (a >> shift) & 0xff
      const cb = (b >> shift) & 0xff
      const cc = Math.round(ca + (cb - ca) * weight)
      return cc.toString(16).padStart(2, '0')
    }
    return `#${mixChannel(16)}${mixChannel(8)}${mixChannel(0)}`
  }

  const toTriplet = (color: string): string => {
    const v = normalize(color)
    return `${parseInt(v.slice(0, 2), 16)} ${parseInt(v.slice(2, 4), 16)} ${parseInt(v.slice(4, 6), 16)}`
  }

  // Lighter steps mix toward white; darker steps mix toward black.
  return {
    '--color-primary-50': toTriplet(mix('#ffffff', 0.92)),
    '--color-primary-100': toTriplet(mix('#ffffff', 0.82)),
    '--color-primary-200': toTriplet(mix('#ffffff', 0.62)),
    '--color-primary-300': toTriplet(mix('#ffffff', 0.4)),
    '--color-primary-400': toTriplet(mix('#ffffff', 0.18)),
    '--color-primary-500': toTriplet(`#${hex}`),
    '--color-primary-600': toTriplet(mix('#000000', 0.12)),
    '--color-primary-700': toTriplet(mix('#000000', 0.26)),
    '--color-primary-800': toTriplet(mix('#000000', 0.42)),
    '--color-primary-900': toTriplet(mix('#000000', 0.58)),
    '--color-primary-950': toTriplet(mix('#000000', 0.75)),
  }
}

function preset(id: string, kind: ThemeKind, label: string, base: string, gradientTo?: string): ThemePreset {
  const palette = { ...paletteFromBase(base) }
  if (kind === 'gradient' && gradientTo) {
    // Gradient themes shift the upper steps to a second hue so that existing
    // `from-primary-500 to-primary-600` gradients render a visible color blend.
    const to = paletteFromBase(gradientTo)
    palette['--color-primary-400'] = to['--color-primary-400']
    palette['--color-primary-500'] = to['--color-primary-500']
  }
  return { id, kind, label, palette, swatch: [base, gradientTo] }
}

export const THEME_PRESETS: ThemePreset[] = [
  preset('teal', 'solid', 'Teal', '#14b8a6'),
  preset('blue', 'solid', 'Blue', '#3b82f6'),
  preset('indigo', 'solid', 'Indigo', '#6366f1'),
  preset('violet', 'solid', 'Violet', '#8b5cf6'),
  preset('emerald', 'solid', 'Emerald', '#10b981'),
  preset('rose', 'solid', 'Rose', '#f43f5e'),
  preset('amber', 'solid', 'Amber', '#f59e0b'),
  preset('ocean', 'gradient', 'Ocean', '#0ea5e9', '#4f46e5'),
  preset('sunset', 'gradient', 'Sunset', '#f97316', '#e11d48'),
  preset('forest', 'gradient', 'Forest', '#10b981', '#0d9488'),
  preset('aurora', 'gradient', 'Aurora', '#8b5cf6', '#c026d3'),
]

export const DEFAULT_THEME_ID = 'teal'

export function getThemePreset(id: string): ThemePreset | undefined {
  return THEME_PRESETS.find((p) => p.id === id)
}

function applyPalette(palette: ThemePalette): void {
  const root = document.documentElement
  for (const [key, value] of Object.entries(palette)) {
    root.style.setProperty(key, value)
  }
}

/**
 * Applies a theme preset or a custom base color to the document.
 * `data-theme` is set for debugging/persistence; the palette itself lives in
 * inline CSS variables so it survives a full page reload without a rebuild.
 */
export function applyTheme(id: string, customBaseHex?: string): void {
  const root = document.documentElement
  if (customBaseHex && customBaseHex.trim()) {
    root.dataset.theme = 'custom'
    applyPalette(paletteFromBase(customBaseHex))
    return
  }
  const preset = getThemePreset(id)
  if (!preset) {
    root.dataset.theme = DEFAULT_THEME_ID
    applyPalette(getThemePreset(DEFAULT_THEME_ID)!.palette)
    return
  }
  root.dataset.theme = preset.id
  applyPalette(preset.palette)
}

export interface PersistedTheme {
  id: string
  custom?: string
}

export function loadTheme(): void {
  const raw = localStorage.getItem(THEME_STORAGE_KEY)
  if (!raw) {
    applyTheme(DEFAULT_THEME_ID)
    return
  }
  try {
    const parsed = JSON.parse(raw) as PersistedTheme
    applyTheme(parsed.id, parsed.custom)
  } catch {
    applyTheme(DEFAULT_THEME_ID)
  }
}

export function saveTheme(theme: PersistedTheme): void {
  localStorage.setItem(THEME_STORAGE_KEY, JSON.stringify(theme))
}

export function isValidHexColor(value: string): boolean {
  const v = value.trim().replace(/^#/, '')
  return /^[0-9a-fA-F]{3}$/.test(v) || /^[0-9a-fA-F]{6}$/.test(v)
}
