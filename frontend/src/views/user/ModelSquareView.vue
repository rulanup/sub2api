<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('modelSquare.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('modelSquare.description') }}</p>
      </div>

      <!-- Stats -->
      <div class="grid grid-cols-3 gap-4">
        <div class="card p-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelSquare.availableModels') }}</p>
          <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ models.length }}</p>
        </div>
        <div class="card p-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelSquare.availableChannels') }}</p>
          <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ channelCount }}</p>
        </div>
        <div class="card p-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelSquare.platforms') }}</p>
          <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ platformSet.size }}</p>
        </div>
      </div>

      <!-- Search & Filter -->
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
        <!-- Search -->
        <div class="relative flex-1">
          <svg class="absolute left-3 top-2.5 h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
          <input
            v-model="search"
            type="text"
            :placeholder="t('modelSquare.searchPlaceholder')"
            class="w-full rounded-lg border-0 bg-white py-2 pl-9 pr-4 text-sm text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-blue-500 dark:bg-dark-900 dark:text-white dark:ring-dark-600"
          />
        </div>
        <!-- Sort -->
        <select v-model="sortBy" class="rounded-lg border-0 bg-white px-3 py-2 text-sm text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-blue-500 dark:bg-dark-900 dark:text-white dark:ring-dark-600">
          <option value="name">{{ t('modelSquare.sortByName') }}</option>
          <option value="channels">{{ t('modelSquare.sortByChannels') }}</option>
          <option value="platform">{{ t('modelSquare.sortByPlatform') }}</option>
        </select>
      </div>

      <!-- Platform Tabs -->
      <div class="flex flex-wrap gap-2">
        <button
          v-for="tab in platformTabs"
          :key="tab.key"
          @click="activePlatform = tab.key"
          :class="[
            'rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors',
            activePlatform === tab.key
              ? 'bg-blue-600 text-white'
              : 'bg-white text-gray-600 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-400 dark:ring-dark-600 dark:hover:bg-dark-700'
          ]"
        >
          {{ tab.label }} ({{ tab.count }})
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <!-- Model Grid -->
      <div v-else-if="filteredModels.length > 0" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="m in filteredModels"
          :key="m.name"
          class="flex flex-col rounded-xl bg-white p-4 shadow-sm ring-1 ring-gray-200 transition-shadow hover:shadow-md dark:bg-dark-900 dark:ring-dark-700"
        >
          <!-- Header -->
          <div class="flex items-center gap-2">
            <h3 class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900 dark:text-white">{{ m.name }}</h3>
            <span class="flex-shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700 dark:text-gray-400">
              {{ m.platform }}
            </span>
          </div>

          <!-- Channel count -->
          <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
            {{ m.channelCount }} {{ t('modelSquare.channelUnit') }}
          </p>

          <!-- Pricing -->
          <div v-if="m.pricing" class="mt-3 space-y-1 rounded-lg bg-gray-50 p-2.5 text-xs dark:bg-dark-800">
            <div v-if="m.pricing.input_price !== null" class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-gray-400">{{ t('modelSquare.input') }}</span>
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ formatPrice(m.pricing.input_price) }}</span>
            </div>
            <div v-if="m.pricing.output_price !== null" class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-gray-400">{{ t('modelSquare.output') }}</span>
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ formatPrice(m.pricing.output_price) }}</span>
            </div>
            <div v-if="m.pricing.cache_read_price !== null && m.pricing.cache_read_price > 0" class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-gray-400">{{ t('modelSquare.cacheRead') }}</span>
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ formatPrice(m.pricing.cache_read_price) }}</span>
            </div>
            <div v-if="m.pricing.per_request_price !== null" class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-gray-400">{{ t('modelSquare.perRequest') }}</span>
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ formatPrice(m.pricing.per_request_price) }}</span>
            </div>
            <div v-if="m.pricing.image_output_price !== null && m.pricing.image_output_price > 0" class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-gray-400">{{ t('modelSquare.perImage') }}</span>
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ formatPrice(m.pricing.image_output_price) }}</span>
            </div>
          </div>
          <div v-else class="mt-3 rounded-lg bg-gray-50 p-2.5 text-xs text-gray-400 dark:bg-dark-800 dark:text-gray-500">
            {{ t('modelSquare.noPricing') }}
          </div>

          <!-- Channels & Groups -->
          <div v-if="m.channels.length > 0" class="mt-auto pt-3">
            <div class="flex flex-wrap gap-1.5">
              <div
                v-for="ch in m.channels"
                :key="ch.channelName + ch.groupName"
                class="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-[11px] dark:bg-blue-900/20"
              >
                <span class="font-medium text-blue-700 dark:text-blue-300">{{ ch.groupName }}</span>
                <span class="rounded-full bg-blue-100 px-1 py-0.5 text-[10px] font-medium text-blue-600 dark:bg-blue-800/40 dark:text-blue-400">
                  {{ ch.rateMultiplier }}x
                </span>
              </div>
            </div>
          </div>

          <!-- Latency -->
          <div class="mt-2">
            <button
              @click.stop="testLatency(m.name)"
              :disabled="latencyMap[m.name]?.testing"
              class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium transition-colors hover:opacity-80"
              :class="latencyClass(m.name)"
            >
              <template v-if="latencyMap[m.name]?.testing">
                <svg class="h-3 w-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
                {{ t('modelSquare.testing') }}
              </template>
              <template v-else-if="latencyMap[m.name]?.status === 'error'">
                ✕ {{ t('modelSquare.latencyError') }}
              </template>
              <template v-else-if="latencyMap[m.name]?.latency != null">
                {{ latencyMap[m.name].latency }}ms
              </template>
              <template v-else>
                {{ t('modelSquare.testLatency') }}
              </template>
            </button>
          </div>
        </div>
      </div>

      <!-- Empty -->
      <div v-else class="py-12 text-center text-sm text-gray-400 dark:text-gray-500">
        {{ t('modelSquare.noModels') }}
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAvailable, type UserAvailableChannel, type UserSupportedModelPricing } from '@/api/channels'
import { testModelLatency } from '@/api/usage'
import { keysAPI } from '@/api/keys'
import type { ApiKey } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const { t } = useI18n()

