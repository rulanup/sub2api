<template>
  <div class="flex h-full gap-4">
    <!-- Conversation List -->
    <div class="hidden w-60 flex-shrink-0 flex-col rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700 sm:flex">
      <div class="flex items-center justify-between border-b border-gray-200 px-3 py-2.5 dark:border-dark-700">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('workbench.conversations') }}</h3>
        <button @click="createConversation" class="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-800 dark:hover:text-gray-300">
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
        </button>
      </div>
      <div class="flex-1 overflow-y-auto py-1">
        <div v-if="conversations.length === 0" class="px-3 py-6 text-center text-xs text-gray-400 dark:text-gray-500">
          {{ t('workbench.noConversations') }}
        </div>
        <button
          v-for="conv in conversations"
          :key="conv.id"
          @click="switchConversation(conv.id)"
          :class="[
            'group flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors',
            conv.id === activeConvId
              ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300'
              : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
          ]"
        >
          <svg class="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" />
          </svg>
          <span class="min-w-0 flex-1 truncate">{{ conv.title }}</span>
          <button
            @click.stop="deleteConversation(conv.id)"
            class="flex-shrink-0 rounded p-0.5 text-gray-300 opacity-0 transition-opacity hover:text-red-500 group-hover:opacity-100 dark:text-gray-600"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </button>
      </div>
    </div>

    <!-- Chat Area -->
    <div class="flex min-w-0 flex-1 flex-col rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700">
      <!-- Messages -->
      <div ref="messagesContainer" class="flex-1 space-y-4 overflow-y-auto p-4">
        <div v-if="activeMessages.length === 0" class="flex h-full items-center justify-center">
          <p class="text-sm text-gray-400 dark:text-gray-500">{{ t('workbench.chatWelcome') }}</p>
        </div>
        <div v-for="(msg, i) in activeMessages" :key="i" class="flex gap-3" :class="msg.role === 'user' ? 'justify-end' : 'justify-start'">
          <div
            :class="[
              'max-w-[80%] rounded-2xl px-4 py-2.5 text-sm',
              msg.role === 'user'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-900 dark:bg-dark-800 dark:text-gray-100'
            ]"
          >
            <div class="whitespace-pre-wrap break-words">{{ msg.content }}</div>
          </div>
        </div>
        <div v-if="streaming" class="flex gap-3">
          <div class="max-w-[80%] rounded-2xl bg-gray-100 px-4 py-2.5 text-sm text-gray-900 dark:bg-dark-800 dark:text-gray-100">
            <div class="whitespace-pre-wrap break-words">{{ streamingContent }}<span class="animate-pulse">|</span></div>
          </div>
        </div>
      </div>

      <!-- Input -->
      <div class="border-t border-gray-200 p-4 dark:border-dark-700">
        <div class="flex gap-2">
          <textarea
            v-model="input"
            @keydown.enter.exact.prevent="sendMessage"
            :placeholder="t('workbench.chatPlaceholder')"
            rows="1"
            class="flex-1 resize-none rounded-xl border-0 bg-gray-100 px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white dark:placeholder-gray-500"
            style="field-sizing: content; max-height: 120px;"
          />
          <button
            @click="sendMessage"
            :disabled="!canSend"
            class="flex-shrink-0 rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ t('workbench.send') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Settings Panel -->
    <div class="hidden w-72 flex-shrink-0 flex-col rounded-xl bg-white p-4 shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700 lg:flex">
      <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('workbench.chatSettings') }}</h3>

      <!-- API Key -->
      <div class="mb-3">
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('workbench.chatKey') }}</label>
        <select v-model="selectedKeyId" class="w-full rounded-lg border-0 bg-gray-100 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white">
          <option :value="null" disabled>{{ t('workbench.selectKey') }}</option>
          <option v-for="k in apiKeys" :key="k.id" :value="k.id">
            {{ k.name || ('sk-' + k.key.slice(-4)) }} ({{ k.key.slice(0, 10) }}...{{ k.key.slice(-4) }})
          </option>
        </select>
      </div>

      <!-- Model -->
      <div class="mb-3">
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('workbench.chatModel') }}</label>
        <div class="relative" ref="modelDropdownRef">
          <button
            @click="modelDropdownOpen = !modelDropdownOpen"
            type="button"
            class="flex w-full items-center justify-between rounded-lg border-0 bg-gray-100 px-3 py-2 text-left text-sm text-gray-900 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white"
          >
            <span class="truncate">{{ model || t('workbench.selectModel') }}</span>
            <svg class="h-4 w-4 flex-shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" /></svg>
          </button>
          <div v-if="modelDropdownOpen" class="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-lg bg-white py-1 shadow-lg ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-600">
            <!-- Search -->
            <div class="sticky top-0 bg-white px-2 pb-1 pt-1 dark:bg-dark-800">
              <input
                v-model="modelSearch"
                ref="modelSearchInput"
                type="text"
                class="w-full rounded-md border-0 bg-gray-100 px-2.5 py-1.5 text-xs text-gray-900 placeholder-gray-400 focus:ring-1 focus:ring-blue-500 dark:bg-dark-700 dark:text-white dark:placeholder-gray-500"
                :placeholder="t('workbench.modelSearchPlaceholder')"
              />
            </div>
            <!-- Grouped models -->
            <template v-if="groupedModels.length > 0">
              <div v-for="group in groupedModels" :key="group.platform">
                <div class="px-3 py-1 text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                  {{ group.platform }}
                </div>
                <button
                  v-for="m in group.models"
                  :key="m"
                  @mousedown.prevent="selectModel(m)"
                  class="flex w-full items-center px-3 py-1.5 text-left text-sm transition-colors"
                  :class="m === model
                    ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300'
                    : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700'"
                >
                  <span class="truncate">{{ m }}</span>
                  <svg v-if="m === model" class="ml-auto h-3.5 w-3.5 flex-shrink-0 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>
                </button>
              </div>
            </template>
            <div v-else class="px-3 py-4 text-center text-xs text-gray-400 dark:text-gray-500">
              {{ t('workbench.noModels') }}
            </div>
          </div>
        </div>
      </div>

      <!-- Reasoning Level -->
      <div class="mb-3">
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('workbench.reasoningLevel') }}</label>
        <select v-model="reasoningEffort" class="w-full rounded-lg border-0 bg-gray-100 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white">
          <option value="auto">Auto</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
        </select>
      </div>

      <!-- Actions -->
      <div class="mt-auto space-y-2">
        <button @click="clearActiveChat" class="w-full rounded-lg bg-gray-100 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700">
          {{ t('workbench.clearChat') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ApiKey } from '@/types'

const { t } = useI18n()

const props = defineProps<{
  apiKeys: ApiKey[]
  loadingKeys: boolean
}>()

interface ChatMessage { role: 'user' | 'assistant'; content: string }
interface Conversation {
  id: string
  title: string
  messages: ChatMessage[]
  model: string
  keyId: number | null
  createdAt: number
}

interface ModelGroup { platform: string; models: string[] }

const STORAGE_KEY = 'sub2api_workbench_conversations'

// State
const input = ref('')
const streaming = ref(false)
const streamingContent = ref('')
const messagesContainer = ref<HTMLElement | null>(null)

const selectedKeyId = ref<number | null>(null)
const model = ref('gpt-4o-mini')
const reasoningEffort = ref('auto')
const modelSearch = ref('')
const modelDropdownOpen = ref(false)
const modelSearchInput = ref<HTMLInputElement | null>(null)
const modelDropdownRef = ref<HTMLElement | null>(null)

const conversations = ref<Conversation[]>([])
const activeConvId = ref<string>('')

// Available models from gateway /v1/models
const modelGroups = ref<ModelGroup[]>([])
const modelsLoading = ref(false)

onMounted(async () => {
  loadConversations()
  document.addEventListener('click', handleOutsideClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleOutsideClick)
})

// Fetch models from gateway when API key changes
async function fetchModels(key: ApiKey) {
  modelsLoading.value = true
  try {
    const resp = await fetch('/v1/models', {
      headers: { 'Authorization': `Bearer ${key.key}` },
    })
    if (!resp.ok) { modelGroups.value = []; return }
    const data = await resp.json()
    const models: string[] = (data.data || []).map((m: { id: string }) => m.id).sort()
    // Group by prefix as platform
    const platformMap = new Map<string, string[]>()
    for (const m of models) {
      const parts = m.split('-')
      const platform = parts[0] || 'other'
      if (!platformMap.has(platform)) platformMap.set(platform, [])
      platformMap.get(platform)!.push(m)
    }
    modelGroups.value = [...platformMap.entries()].map(([platform, ms]) => ({ platform, models: ms }))
  } catch (e) {
    console.warn('Failed to fetch models:', e)
    modelGroups.value = []
  } finally {
    modelsLoading.value = false
  }
}

onBeforeUnmount(() => {
  document.removeEventListener('click', handleOutsideClick)
})

function handleOutsideClick(e: MouseEvent) {
  if (modelDropdownRef.value && !modelDropdownRef.value.contains(e.target as Node)) {
    modelDropdownOpen.value = false
  }
}

watch(modelDropdownOpen, (open) => {
  if (open) {
    modelSearch.value = ''
    nextTick(() => modelSearchInput.value?.focus())
  }
})

const groupedModels = computed(() => {
  const q = modelSearch.value.toLowerCase().trim()
  if (!q) return modelGroups.value.map(g => ({ platform: g.platform, models: g.models.slice(0, 30) }))
  return modelGroups.value
    .map(g => ({ platform: g.platform, models: g.models.filter(m => m.toLowerCase().includes(q)) }))
    .filter(g => g.models.length > 0)
})

function selectModel(m: string) {
  model.value = m
  modelSearch.value = ''
  modelDropdownOpen.value = false
  const conv = activeConv.value
  if (conv) { conv.model = m; saveConversations() }
}

// Auto-select first key and fetch models
watch(() => props.apiKeys, (keys) => {
  if (keys.length > 0 && selectedKeyId.value === null) {
    selectedKeyId.value = keys[0].id
    fetchModels(keys[0])
  }
}, { immediate: true })

// Fetch models when key changes
watch(selectedKeyId, (keyId) => {
  const key = props.apiKeys.find(k => k.id === keyId)
  if (key) fetchModels(key)
})

const selectedKey = computed(() => props.apiKeys.find(k => k.id === selectedKeyId.value))
const activeConv = computed(() => conversations.value.find(c => c.id === activeConvId.value))
const activeMessages = computed(() => activeConv.value?.messages || [])
const canSend = computed(() => !streaming.value && input.value.trim() && selectedKey.value && activeConv.value)

// Conversation management
function genId() { return Date.now().toString(36) + Math.random().toString(36).slice(2, 6) }

function createConversation() {
  const conv: Conversation = { id: genId(), title: t('workbench.newConversation'), messages: [], model: model.value, keyId: selectedKeyId.value, createdAt: Date.now() }
  conversations.value.unshift(conv)
  activeConvId.value = conv.id
  saveConversations()
}

function switchConversation(id: string) {
  activeConvId.value = id
  const conv = activeConv.value
  if (conv) { model.value = conv.model; if (conv.keyId) selectedKeyId.value = conv.keyId }
}

function deleteConversation(id: string) {
  const idx = conversations.value.findIndex(c => c.id === id)
  if (idx < 0) return
  conversations.value.splice(idx, 1)
  if (activeConvId.value === id) activeConvId.value = conversations.value[0]?.id || ''
  saveConversations()
}

function clearActiveChat() {
  const conv = activeConv.value
  if (conv) { conv.messages = []; conv.title = t('workbench.newConversation'); saveConversations() }
}

// Persistence
function saveConversations() { try { localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations.value)) } catch {} }
function loadConversations() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      conversations.value = JSON.parse(raw)
      if (conversations.value.length > 0) {
        activeConvId.value = conversations.value[0].id
        model.value = conversations.value[0].model
        if (conversations.value[0].keyId) selectedKeyId.value = conversations.value[0].keyId
      }
    }
  } catch {}
  if (conversations.value.length === 0) createConversation()
}

