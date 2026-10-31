<template>
  <div class="dev-page dev-page--wide dev-debug-page">
    <header class="dev-debug-heading">
      <div>
        <p class="dev-eyebrow">{{ t('developer.debug.eyebrow') }}</p>
        <h1 class="dev-page-title">{{ t('developer.debug.title') }}</h1>
        <p class="dev-page-subtitle">{{ t('developer.debug.subtitle') }}</p>
      </div>
      <button type="button" class="dev-danger-button" :disabled="!records.length" @click="clearRecords">
        {{ t('developer.debug.clearHistory') }}
      </button>
    </header>

    <p class="dev-workspace-notice">{{ t('developer.debug.redactedNotice') }}</p>

    <div v-if="records.length" class="dev-debug-layout">
      <aside class="dev-debug-list" :aria-label="t('developer.debug.records')">
        <button
          v-for="record in records"
          :key="record.requestId"
          type="button"
          class="dev-debug-list__item"
          :class="{ 'is-active': selected?.requestId === record.requestId }"
          @click="selected = record"
        >
          <span
            class="dev-status"
            :class="record.status && record.status >= 400 ? 'dev-status--danger' : 'dev-status--success'"
          >
            {{ record.status || 'ERR' }}
          </span>
          <span>
            <strong>{{ record.method }} {{ pathname(record.url) }}</strong>
            <small>{{ formatDate(record.createdAt) }} — {{ record.durationMs }} ms</small>
          </span>
        </button>
      </aside>

      <section v-if="selected" class="dev-debug-detail">
        <header class="dev-debug-detail__header">
          <div>
            <span
              class="dev-status"
              :class="selected.status && selected.status >= 400 ? 'dev-status--danger' : 'dev-status--success'"
            >
              {{ selected.status || t('developer.common.error') }}
            </span>
            <h2>{{ selected.method }} {{ pathname(selected.url) }}</h2>
            <code>{{ selected.requestId }}</code>
          </div>
          <button type="button" class="dev-secondary-button" @click="exportReport">
            {{ t('developer.debug.exportReport') }}
          </button>
        </header>

        <section class="dev-debug-overview">
          <div>
            <span>{{ t('developer.playground.requestId') }}</span>
            <strong>{{ selected.requestId }}</strong>
          </div>
          <div>
            <span>{{ t('developer.debug.timing') }}</span>
            <strong>{{ selected.durationMs }} ms</strong>
          </div>
          <div>
            <span>{{ t('developer.common.status') }}</span>
            <strong>{{ selected.status || '—' }}</strong>
          </div>
        </section>

        <section v-if="!selected.status || selected.status >= 400" class="dev-diagnosis-panel">
          <div class="dev-diagnosis-panel__heading">
            <div>
              <p>{{ t('developer.debug.diagnosis') }}</p>
              <h3>{{ diagnosis.title[locale] }}</h3>
            </div>
            <span class="dev-status dev-status--neutral">
              {{ t('developer.debug.confidence') }}:
              {{ t('developer.debug.' + diagnosis.confidence) }}
            </span>
          </div>
          <p>{{ diagnosis.summary[locale] }}</p>
          <div class="dev-diagnosis-steps">
            <p>{{ t('developer.debug.steps') }}</p>
            <ol>
              <li v-for="(step, index) in diagnosis.steps" :key="index">{{ step[locale] }}</li>
            </ol>
          </div>
          <div class="dev-diagnosis-links">
            <span>{{ t('developer.debug.relatedDocs') }}</span>
            <RouterLink
              v-for="docId in diagnosis.relatedDocIds"
              :key="docId"
              :to="'/developer/docs/' + docId"
            >
              {{ documentTitle(docId) }}
            </RouterLink>
          </div>
        </section>
        <section v-else class="dev-diagnosis-panel dev-diagnosis-panel--success">
          <div class="dev-diagnosis-panel__heading">
            <div>
              <p>{{ t('developer.debug.diagnosis') }}</p>
              <h3>{{ t('developer.debug.noFailureTitle') }}</h3>
            </div>
            <span class="dev-status dev-status--success">{{ selected.status }}</span>
          </div>
          <p>{{ t('developer.debug.noFailureBody') }}</p>
        </section>

        <section class="dev-debug-section">
          <h3>{{ t('developer.debug.request') }}</h3>
          <div class="dev-debug-url">{{ selected.method }} {{ selected.url }}</div>
          <DeveloperCodeBlock
            :code="JSON.stringify(selected.requestHeaders, null, 2)"
            language="json"
            label="Headers"
          />
          <DeveloperCodeBlock
            v-if="selected.requestBody"
            :code="selected.requestBody"
            language="json"
            label="Body"
          />
        </section>

        <section class="dev-debug-section">
          <h3>{{ t('developer.debug.response') }}</h3>
          <DeveloperCodeBlock
            :code="JSON.stringify(selected.responseHeaders, null, 2)"
            language="json"
            label="Headers"
          />
          <DeveloperCodeBlock
            v-if="selected.responseBody"
            :code="selected.responseBody"
            language="json"
            label="Body"
          />
        </section>
      </section>
    </div>

    <section v-else class="dev-debug-empty">
      <span>!</span>
      <h2>{{ t('developer.debug.empty') }}</h2>
      <p>{{ t('developer.debug.emptyHint') }}</p>
      <RouterLink class="dev-primary-button" to="/developer/playground">
        {{ t('developer.navigation.playground') }}
      </RouterLink>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { useAppStore } from '@/stores'
