<template>
  <footer class="landing-footer">
    <div class="landing-container landing-footer__inner">
      <div class="landing-footer__row">
        <a href="#top" class="landing-footer__brand">{{ siteName }}</a>
        <nav class="landing-footer__nav" :aria-label="t('home.landing.footer.navigation')">
          <a :href="docsUrl" target="_blank" rel="noopener noreferrer">
            {{ t('home.landing.footer.docs') }}
          </a>
          <a :href="githubUrl" target="_blank" rel="noopener noreferrer">GitHub</a>
        </nav>
      </div>
      <p class="landing-footer__copyright">
        &copy; {{ currentYear }} {{ siteName }}. {{ t('home.landing.footer.rights') }}
      </p>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
defineProps<{
  siteName: string
  docsUrl: string
  githubUrl: string
  currentYear: number
}>()
const { t } = useI18n()
</script>