function scrollToBottom() { nextTick(() => { if (messagesContainer.value) messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight }) }

async function sendMessage() {
  if (!canSend.value) return
  const key = selectedKey.value!
  const conv = activeConv.value!
  const userMsg = input.value.trim()
  input.value = ''

  conv.messages.push({ role: 'user', content: userMsg })
  if (conv.messages.length === 1) conv.title = userMsg.slice(0, 40) + (userMsg.length > 40 ? '...' : '')
  conv.model = model.value; conv.keyId = selectedKeyId.value
  saveConversations(); scrollToBottom()

  streaming.value = true; streamingContent.value = ''

  const body: Record<string, unknown> = { model: model.value, messages: conv.messages.map(m => ({ role: m.role, content: m.content })), stream: true }
  if (reasoningEffort.value !== 'auto') body.reasoning_effort = reasoningEffort.value

  try {
    const resp = await fetch('/v1/chat/completions', { method: 'POST', headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${key.key}` }, body: JSON.stringify(body) })
    if (!resp.ok) {
      let errMsg: string
      try { const j = await resp.json(); errMsg = j.error?.message || JSON.stringify(j) } catch { errMsg = await resp.text() }
      conv.messages.push({ role: 'assistant', content: `Error ${resp.status}: ${errMsg}` })
      streaming.value = false; saveConversations(); return
    }

    const reader = resp.body?.getReader(), decoder = new TextDecoder()
    let buffer = ''
    while (reader) {
      const { done, value } = await reader.read(); if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n'); buffer = lines.pop() || ''
      for (const line of lines) {
        const trimmed = line.trim(); if (!trimmed || !trimmed.startsWith('data: ')) continue
        const data = trimmed.slice(6); if (data === '[DONE]') continue
        try { const parsed = JSON.parse(data); const delta = parsed.choices?.[0]?.delta?.content; if (delta) streamingContent.value += delta; scrollToBottom() } catch {}
      }
    }
    if (streamingContent.value) conv.messages.push({ role: 'assistant', content: streamingContent.value })
  } catch (e) {
    conv.messages.push({ role: 'assistant', content: `Error: ${e}` })
  } finally {
    streaming.value = false; streamingContent.value = ''; saveConversations(); scrollToBottom()
  }
}
</script>
