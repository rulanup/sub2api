import { onBeforeUnmount, onMounted, ref } from 'vue'

export type DeveloperThemeMode = 'light' | 'dark' | 'system'

const THEME_PREFERENCE_KEY = 'sub2api_theme_preference'
const LEGACY_THEME_KEY = 'theme'

function isThemeMode(value: string | null): value is DeveloperThemeMode {
  return value === 'light' || value === 'dark' || value === 'system'
}

export function useDeveloperTheme() {
  const mode = ref<DeveloperThemeMode>('system')
  const isDark = ref(false)
  let mediaQuery: MediaQueryList | null = null

  function resolveDark(nextMode: DeveloperThemeMode): boolean {
    if (nextMode === 'dark') return true
    if (nextMode === 'light') return false
    return mediaQuery?.matches ?? false
  }

  function apply(nextMode: DeveloperThemeMode, persist: boolean): void {
    mode.value = nextMode
    const dark = resolveDark(nextMode)
    isDark.value = dark
    document.documentElement.classList.toggle('dark', dark)
    document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
    if (persist) {
      localStorage.setItem(THEME_PREFERENCE_KEY, nextMode)
      localStorage.setItem(LEGACY_THEME_KEY, dark ? 'dark' : 'light')
    }
  }

  function setMode(nextMode: DeveloperThemeMode): void {
    apply(nextMode, true)
  }

  function handleSystemChange(): void {
    if (mode.value === 'system') apply('system', true)
  }

  onMounted(() => {
    mediaQuery =
      typeof window.matchMedia === 'function'
        ? window.matchMedia('(prefers-color-scheme: dark)')
        : null
    const savedPreference = localStorage.getItem(THEME_PREFERENCE_KEY)
    const legacyTheme = localStorage.getItem(LEGACY_THEME_KEY)
    const initialMode = isThemeMode(savedPreference)
      ? savedPreference
      : legacyTheme === 'light' || legacyTheme === 'dark'
        ? legacyTheme
        : 'system'

    if (mediaQuery && typeof mediaQuery.addEventListener === 'function') {
      mediaQuery.addEventListener('change', handleSystemChange)
    } else if (mediaQuery && typeof mediaQuery.addListener === 'function') {
      mediaQuery.addListener(handleSystemChange)
    }
    apply(initialMode, false)
  })

  onBeforeUnmount(() => {
    if (!mediaQuery) return
    if (typeof mediaQuery.removeEventListener === 'function') {
      mediaQuery.removeEventListener('change', handleSystemChange)
    } else if (typeof mediaQuery.removeListener === 'function') {
      mediaQuery.removeListener(handleSystemChange)
    }
  })

  return {
    mode,
    isDark,
    setMode,
  }
}
