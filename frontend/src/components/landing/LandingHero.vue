<template>
  <section id="hero" class="landing-hero" data-act="hero" aria-labelledby="landing-hero-title">
    <div class="landing-container landing-hero__content">
      <p class="landing-hero__identity">
        <span>{{ siteName }}</span>
        <span class="landing-hero__identity-separator" aria-hidden="true">&#183;</span>
        <span>{{ t('home.landing.hero.identitySuffix') }}</span>
      </p>
      <h1 id="landing-hero-title" class="landing-hero__title">
        {{ t('home.landing.hero.title') }}
      </h1>
      <p class="landing-hero__subtitle">
        {{ t('home.landing.hero.subtitle') }}
      </p>
      <RouterLink class="landing-cta landing-hero__cta" :to="startPath" data-testid="start-link">
        <span>{{ t('home.landing.actions.start') }}</span>
        <svg class="landing-arrow" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <path d="M4 12 12 4M6 4h6v6" stroke="currentColor" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </RouterLink>
    </div>

    <a
      class="landing-scroll-cue"
      href="#optical"
      :aria-label="t('home.landing.hero.scrollHint')"
      @click="scrollToOptical"
    >
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M12 5v13m-5-5 5 5 5-5" stroke="currentColor" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </a>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  siteName: string
  startPath: string
}>()
const { t } = useI18n()

function scrollToOptical(event: MouseEvent) {
  const target = document.getElementById('optical')
  if (!target) return
  event.preventDefault()
  const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
  target.scrollIntoView({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'start' })
  history.replaceState(history.state, '', '#optical')
}
</script>
