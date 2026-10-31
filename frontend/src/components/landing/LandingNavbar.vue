<template>
  <header
    class="landing-header"
    :class="{ 'is-scrolled': isScrolled }"
    :data-scrolled="isScrolled"
    data-testid="landing-navbar"
  >
    <div class="landing-container landing-header__inner">
      <a class="landing-wordmark" href="#top" :aria-label="siteName">
        {{ siteName }}
      </a>

      <nav class="landing-header__nav" :aria-label="t('home.landing.nav.primary')">
        <a
          class="landing-text-link landing-header__docs"
          :href="docsUrl"
          target="_blank"
          rel="noopener noreferrer"
        >
          {{ t('home.landing.nav.docs') }}
        </a>

        <LandingPreferences
          :theme-mode="themeMode"
          :set-theme-mode="setThemeMode"
        />

        <RouterLink
          class="landing-cta landing-header__cta"
          :to="startPath"
          data-testid="start-link"
        >
          <span>{{ t('home.landing.actions.start') }}</span>
          <svg class="landing-arrow" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path d="M4 12 12 4M6 4h6v6" stroke="currentColor" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </RouterLink>
      </nav>
    </div>
  </header>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LandingPreferences from './LandingPreferences.vue'

type ThemeMode = 'light' | 'dark' | 'system'

defineProps<{
  siteName: string
  docsUrl: string
  startPath: string
  themeMode: ThemeMode
  setThemeMode: (mode: ThemeMode) => void
}>()

const { t } = useI18n()
const isScrolled = ref(false)
let frameId = 0

function updateScrolledState() {
  frameId = 0
  const next = window.scrollY > 16
  if (next !== isScrolled.value) isScrolled.value = next
}

function scheduleScrolledState() {
  if (frameId) return
  if (typeof window.requestAnimationFrame !== 'function') {
    updateScrolledState()
    return
  }
  frameId = window.requestAnimationFrame(updateScrolledState)
}

onMounted(() => {
  window.addEventListener('scroll', scheduleScrolledState, { passive: true })
  updateScrolledState()
})

onBeforeUnmount(() => {
  if (frameId) window.cancelAnimationFrame(frameId)
  window.removeEventListener('scroll', scheduleScrolledState)
})
</script>