interface ModelChannel {
  channelName: string
  groupName: string
  rateMultiplier: number
}

interface ModelEntry {
  name: string
  platform: string
  pricing: UserSupportedModelPricing | null
  channels: ModelChannel[]
  channelCount: number
}

const loading = ref(true)
const search = ref('')
const sortBy = ref<'name' | 'channels' | 'platform'>('channels')
const activePlatform = ref('all')
const channels = ref<UserAvailableChannel[]>([])
const apiKeys = ref<ApiKey[]>([])

// Latency state: modelName -> { latency, status, testing }
const latencyMap = ref<Record<string, { latency: number | null; status: string; testing: boolean }>>({})

onMounted(async () => {
  try {
    const [chRes, keysRes] = await Promise.all([
      getAvailable(),
      keysAPI.list(1, 100, { status: 'active' }),
    ])
    channels.value = chRes
    apiKeys.value = keysRes.items || []
  } catch (e) {
    console.error('Failed to load data:', e)
  } finally {
    loading.value = false
  }
})

async function testLatency(modelName: string, keyId?: number) {
  if (!keyId && apiKeys.value.length === 0) return
  const kid = keyId ?? apiKeys.value[0].id
  latencyMap.value[modelName] = { latency: null, status: 'testing', testing: true }
  try {
    const result = await testModelLatency(modelName, kid)
    latencyMap.value[modelName] = {
      latency: result.latency,
      status: result.status,
      testing: false,
    }
  } catch {
    latencyMap.value[modelName] = { latency: null, status: 'error', testing: false }
  }
}

function latencyClass(name: string): string {
  const l = latencyMap.value[name]
  if (!l || l.testing) return 'bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-gray-500 cursor-wait'
  if (l.status === 'error') return 'bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400'
  if (l.latency == null) return 'bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-gray-500 hover:bg-gray-200 dark:hover:bg-dark-600'
  if (l.latency < 1000) return 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400'
  if (l.latency < 3000) return 'bg-yellow-50 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400'
  return 'bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400'
}

// Build model list from channels
const models = computed<ModelEntry[]>(() => {
  const map = new Map<string, ModelEntry>()
  for (const ch of channels.value) {
    for (const sec of ch.platforms) {
      for (const m of sec.supported_models) {
        const key = m.name.toLowerCase()
        if (!map.has(key)) {
          map.set(key, {
            name: m.name,
            platform: sec.platform,
            pricing: m.pricing,
            channels: [],
            channelCount: 0,
          })
        }
        const entry = map.get(key)!
        for (const g of sec.groups) {
          entry.channels.push({
            channelName: ch.name,
            groupName: g.name,
            rateMultiplier: g.rate_multiplier,
          })
        }
        // Use first non-null pricing
        if (!entry.pricing && m.pricing) entry.pricing = m.pricing
      }
    }
  }
  // Deduplicate channels and count
  for (const entry of map.values()) {
    const seen = new Set<string>()
    entry.channels = entry.channels.filter(ch => {
      const key = `${ch.channelName}|${ch.groupName}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
    entry.channelCount = entry.channels.length
  }
  return [...map.values()]
})

const channelCount = computed(() => channels.value.length)

const platformSet = computed(() => new Set(models.value.map(m => m.platform)))

const platformTabs = computed(() => {
  const tabs = [{ key: 'all', label: t('modelSquare.all'), count: models.value.length }]
  const counts = new Map<string, number>()
  for (const m of models.value) counts.set(m.platform, (counts.get(m.platform) || 0) + 1)
  for (const [platform, count] of [...counts.entries()].sort((a, b) => b[1] - a[1])) {
    tabs.push({ key: platform, label: platform, count })
  }
  return tabs
})

const filteredModels = computed(() => {
  let list = models.value
  // Platform filter
  if (activePlatform.value !== 'all') {
    list = list.filter(m => m.platform === activePlatform.value)
  }
  // Search
  const q = search.value.toLowerCase().trim()
  if (q) {
    list = list.filter(m =>
      m.name.toLowerCase().includes(q) ||
      m.platform.toLowerCase().includes(q) ||
      m.channels.some(ch => ch.channelName.toLowerCase().includes(q) || ch.groupName.toLowerCase().includes(q))
    )
  }
  // Sort
  if (sortBy.value === 'channels') list = [...list].sort((a, b) => b.channelCount - a.channelCount)
  else if (sortBy.value === 'platform') list = [...list].sort((a, b) => a.platform.localeCompare(b.platform) || a.name.localeCompare(b.name))
  else list = [...list].sort((a, b) => a.name.localeCompare(b.name))
  return list
})

function formatPrice(v: number | null): string {
  if (v === null || v === undefined) return '-'
  if (v === 0) return 'Free'
  if (v >= 1) return `$${v.toFixed(2)}`
  if (v >= 0.01) return `$${v.toFixed(4)}`
  if (v >= 0.001) return `$${v.toFixed(6)}`
  // Per-token price → display as per 1M tokens
  const per1m = v * 1_000_000
  if (per1m >= 1) return `$${per1m.toFixed(2)} / 1M token`
  return `$${per1m.toFixed(4)} / 1M token`
}
</script>
