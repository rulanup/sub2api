<template>
  <Teleport to="body">
    <div v-if="open" class="dev-search-backdrop" @pointerdown.self="$emit('close')">
      <section
        ref="dialog"
        class="dev-search-dialog"
        role="dialog"
        aria-modal="true"
        :aria-label="t('developer.search.title')"
        @keydown.esc.prevent="$emit('close')"
      >
        <div class="dev-search-dialog__input-row">
          <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
            <circle cx="8.5" cy="8.5" r="4.75" stroke="currentColor" stroke-width="1.4" />
            <path d="m12 12 4 4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
          </svg>
          <input
            ref="input"
            v-model="query"
            type="search"
            autocomplete="off"
            :placeholder="t('developer.search.placeholder')"
            :aria-label="t('developer.search.title')"
            @keydown.down.prevent="moveSelection(1)"
            @keydown.up.prevent="moveSelection(-1)"
            @keydown.enter.prevent="openSelection"
          />
          <button type="button" class="dev-search-dialog__close" :aria-label="t('developer.common.close')" @click="$emit('close')">
            Esc
          </button>
        </div>

        <div class="dev-search-dialog__filters">
          <label>
            <span>{{ t('developer.common.category') }}</span>
            <select v-model="category">
              <option value="all">{{ t('developer.common.all') }}</option>
              <option value="start">{{ t('developer.categories.start') }}</option>
              <option value="guides">{{ t('developer.categories.guides') }}</option>
              <option value="reference">{{ t('developer.categories.reference') }}</option>
              <option value="operations">{{ t('developer.categories.operations') }}</option>
              <option value="chat">Chat</option>
              <option value="responses">Responses</option>
              <option value="models">Models</option>
              <option value="images">Images</option>
              <option value="billing">Billing</option>
              <option value="gemini">Gemini</option>
            </select>
          </label>
          <label>
            <span>{{ t('developer.common.version') }}</span>
            <select v-model="version">
              <option value="all">{{ t('developer.common.all') }}</option>
              <option value="v1">v1</option>
              <option value="v1beta">v1beta</option>
            </select>
          </label>
        </div>

        <div class="dev-search-dialog__body">
          <template v-if="query.trim()">
            <div v-if="results.length" class="dev-search-results" role="listbox" :aria-label="t('developer.search.results')">
              <button
                v-for="(result, index) in results"
                :key="result.kind + '-' + result.id"
                type="button"
                class="dev-search-result"
                :class="{ 'is-active': activeIndex === index }"
                role="option"
                :aria-selected="activeIndex === index"
                @mouseenter="activeIndex = index"
                @click="go(result.path)"
              >
                <span class="dev-search-result__kind">{{ result.kind === 'endpoint' ? 'API' : 'DOC' }}</span>
                <span class="dev-search-result__copy">
                  <strong>{{ result.title }}</strong>
                  <small>{{ result.description }}</small>
                </span>
              </button>
            </div>
            <div v-else class="dev-search-empty">
              <strong>{{ t('developer.search.noResults') }}</strong>
              <span>{{ t('developer.search.tryAgain') }}</span>
            </div>
          </template>

          <template v-else>
            <p class="dev-search-dialog__hint">{{ t('developer.search.hint') }}</p>
            <div class="dev-search-suggestions">
              <button
                v-for="document in suggestions"
                :key="document.id"
                type="button"
                class="dev-search-suggestion"
                @click="go('/developer/docs/' + document.slug)"
              >
                <span>{{ brandText(document.title[locale], siteName) }}</span>
                <small>{{ brandText(document.description[locale], siteName) }}</small>
              </button>
            </div>
          </template>
        </div>

        <footer class="dev-search-dialog__footer">
          <span>↑ ↓</span>
          <span>{{ t('developer.search.results') }}</span>
          <span>Enter</span>
          <span>{{ t('developer.common.open') }}</span>
        </footer>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

import { useAppStore } from '@/stores'
import { brandText } from '../branding'
import { developerDocuments } from '../content'
import { searchDeveloperContent } from '../search'
import type { DeveloperSearchResult, PortalLocale } from '../types'

const props = defineProps<{
  open: boolean
  locale: PortalLocale
}>()

const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const input = ref<HTMLInputElement | null>(null)
const dialog = ref<HTMLElement | null>(null)
const query = ref('')
const category = ref('all')
const version = ref('all')
const activeIndex = ref(0)
let previousOverflow = ''

const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)

const suggestions = computed(() =>
  ['getting-started', 'authentication', 'errors-rate-limits']
    .map((id) => developerDocuments.find((document) => document.id === id))
    .filter((document): document is NonNullable<typeof document> => Boolean(document)),
)

const results = computed<DeveloperSearchResult[]>(() =>
  searchDeveloperContent(query.value, {
    locale: props.locale,
    siteName: siteName.value,
    category: category.value,
    version: version.value,
    limit: 12,
  }),
)

function moveSelection(direction: number): void {
  if (!results.value.length) return
  activeIndex.value =
    (activeIndex.value + direction + results.value.length) % results.value.length
}

function openSelection(): void {
  const result = results.value[activeIndex.value]
  if (result) go(result.path)
}

async function go(path: string): Promise<void> {
  emit('close')
  await router.push(path)
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      previousOverflow = document.body.style.overflow
      document.body.style.overflow = 'hidden'
      await nextTick()
      input.value?.focus()
      return
    }
    document.body.style.overflow = previousOverflow
  },
  { immediate: true },
)

watch([query, category, version], () => {
  activeIndex.value = 0
})

function onFocusIn(event: FocusEvent): void {
  if (dialog.value && !dialog.value.contains(event.target as Node)) {
    input.value?.focus()
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) document.addEventListener('focusin', onFocusIn)
    else document.removeEventListener('focusin', onFocusIn)
  },
)

onBeforeUnmount(() => {
  document.body.style.overflow = previousOverflow
  document.removeEventListener('focusin', onFocusIn)
})
</script>
