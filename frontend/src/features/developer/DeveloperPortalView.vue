<template>
  <div class="dev-shell">
    <a class="dev-skip-link" href="#developer-content">{{ t('developer.navigation.skipToContent') }}</a>

    <header class="dev-header">
      <div class="dev-header__inner">
        <RouterLink
          to="/developer"
          class="dev-brand"
          :aria-label="t('developer.portalTitle', { siteName })"
        >
          <span>{{ siteName }}</span>
          <small>{{ t('developer.badge') }}</small>
        </RouterLink>

        <nav class="dev-header__nav" :aria-label="t('developer.navigation.primary')">
          <RouterLink to="/developer/docs/getting-started">{{ t('developer.navigation.docs') }}</RouterLink>
          <RouterLink to="/developer/api">{{ t('developer.navigation.api') }}</RouterLink>
          <RouterLink to="/developer/playground">{{ t('developer.navigation.playground') }}</RouterLink>
          <RouterLink to="/developer/workspace">{{ t('developer.navigation.workspace') }}</RouterLink>
          <RouterLink to="/developer/debug">{{ t('developer.navigation.debug') }}</RouterLink>
        </nav>

        <div class="dev-header__actions">
          <button
            type="button"
            class="dev-search-trigger"
            :aria-label="t('developer.navigation.search')"
            @click="searchOpen = true"
          >
            <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
              <circle cx="8.5" cy="8.5" r="4.75" stroke="currentColor" stroke-width="1.4" />
              <path d="m12 12 4 4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
            </svg>
            <span>{{ t('developer.common.search') }}</span>
            <kbd>Ctrl K</kbd>
          </button>

          <DeveloperPreferences
            :locale="locale"
            :mode="themeMode"
            @update:mode="setMode"
            @change-locale="changeLocale"
          />

          <RouterLink :to="startPath" class="dev-primary-button dev-header__cta">
            {{ t('developer.navigation.start') }}
          </RouterLink>

          <button
            type="button"
            class="dev-icon-button dev-menu-button"
            :aria-label="mobileNavOpen ? t('developer.navigation.close') : t('developer.navigation.menu')"
            :aria-expanded="mobileNavOpen"
            @click="mobileNavOpen = !mobileNavOpen"
          >
            <svg v-if="!mobileNavOpen" viewBox="0 0 20 20" fill="none" aria-hidden="true">
              <path d="M4 6h12M4 10h12M4 14h12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
            <svg v-else viewBox="0 0 20 20" fill="none" aria-hidden="true">
              <path d="m5 5 10 10M15 5 5 15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
          </button>
        </div>
      </div>
    </header>

    <div class="dev-layout">
      <aside class="dev-sidebar" :class="{ 'is-open': mobileNavOpen }">
        <nav :aria-label="t('developer.docs.sidebar')">
          <section class="dev-sidebar__section">
            <p class="dev-sidebar__label">{{ t('developer.badge') }}</p>
            <RouterLink class="dev-sidebar__tool" to="/developer/api">
              <span class="dev-sidebar__tool-mark">API</span>
              <span>{{ t('developer.navigation.api') }}</span>
            </RouterLink>
            <RouterLink class="dev-sidebar__tool" to="/developer/playground">
              <span class="dev-sidebar__tool-mark">↗</span>
              <span>{{ t('developer.navigation.playground') }}</span>
            </RouterLink>
            <RouterLink class="dev-sidebar__tool" to="/developer/workspace">
              <span class="dev-sidebar__tool-mark">⌘</span>
              <span>{{ t('developer.navigation.workspace') }}</span>
            </RouterLink>
            <RouterLink class="dev-sidebar__tool" to="/developer/debug">
              <span class="dev-sidebar__tool-mark">!</span>
              <span>{{ t('developer.navigation.debug') }}</span>
            </RouterLink>
          </section>

          <section v-for="category in developerCategories" :key="category.id" class="dev-sidebar__section">
            <p class="dev-sidebar__label">{{ category[locale] }}</p>
            <RouterLink
              v-for="document in documentsByCategory(category.id)"
              :key="document.id"
              class="dev-sidebar__link"
              :to="'/developer/docs/' + document.slug"
            >
              {{ document.title[locale] }}
            </RouterLink>
          </section>
        </nav>
      </aside>

      <button
        v-if="mobileNavOpen"
        type="button"
        class="dev-sidebar-scrim"
        :aria-label="t('developer.common.close')"
        @click="mobileNavOpen = false"
      ></button>

      <main id="developer-content" class="dev-main" tabindex="-1">
        <RouterView />
      </main>
    </div>

    <footer class="dev-footer">
      <div class="dev-footer__inner">
        <div>
          <strong>{{ siteName }}</strong>
          <span>{{ t('developer.footer.product') }}</span>
        </div>
        <nav :aria-label="t('developer.footer.product')">
          <RouterLink to="/developer/docs/getting-started">{{ t('developer.navigation.docs') }}</RouterLink>
          <RouterLink to="/developer/api">{{ t('developer.navigation.api') }}</RouterLink>
          <a :href="githubUrl" target="_blank" rel="noopener noreferrer">{{ t('developer.navigation.github') }}</a>
          <RouterLink to="/home">{{ t('developer.navigation.home') }}</RouterLink>
        </nav>
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('developer.footer.rights') }}</p>
      </div>
    </footer>

    <DeveloperSearchDialog :open="searchOpen" :locale="locale" @close="searchOpen = false" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

import { useAppStore, useAuthStore } from '@/stores'
import DeveloperPreferences from './components/DeveloperPreferences.vue'
import DeveloperSearchDialog from './components/DeveloperSearchDialog.vue'
import { developerCategories, developerDocuments } from './content'
import type { DocumentationCategory } from './types'
import { useDeveloperLocale } from './useDeveloperLocale'
import { useDeveloperTheme } from './useDeveloperTheme'
import './developer.css'

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { locale, changeLocale } = useDeveloperLocale()
const { mode: themeMode, setMode } = useDeveloperTheme()
const searchOpen = ref(false)
const mobileNavOpen = ref(false)
const githubUrl = 'https://github.com/rulanup/sub2api'
const currentYear = new Date().getFullYear()

const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)
const startPath = computed(() => {
  if (authStore.isAuthenticated) return authStore.isAdmin ? '/admin/dashboard' : '/dashboard'
  const registrationEnabled = appStore.cachedPublicSettings?.registration_enabled === true
  const backendModeEnabled = appStore.cachedPublicSettings?.backend_mode_enabled === true
  return registrationEnabled && !backendModeEnabled ? '/register' : '/login'
})

function documentsByCategory(category: DocumentationCategory) {
  return developerDocuments.filter((document) => document.category === category)
}

function handleGlobalShortcut(event: KeyboardEvent): void {
  if ((event.ctrlKey || event.metaKey) && event.key.toLocaleLowerCase() === 'k') {
    event.preventDefault()
    searchOpen.value = true
  }
}

watch(
  () => route.fullPath,
  () => {
    mobileNavOpen.value = false
  },
)

onMounted(() => {
  document.addEventListener('keydown', handleGlobalShortcut)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleGlobalShortcut)
})
</script>
