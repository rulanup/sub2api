import { computed } from 'vue'

import { getLocale, setLocale } from '@/i18n'
import type { PortalLocale } from './types'

export function useDeveloperLocale() {
  const locale = computed<PortalLocale>(() => getLocale())

  async function changeLocale(next: PortalLocale): Promise<void> {
    await setLocale(next)
  }

  return {
    locale,
    changeLocale,
  }
}
