<template>
  <div class="dev-page dev-page--wide dev-workspace-page">
    <header class="dev-workspace-heading">
      <div>
        <p class="dev-eyebrow">{{ t('developer.workspace.eyebrow') }}</p>
        <h1 class="dev-page-title">{{ t('developer.workspace.title') }}</h1>
        <p class="dev-page-subtitle">{{ t('developer.workspace.subtitle') }}</p>
      </div>
      <div class="dev-workspace-actions">
        <button type="button" class="dev-primary-button" @click="newRequest">
          {{ t('developer.workspace.newRequest') }}
        </button>
        <button type="button" class="dev-secondary-button" @click="fileInput?.click()">
          {{ t('developer.workspace.import') }}
        </button>
        <button type="button" class="dev-danger-button" @click="clearAll">
          {{ t('developer.workspace.clearAll') }}
        </button>
        <input
          ref="fileInput"
          class="dev-visually-hidden"
          type="file"
          accept=".json,application/json"
          @change="handleImport"
        />
      </div>
    </header>

    <p class="dev-workspace-notice">{{ t('developer.workspace.credentialsNotice') }}</p>
    <p v-if="message" class="dev-form-message" :class="messageKind === 'error' ? 'dev-form-message--error' : 'dev-form-message--success'" role="status">
      {{ message }}
    </p>

    <div class="dev-workspace-layout">
      <aside class="dev-workspace-list">
        <div class="dev-panel-heading">
          <h2>{{ t('developer.workspace.requests') }}</h2>
          <span>{{ requests.length }}</span>
        </div>
        <div v-if="requests.length" class="dev-workspace-list__items">
          <button
            v-for="request in requests"
            :key="request.id"
            type="button"
            class="dev-workspace-list__item"
            :class="{ 'is-active': draft.id === request.id }"
            @click="selectRequest(request)"
          >
            <span class="dev-method" :class="'dev-method--' + request.method.toLocaleLowerCase()">
              {{ request.method }}
            </span>
            <span>
              <strong>{{ request.name }}</strong>
              <small>{{ request.url }}</small>
            </span>
          </button>
        </div>
        <p v-else class="dev-workspace-empty">{{ t('developer.workspace.empty') }}</p>

        <div class="dev-history">
          <p>{{ t('developer.workspace.history') }}</p>
          <template v-if="history.length">
            <RouterLink
              v-for="record in history.slice(0, 8)"
              :key="record.historyId"
              :to="{ path: '/developer/playground', query: { requestId: record.id } }"
            >
              <span>{{ record.method }}</span>
              <strong>{{ record.name }}</strong>
            </RouterLink>
          </template>
          <small v-else>{{ t('developer.workspace.emptyHistory') }}</small>
        </div>
      </aside>

      <section class="dev-workspace-editor">
        <div class="dev-editor-heading">
          <div>
            <p>{{ draft.id ? draft.id.slice(0, 8) : 'draft' }}</p>
            <h2>{{ draft.name || t('developer.workspace.newRequest') }}</h2>
          </div>
          <div class="dev-workspace-export">
            <button type="button" class="dev-secondary-button" @click="exportJson">
              {{ t('developer.workspace.exportJson') }}
            </button>
            <button type="button" class="dev-secondary-button" @click="exportPostman">
              {{ t('developer.workspace.exportPostman') }}
            </button>
          </div>
        </div>

        <div class="dev-workspace-form">
          <label class="dev-field">
            <span>{{ t('developer.workspace.name') }}</span>
            <input v-model="draft.name" />
          </label>
          <div class="dev-workspace-form__row">
            <label class="dev-field">
              <span>{{ t('developer.playground.method') }}</span>
              <select v-model="draft.method">
                <option v-for="entry in HTTP_METHODS" :key="entry" :value="entry">{{ entry }}</option>
              </select>
            </label>
            <label class="dev-field">
              <span>{{ t('developer.workspace.environment') }}</span>
              <select v-model="draft.environment">
                <option value="development">{{ t('developer.workspace.development') }}</option>
                <option value="staging">{{ t('developer.workspace.staging') }}</option>
                <option value="production">{{ t('developer.workspace.production') }}</option>
              </select>
            </label>
          </div>
          <label class="dev-field">
            <span>URL</span>
            <input v-model="draft.url" type="url" spellcheck="false" />
          </label>
          <div class="dev-editor-grid">
            <label class="dev-field">
              <span>{{ t('developer.playground.headers') }}</span>
              <textarea v-model="draft.headersText" rows="7" spellcheck="false"></textarea>
            </label>
            <label class="dev-field">
              <span>{{ t('developer.playground.query') }}</span>
              <textarea v-model="draft.queryText" rows="7" spellcheck="false"></textarea>
            </label>
          </div>
          <label class="dev-field">
            <span>{{ t('developer.playground.body') }}</span>
            <textarea v-model="draft.body" rows="12" spellcheck="false"></textarea>
          </label>
        </div>

        <div class="dev-workspace-editor__actions">
          <button type="button" class="dev-primary-button" @click="save">
            {{ t('developer.common.save') }}
          </button>
          <RouterLink
            v-if="draft.id"
            class="dev-secondary-button"
            :to="{ path: '/developer/playground', query: { requestId: draft.id } }"
          >
            {{ t('developer.workspace.openPlayground') }}
          </RouterLink>
          <button v-if="draft.id" type="button" class="dev-danger-button" @click="remove">
            {{ t('developer.common.delete') }}
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { useAppStore } from '@/stores'
import {
  exportDeveloperWorkspace,
  exportPostmanCollection,
  importDeveloperWorkspace,
  importOpenApiCollection,
  importPostmanCollection,
} from '../workspaceIO'
import { parseJsonObject } from '../security'
import {
  clearDeveloperData,
  deleteWorkspaceRequest,
  listRequestHistory,
  listWorkspaceRequests,
  saveWorkspaceRequest,
  type RequestHistoryRecord,
} from '../storage'
import type { HttpMethod, WorkspaceRequest } from '../types'