import { brandText } from '../branding'
import DeveloperCodeBlock from '../components/DeveloperCodeBlock.vue'
import { developerDocuments } from '../content'
import { classifyRequestFailure, createRedactedDebugReport } from '../debug'
import {
  clearDebugRecords,
  listDebugRecords,
  type DeveloperDebugRecord,
} from '../storage'
import { useDeveloperLocale } from '../useDeveloperLocale'

const { t } = useI18n()
const { locale } = useDeveloperLocale()
const appStore = useAppStore()
const records = ref<DeveloperDebugRecord[]>([])
const selected = ref<DeveloperDebugRecord | null>(null)
const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)

const diagnosis = computed(() =>
  classifyRequestFailure({
    status: selected.value?.status,
    error:
      selected.value?.status && selected.value.status >= 400
        ? new Error('HTTP ' + selected.value.status)
        : selected.value?.responseBody
          ? new Error(selected.value.responseBody)
          : undefined,
  }),
)

function pathname(url: string): string {
  try {
    return new URL(url).pathname
  } catch {
    return url
  }
}

function formatDate(value: number): string {
  return new Intl.DateTimeFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function documentTitle(id: string): string {
  const title = developerDocuments.find((document) => document.id === id)?.title[locale.value]
  return title ? brandText(title, siteName.value) : id
}

async function refresh(): Promise<void> {
  records.value = await listDebugRecords()
  if (!selected.value || !records.value.some((record) => record.requestId === selected.value?.requestId)) {
    selected.value = records.value[0] || null
  }
}

async function clearRecords(): Promise<void> {
  if (!window.confirm(t('developer.debug.clearConfirm'))) return
  await clearDebugRecords()
  records.value = []
  selected.value = null
}

function exportReport(): void {
  if (!selected.value) return
  const content = createRedactedDebugReport({
    requestId: selected.value.requestId,
    method: selected.value.method,
    url: selected.value.url,
    status: selected.value.status,
    durationMs: selected.value.durationMs,
    requestHeaders: selected.value.requestHeaders,
    requestBody: selected.value.requestBody,
    responseHeaders: selected.value.responseHeaders,
    responseBody: selected.value.responseBody,
  })
  const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download =
    'sub2api-debug-' + selected.value.requestId.replace(/[^a-zA-Z0-9_-]/g, '_') + '.md'
  anchor.click()
  URL.revokeObjectURL(url)
}

onMounted(refresh)
</script>
