<template>
  <section id="start" class="landing-final-cta" data-act="final" aria-labelledby="landing-final-title">
    <div class="landing-container landing-final-cta__inner">
      <h2 id="landing-final-title">
        <span class="landing-final-cta__lead">{{ t('home.landing.finalCta.titleLead') }}</span>
        <span class="landing-final-cta__accent">{{ t('home.landing.finalCta.titleAccent') }}</span>
      </h2>
      <RouterLink class="landing-cta landing-final-cta__button" :to="startPath" data-testid="start-link">
        <span>{{ t('home.landing.actions.start') }}</span>
        <svg class="landing-arrow" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <path d="M4 12 12 4M6 4h6v6" stroke="currentColor" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </RouterLink>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
defineProps<{ startPath: string }>()
const { t } = useI18n()
</script>
