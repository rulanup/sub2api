<template>
  <div id="top" class="sub2api-landing" data-testid="landing-page">
    <a class="landing-skip-link" href="#main-content">
      {{ t('home.landing.nav.skipToContent') }}
    </a>

    <LandingNavbar
      :site-name="siteName"
      :docs-url="resolvedDocsUrl"
      :start-path="startPath"
      :theme-mode="themeMode"
      :set-theme-mode="setThemeMode"
    />

    <main id="main-content" class="landing-main" tabindex="-1">
      <LandingHero :site-name="siteName" :start-path="startPath" />
      <OpticalWaveArt />
      <LandingBrandStory />
      <LandingFinalCta :start-path="startPath" />
    </main>

    <LandingFooter
      :site-name="siteName"
      :docs-url="resolvedDocsUrl"
      :github-url="githubUrl"
      :current-year="currentYear"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LandingBrandStory from './LandingBrandStory.vue'
import LandingFinalCta from './LandingFinalCta.vue'
import LandingFooter from './LandingFooter.vue'
import LandingHero from './LandingHero.vue'
import LandingNavbar from './LandingNavbar.vue'
import OpticalWaveArt from './OpticalWaveArt.vue'
import './landing.css'

type ThemeMode = 'light' | 'dark' | 'system'

const props = defineProps<{
  siteName: string
  docUrl: string
  githubUrl: string
  currentYear: number
  startPath: string
  themeMode: ThemeMode
  setThemeMode: (mode: ThemeMode) => void
}>()

const { t } = useI18n()
const resolvedDocsUrl = computed(() => props.docUrl || '/developer/docs/getting-started')
</script>
