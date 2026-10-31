<template>
  <div class="dev-page dev-page--wide dev-api-page">
    <header class="dev-api-heading">
      <p class="dev-eyebrow">{{ t('developer.api.eyebrow') }}</p>
      <h1 class="dev-page-title">{{ t('developer.api.title', { siteName }) }}</h1>
      <p class="dev-page-subtitle">{{ t('developer.api.subtitle') }}</p>
    </header>

    <div class="dev-api-toolbar">
      <label class="dev-filter">
        <span>{{ t('developer.api.filter') }}</span>
        <input v-model="query" type="search" :placeholder="t('developer.api.filterPlaceholder')" />
      </label>
      <label class="dev-filter">
        <span>{{ t('developer.api.categories') }}</span>
        <select v-model="category">
          <option value="all">{{ t('developer.common.all') }}</option>
          <option v-for="entry in categories" :key="entry.id" :value="entry.id">
            {{ entry.label }}
          </option>
        </select>
      </label>
    </div>

    <div class="dev-api-layout">
      <aside class="dev-api-list" :aria-label="t('developer.api.categories')">
        <template v-if="filteredEndpoints.length">
          <button
            v-for="endpoint in filteredEndpoints"
            :key="endpoint.id"
            type="button"
            class="dev-api-list__item"
            :class="{ 'is-active': selectedEndpoint?.id === endpoint.id }"
            @click="selectEndpoint(endpoint.id)"
          >
            <span class="dev-method" :class="'dev-method--' + endpoint.method.toLocaleLowerCase()">
              {{ endpoint.method }}
            </span>
            <span>
              <strong>{{ brandText(endpoint.summary[locale], siteName) }}</strong>
              <small>{{ endpoint.path }}</small>
            </span>
          </button>
        </template>
        <p v-else class="dev-api-list__empty">{{ t('developer.api.noResults') }}</p>
      </aside>

      <section v-if="selectedEndpoint" class="dev-api-detail">
        <header class="dev-api-detail__header">
          <div class="dev-api-detail__title">
            <span class="dev-method" :class="'dev-method--' + selectedEndpoint.method.toLocaleLowerCase()">
              {{ selectedEndpoint.method }}
            </span>
            <h2>{{ brandText(selectedEndpoint.summary[locale], siteName) }}</h2>
          </div>
          <p>{{ brandText(selectedEndpoint.description[locale], siteName) }}</p>
          <div class="dev-api-path">
            <code>{{ selectedEndpoint.path }}</code>
            <button type="button" @click="copyPath">{{ t('developer.common.copy') }}</button>
          </div>
          <div class="dev-api-badges">
            <span>{{ selectedEndpoint.version }}</span>
            <span>{{ t('developer.api.auth') }}: {{ t('developer.api.bearer') }}</span>
            <span>{{ t('developer.api.sourceHandler') }}: {{ selectedEndpoint.sourceHandler }}</span>
          </div>
          <RouterLink
            class="dev-primary-button"
            :to="{ path: '/developer/playground', query: { endpoint: selectedEndpoint.id } }"
          >
            {{ t('developer.api.openPlayground') }}
          </RouterLink>
        </header>

        <section class="dev-api-section">
          <h3>{{ t('developer.api.parameters') }}</h3>
          <div class="dev-table-wrap">
            <table class="dev-data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>In</th>
                  <th>Type</th>
                  <th>{{ t('developer.api.parameters') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="parameter in selectedEndpoint.parameters" :key="parameter.name + parameter.location">
                  <td>
                    <code>{{ parameter.name }}</code>
                    <span v-if="parameter.required" class="dev-required">*</span>
                  </td>
                  <td>{{ parameter.location }}</td>
                  <td>{{ parameter.type }}</td>
                  <td>{{ brandText(parameter.description[locale], siteName) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section v-if="selectedEndpoint.requestExample !== null" class="dev-api-section">
          <h3>{{ t('developer.api.requestBody') }}</h3>
          <DeveloperCodeBlock
            :code="requestExample"
            :label="selectedEndpoint.method + ' ' + selectedEndpoint.path"
            language="json"
          />
        </section>

        <section class="dev-api-section">
          <h3>{{ t('developer.api.response') }}</h3>
          <DeveloperCodeBlock :code="responseExample" language="json" label="JSON" />
        </section>

        <section class="dev-api-section">
          <h3>{{ t('developer.api.examples') }}</h3>
          <div class="dev-tabs" role="tablist" :aria-label="t('developer.api.examples')">
            <button
              v-for="entry in CODE_LANGUAGES"
              :key="entry.id"
              type="button"
              role="tab"
              :aria-selected="codeLanguage === entry.id"
              :class="{ 'is-active': codeLanguage === entry.id }"
              @click="codeLanguage = entry.id"
            >
              {{ entry.label }}
            </button>
          </div>
          <DeveloperCodeBlock
            :code="codeExample"
            :language="codeLanguage"
            :label="CODE_LANGUAGES.find((entry) => entry.id === codeLanguage)?.label"
          />
        </section>

        <div class="dev-api-grid">
          <section class="dev-api-section">
            <h3>{{ t('developer.api.errors') }}</h3>
            <div class="dev-error-list">
              <div v-for="error in selectedEndpoint.errors" :key="error.code">
                <span class="dev-status dev-status--danger">{{ error.status }}</span>
                <code>{{ error.code }}</code>
                <p>{{ brandText(error.description[locale], siteName) }}</p>
              </div>
            </div>
          </section>
          <section class="dev-api-section">
            <h3>{{ t('developer.api.rateLimit') }}</h3>
            <p class="dev-api-note">{{ brandText(selectedEndpoint.rateLimit[locale], siteName) }}</p>
          </section>
        </div>
      </section>

      <section v-else class="dev-api-empty">
        <span>⌘</span>
        <h2>{{ t('developer.api.selectEndpoint') }}</h2>
        <p>{{ t('developer.api.selectEndpointHint') }}</p>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

import { useAppStore } from '@/stores'
import { brandText } from '../branding'
import { copyText } from '../clipboard'
import DeveloperCodeBlock from '../components/DeveloperCodeBlock.vue'
import { CODE_LANGUAGES, generateCodeExample, type CodeLanguage } from '../codeExamples'
import { developerEndpoints } from '../content'
import { searchDeveloperContent } from '../search'
import { useDeveloperLocale } from '../useDeveloperLocale'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { locale } = useDeveloperLocale()
const appStore = useAppStore()
const query = ref('')
const category = ref('all')
const codeLanguage = ref<CodeLanguage>('curl')

const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)

const categoryLabels = computed<Record<string, string>>(() => ({
  models: locale.value === 'zh' ? '模型' : 'Models',
  chat: 'Chat',
  responses: 'Responses',
  images: locale.value === 'zh' ? '图片' : 'Images',
  billing: locale.value === 'zh' ? '计费' : 'Billing',
  gemini: 'Gemini',
}))

const categories = computed(() =>
  [...new Set(developerEndpoints.map((endpoint) => endpoint.category))].map((id) => ({
    id,
    label: categoryLabels.value[id] || id,
  })),
)

const selectedEndpoint = computed(() => {
  const id = String(route.query.endpoint || '')
  return developerEndpoints.find((endpoint) => endpoint.id === id)
})

const filteredEndpoints = computed(() => {
  if (!query.value.trim()) {
    return developerEndpoints.filter(
      (endpoint) => category.value === 'all' || endpoint.category === category.value,
    )
  }
  return searchDeveloperContent(query.value, {
    locale: locale.value,
    siteName: siteName.value,
    category: category.value,
    limit: 100,
  })
    .filter((result) => result.kind === 'endpoint')
    .map((result) => developerEndpoints.find((endpoint) => endpoint.id === result.id))
    .filter((endpoint): endpoint is NonNullable<typeof endpoint> => Boolean(endpoint))
})

const requestExample = computed(() => {
  const value = selectedEndpoint.value?.requestExample
  if (value === null || value === undefined) return ''
  return typeof value === 'string' ? value : JSON.stringify(value, null, 2)
})

const responseExample = computed(() =>
  JSON.stringify(selectedEndpoint.value?.responseExample ?? {}, null, 2),
)

const codeExample = computed(() =>
  selectedEndpoint.value ? generateCodeExample(selectedEndpoint.value, codeLanguage.value) : '',
)

async function selectEndpoint(id: string): Promise<void> {
  await router.replace({ query: { ...route.query, endpoint: id } })
}

async function copyPath(): Promise<void> {
  if (selectedEndpoint.value) await copyText(selectedEndpoint.value.path)
}

watch(
  () => route.query.endpoint,
  (id) => {
    if (id && !developerEndpoints.some((endpoint) => endpoint.id === id)) {
      router.replace({ query: { ...route.query, endpoint: developerEndpoints[0]?.id } })
    }
  },
  { immediate: true },
)
</script>
