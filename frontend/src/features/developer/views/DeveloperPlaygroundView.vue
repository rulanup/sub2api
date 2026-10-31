<template>
  <div class="dev-page dev-page--wide dev-playground-page">
    <header class="dev-playground-heading">
      <div>
        <p class="dev-eyebrow">{{ t('developer.playground.eyebrow') }}</p>
        <h1 class="dev-page-title">{{ t('developer.playground.title') }}</h1>
        <p class="dev-page-subtitle">{{ t('developer.playground.subtitle') }}</p>
      </div>
      <div class="dev-mode-tabs" role="tablist" :aria-label="t('developer.playground.title')">
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'mock'"
          :class="{ 'is-active': mode === 'mock' }"
          @click="mode = 'mock'"
        >
          {{ t('developer.playground.mock') }}
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'live'"
          :class="{ 'is-active': mode === 'live' }"
          @click="mode = 'live'"
        >
          {{ t('developer.playground.live') }}
        </button>
      </div>
    </header>

    <div v-if="mode === 'mock'" class="dev-inline-notice">
      {{ t('developer.playground.mockNotice') }}
    </div>
    <div v-else class="dev-inline-notice dev-inline-notice--warning">
      {{ t('developer.playground.corsNotice') }}
    </div>

    <div class="dev-playground-layout">
      <section class="dev-request-panel">
        <div class="dev-panel-heading">
          <h2>{{ t('developer.playground.endpoint') }}</h2>
          <select v-model="endpointId" @change="applyEndpoint">
            <option v-for="endpoint in developerEndpoints" :key="endpoint.id" :value="endpoint.id">
              {{ endpoint.method }} {{ endpoint.path }}
            </option>
          </select>
        </div>

        <div class="dev-request-line">
          <select v-model="method" :aria-label="t('developer.playground.method')">
            <option v-for="entry in HTTP_METHODS" :key="entry" :value="entry">{{ entry }}</option>
          </select>
          <input v-model="path" :aria-label="t('developer.playground.path')" />
        </div>

        <label class="dev-field">
          <span>{{ t('developer.playground.baseUrl') }}</span>
          <input v-model="baseUrl" type="url" spellcheck="false" />
        </label>

        <label v-if="mode === 'mock'" class="dev-field">
          <span>{{ t('developer.playground.mockResponse') }}</span>
          <select v-model="mockScenario">
            <option value="success">{{ t('developer.playground.mockSuccess') }}</option>
            <option value="rate-limit">{{ t('developer.playground.mockRateLimit') }}</option>
          </select>
        </label>

        <div v-if="mode === 'live'" class="dev-field">
          <label for="developer-playground-api-key">{{ t('developer.playground.apiKey') }}</label>
          <div class="dev-secret-field">
            <input
              id="developer-playground-api-key"
              v-model="apiKey"
              type="password"
              autocomplete="off"
              spellcheck="false"
              :placeholder="t('developer.playground.apiKeyPlaceholder')"
            />
            <button type="button" @click="clearKey">{{ t('developer.playground.clearKey') }}</button>
          </div>
          <small>{{ t('developer.playground.apiKeyHint') }}</small>
        </div>

        <div class="dev-editor-grid">
          <label class="dev-field">
            <span>{{ t('developer.playground.headers') }}</span>
            <textarea v-model="headersText" rows="5" spellcheck="false"></textarea>
          </label>
          <label class="dev-field">
            <span>{{ t('developer.playground.query') }}</span>
            <textarea v-model="queryText" rows="5" spellcheck="false"></textarea>
          </label>
        </div>

        <label class="dev-field">
          <span>{{ t('developer.playground.body') }}</span>
          <textarea v-model="bodyText" rows="12" spellcheck="false"></textarea>
        </label>

        <label v-if="mode === 'live'" class="dev-confirmation">
          <input v-model="liveConfirmed" type="checkbox" />
          <span>{{ t('developer.playground.billingWarning') }}</span>
        </label>

        <p v-if="formError" class="dev-form-message dev-form-message--error" role="alert">
          {{ formError }}
        </p>
        <p v-if="savedNotice" class="dev-form-message dev-form-message--success" role="status">
          {{ savedNotice }}
        </p>

        <div class="dev-request-actions">
          <button
            v-if="!sending"
            type="button"
            class="dev-primary-button"
            @click="sendRequest"
          >
            {{ mode === 'mock' ? t('developer.playground.send') : t('developer.playground.send') }}
          </button>
          <button v-else type="button" class="dev-danger-button" @click="stopRequest">
            {{ t('developer.playground.stop') }}
          </button>
          <button type="button" class="dev-secondary-button" @click="saveToWorkspace">
            {{ t('developer.playground.saveToWorkspace') }}
          </button>
          <button type="button" class="dev-secondary-button" @click="resetRequest">
            {{ t('developer.playground.reset') }}
          </button>
        </div>
      </section>

      <section class="dev-response-panel" aria-live="polite">
        <div class="dev-panel-heading">
          <h2>{{ t('developer.playground.response') }}</h2>
          <span v-if="response" class="dev-status" :class="response.status < 400 ? 'dev-status--success' : 'dev-status--danger'">
            {{ response.status }} {{ response.statusText }}
          </span>
        </div>

        <div v-if="response" class="dev-response-meta">
          <div>
            <span>{{ t('developer.playground.requestId') }}</span>
            <code>{{ response.requestId }}</code>
          </div>
          <div>
            <span>{{ t('developer.playground.duration') }}</span>
            <strong>{{ response.durationMs }} ms</strong>
          </div>
          <div>
            <span>{{ t('developer.playground.size') }}</span>
            <strong>{{ formatBytes(response.bytes) }}</strong>
          </div>
        </div>

        <div v-if="streamingText" class="dev-stream-output">
          <p>{{ t('developer.playground.streaming') }}</p>
          <pre>{{ streamingText }}</pre>
        </div>

        <DeveloperCodeBlock
          v-if="response"
          :code="response.body"
          language="json"
          :label="t('developer.playground.responseBody')"
        />

        <div v-if="response && Object.keys(response.headers).length" class="dev-response-headers">
          <h3>Headers</h3>
          <dl>
            <template v-for="(value, key) in response.headers" :key="key">
              <dt>{{ key }}</dt>
              <dd>{{ value }}</dd>
            </template>
          </dl>
        </div>

        <div v-if="errorDiagnosis" class="dev-diagnosis-card">
          <div>
            <span class="dev-status dev-status--danger">{{ errorDiagnosis.confidence }}</span>
            <h3>{{ errorDiagnosis.title[locale] }}</h3>
          </div>
          <p>{{ errorDiagnosis.summary[locale] }}</p>
          <RouterLink to="/developer/debug" class="dev-secondary-button">
            {{ t('developer.navigation.debug') }}
          </RouterLink>
        </div>

        <div v-if="!response" class="dev-response-empty">
          <span>{ }</span>
          <p>{{ t('developer.playground.responsePlaceholder') }}</p>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