const HTTP_METHODS: HttpMethod[] = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']
const { t } = useI18n()
const appStore = useAppStore()
const requests = ref<WorkspaceRequest[]>([])
const history = ref<RequestHistoryRecord[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const message = ref('')
const messageKind = ref<'success' | 'error'>('success')
const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)
const draft = reactive({
  id: '',
  name: '',
  method: 'GET' as HttpMethod,
  url: 'https://your-api-host/v1/models',
  environment: 'development',
  headersText: '{\n  "Content-Type": "application/json"\n}',
  queryText: '{}',
  body: '',
  createdAt: 0,
  updatedAt: 0,
})

function createId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return 'workspace-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 8)
}

function setMessage(value: string, kind: 'success' | 'error' = 'success'): void {
  message.value = value
  messageKind.value = kind
}

function resetDraft(): void {
  draft.id = createId()
  draft.name = ''
  draft.method = 'GET'
  draft.url = 'https://your-api-host/v1/models'
  draft.environment = 'development'
  draft.headersText = '{\n  "Content-Type": "application/json"\n}'
  draft.queryText = '{}'
  draft.body = ''
  draft.createdAt = Date.now()
  draft.updatedAt = Date.now()
}

function newRequest(): void {
  resetDraft()
  message.value = ''
}

function selectRequest(request: WorkspaceRequest): void {
  draft.id = request.id
  draft.name = request.name
  draft.method = request.method
  draft.url = request.url
  draft.environment = request.environment || 'development'
  draft.headersText = JSON.stringify(request.headers, null, 2)
  draft.queryText = JSON.stringify(request.query || {}, null, 2)
  draft.body = request.body
  draft.createdAt = request.createdAt
  draft.updatedAt = request.updatedAt
}

function toWorkspaceRequest(): WorkspaceRequest {
  return {
    id: draft.id || createId(),
    name: draft.name.trim() || draft.method + ' ' + draft.url,
    method: draft.method,
    url: draft.url.trim(),
    environment: draft.environment,
    headers: parseJsonObject(draft.headersText) as Record<string, string>,
    query: parseJsonObject(draft.queryText) as Record<string, string>,
    body: draft.body,
    createdAt: draft.createdAt || Date.now(),
    updatedAt: Date.now(),
  }
}

async function refresh(): Promise<void> {
  try {
    requests.value = await listWorkspaceRequests()
    history.value = await listRequestHistory()
  } catch {
    setMessage(t('developer.workspace.importError'), 'error')
  }
}

async function save(): Promise<void> {
  try {
    const saved = await saveWorkspaceRequest(toWorkspaceRequest())
    selectRequest(saved)
    setMessage(t('developer.workspace.saved'))
    await refresh()
  } catch {
    setMessage(t('developer.playground.invalidJson'), 'error')
  }
}

async function remove(): Promise<void> {
  if (!draft.id || !window.confirm(t('developer.workspace.clearConfirm'))) return
  await deleteWorkspaceRequest(draft.id)
  setMessage(t('developer.workspace.deleted'))
  resetDraft()
  await refresh()
}

async function clearAll(): Promise<void> {
  if (!window.confirm(t('developer.workspace.clearConfirm'))) return
  await clearDeveloperData()
  requests.value = []
  history.value = []
  resetDraft()
}

function downloadText(filename: string, content: string): void {
  const blob = new Blob([content], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
  setMessage(t('developer.workspace.exportReady'))
}

function exportJson(): void {
  downloadText('developer-workspace.json', exportDeveloperWorkspace(requests.value))
}

function exportPostman(): void {
  downloadText(
    'developer-postman.json',
    exportPostmanCollection(requests.value, siteName.value),
  )
}

async function persistImported(imported: WorkspaceRequest[]): Promise<void> {
  for (const request of imported) {
    await saveWorkspaceRequest({ ...request, id: createId(), updatedAt: Date.now() })
  }
  await refresh()
  setMessage(t('developer.workspace.imported'))
}

async function handleImport(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const text = await file.text()
    const parsed = JSON.parse(text) as Record<string, unknown>
    if (Array.isArray(parsed.requests) || parsed.schema === 'sub2api.developer-workspace/v1') {
      await persistImported(importDeveloperWorkspace(parsed))
    } else if (parsed.openapi || parsed.swagger) {
      await persistImported(importOpenApiCollection(parsed))
    } else if (parsed.info && Array.isArray(parsed.item)) {
      await persistImported(importPostmanCollection(parsed))
    } else {
      throw new Error('Unknown format')
    }
  } catch {
    setMessage(t('developer.workspace.importError'), 'error')
  } finally {
    input.value = ''
  }
}

onMounted(async () => {
  resetDraft()
  await refresh()
  if (requests.value[0]) selectRequest(requests.value[0])
})
</script>