import { useAppStore } from '@/stores'
import { brandText } from '../branding'
import DeveloperCodeBlock from '../components/DeveloperCodeBlock.vue'
import { developerEndpoints, getDeveloperEndpoint } from '../content'
import { classifyRequestFailure } from '../debug'
import { executeLiveRequest, type LiveRequestResult } from '../liveRequest'
import {
  parseJsonObject,
  resolveDefaultLiveBaseUrl,
  validateLiveTarget,
} from '../security'
import {
  addDebugRecord,
  addRequestHistory,
  getWorkspaceRequest,
  saveWorkspaceRequest,
} from '../storage'
import type { HttpMethod, WorkspaceRequest } from '../types'
import { useDeveloperLocale } from '../useDeveloperLocale'

const HTTP_METHODS: HttpMethod[] = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

const route = useRoute()
const { t } = useI18n()
const { locale } = useDeveloperLocale()
const appStore = useAppStore()
const mode = ref<'mock' | 'live'>('mock')
const mockScenario = ref<'success' | 'rate-limit'>('success')
const endpointId = ref('chat-completions')
const method = ref<HttpMethod>('POST')
const path = ref('/v1/chat/completions')
const baseUrl = ref(resolveDefaultLiveBaseUrl(window.location.origin))
const apiKey = ref('')
const headersText = ref('{\n  "Content-Type": "application/json"\n}')
const queryText = ref('{}')
const bodyText = ref('{}')
const liveConfirmed = ref(false)
const sending = ref(false)
const response = ref<LiveRequestResult | null>(null)
const streamingText = ref('')
const formError = ref('')
const savedNotice = ref('')
const failedError = ref<unknown>(null)
let abortController: AbortController | null = null
let currentRequest: WorkspaceRequest | null = null
let savedTimer = 0

const selectedEndpoint = computed(() => getDeveloperEndpoint(endpointId.value))
const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)
const errorDiagnosis = computed(() =>
  failedError.value
    ? classifyRequestFailure({
        status: response.value?.status,
        error: failedError.value,
      })
    : null,
)

function createId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return 'request-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 8)
}

function requestBodyDefault(): string {
  const example = selectedEndpoint.value?.requestExample
  if (example === null || example === undefined) return ''
  return typeof example === 'string' ? example : JSON.stringify(example, null, 2)
}

function applyEndpoint(): void {
  const endpoint = selectedEndpoint.value
  if (!endpoint) return
  method.value = endpoint.method
  path.value = endpoint.path
  bodyText.value = requestBodyDefault()
  response.value = null
  failedError.value = null
}

function buildRequest(): WorkspaceRequest {
  return {
    id: currentRequest?.id || createId(),
    name: selectedEndpoint.value
      ? brandText(selectedEndpoint.value.summary[locale.value], siteName.value)
      : method.value + ' ' + path.value,
    method: method.value,
    url: baseUrl.value.replace(/\/$/, '') + '/' + path.value.replace(/^\//, ''),
    headers: parseJsonObject(headersText.value) as Record<string, string>,
    query: parseJsonObject(queryText.value) as Record<string, string>,
    body: bodyText.value,
    environment: currentRequest?.environment || 'development',
    createdAt: currentRequest?.createdAt || Date.now(),
    updatedAt: Date.now(),
  }
}

async function sendRequest(): Promise<void> {
  formError.value = ''
  savedNotice.value = ''
  failedError.value = null
  streamingText.value = ''

  let request: WorkspaceRequest
  try {
    request = buildRequest()
    parseJsonObject(headersText.value)
    parseJsonObject(queryText.value)
    if (bodyText.value.trim()) JSON.parse(bodyText.value)
  } catch {
    formError.value = t('developer.playground.invalidJson')
    return
  }

  if (mode.value === 'mock') {
    const mockBody = JSON.stringify(
      mockScenario.value === 'rate-limit'
        ? {
            error: {
              type: 'rate_limit_error',
              code: 'rate_limit_exceeded',
              message: 'Mock rate limit response',
            },
          }
        : selectedEndpoint.value?.responseExample ?? {},
      null,
      2,
    )
    response.value = {
      requestId: 'mock-' + createId(),
      status: mockScenario.value === 'rate-limit' ? 429 : 200,
      statusText: mockScenario.value === 'rate-limit' ? 'Mock Rate Limited' : 'Mock OK',
      headers: { 'content-type': 'application/json' },
      body: mockBody,
      durationMs: 0,
      firstByteMs: 0,
      bytes: new TextEncoder().encode(mockBody).byteLength,
      streaming: false,
    }
    if (mockScenario.value === 'rate-limit') {
      failedError.value = new Error('HTTP 429')
    }
    return
  }

  if (!liveConfirmed.value) {
    formError.value = t('developer.playground.liveConfirmationRequired')
    return
  }
  if (!apiKey.value.trim()) {
    formError.value = t('developer.playground.apiKeyRequired')
    return
  }
  const validation = validateLiveTarget(request.url)
  if (!validation.ok) {
    formError.value = t('developer.playground.httpsRequired')
    return
  }

  sending.value = true
  abortController = new AbortController()
  const startedAt = performance.now()
  try {
    const result = await executeLiveRequest({
      url: request.url,
      method: request.method,
      headers: request.headers,
      query: request.query || {},
      body: request.body,
      apiKey: apiKey.value,
      signal: abortController.signal,
      onChunk: (chunk) => {
        streamingText.value += chunk
      },
    })
    response.value = result
    await addRequestHistory({ ...request, body: '' })
    await addDebugRecord({
      requestId: result.requestId,
      method: request.method,
      url: request.url,
      status: result.status,
      durationMs: result.durationMs,
      requestHeaders: request.headers,
      requestBody: request.body,
      responseHeaders: result.headers,
      responseBody: result.body,
      createdAt: Date.now(),
    })
    if (result.status >= 400) {
      failedError.value = new Error('HTTP ' + result.status)
    }
  } catch (error) {
    failedError.value = error
    await addRequestHistory({ ...request, body: '' }).catch(() => undefined)
    await addDebugRecord({
      requestId: 'local-' + createId(),
      method: request.method,
      url: request.url,
      durationMs: Math.round(performance.now() - startedAt),
      requestHeaders: request.headers,
      requestBody: request.body,
      responseHeaders: {},
      responseBody: error instanceof Error ? error.message : String(error),
      createdAt: Date.now(),
    }).catch(() => undefined)
  } finally {
    sending.value = false
    abortController = null
  }
}

function stopRequest(): void {
  abortController?.abort(new DOMException('Stopped by user', 'AbortError'))
}

function clearKey(): void {
  apiKey.value = ''
  liveConfirmed.value = false
}

function resetRequest(): void {
  abortController?.abort()
  abortController = null
  apiKey.value = ''
  liveConfirmed.value = false
  response.value = null
  failedError.value = null
  streamingText.value = ''
  formError.value = ''
  savedNotice.value = ''
  applyEndpoint()
}

async function saveToWorkspace(): Promise<void> {
  formError.value = ''
  try {
    const request = buildRequest()
    currentRequest = await saveWorkspaceRequest(request)
    savedNotice.value = t('developer.playground.savedNotice')
    window.clearTimeout(savedTimer)
    savedTimer = window.setTimeout(() => {
      savedNotice.value = ''
    }, 2400)
  } catch {
    formError.value = t('developer.workspace.importError')
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

async function loadRouteRequest(): Promise<void> {
  const endpoint = String(route.query.endpoint || '')
  if (endpoint && getDeveloperEndpoint(endpoint)) {
    endpointId.value = endpoint
    applyEndpoint()
  }

  const requestId = String(route.query.requestId || '')
  if (!requestId) return
  try {
    const stored = await getWorkspaceRequest(requestId)
    if (!stored) return
    currentRequest = stored
    method.value = stored.method
    path.value = new URL(stored.url).pathname
    baseUrl.value = new URL(stored.url).origin
    headersText.value = JSON.stringify(stored.headers, null, 2)
    queryText.value = JSON.stringify(stored.query || {}, null, 2)
    bodyText.value = stored.body
  } catch {
    formError.value = t('developer.workspace.importError')
  }
}

watch(
  () => route.query,
  loadRouteRequest,
  { immediate: true },
)

watch(mode, () => {
  response.value = null
  failedError.value = null
  streamingText.value = ''
  formError.value = ''
})

onMounted(() => {
  localStorage.removeItem('sub2api_developer_base_url')
})

onBeforeUnmount(() => {
  apiKey.value = ''
  abortController?.abort()
  window.clearTimeout(savedTimer)
})
</script>
