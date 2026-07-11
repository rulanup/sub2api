<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('tokenRanking.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('tokenRanking.description') }}</p>
        </div>
        <button @click="load" class="btn btn-secondary">
          <svg class="mr-1.5 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" />
          </svg>
          {{ t('tokenRanking.refresh') }}
        </button>
      </div>

      <!-- Filters -->
      <div class="card p-4">
        <div class="flex flex-wrap items-end gap-4">
          <div>
            <label class="input-label">{{ t('tokenRanking.period') }}</label>
            <Select v-model="period" :options="periodOptions" class="w-32" @change="onPeriodChange" />
          </div>
          <div>
            <label class="input-label">{{ t('tokenRanking.limit') }}</label>
            <Select v-model="limit" :options="limitOptions" class="w-28" @change="load" />
          </div>
          <div>
            <label class="input-label">{{ t('tokenRanking.sortBy') }}</label>
            <Select v-model="sortBy" :options="sortOptions" class="w-40" @change="load" />
          </div>
        </div>
      </div>

      <!-- Date range display -->
      <div class="flex items-center gap-2 text-xs text-gray-400 dark:text-gray-500">
        <span>{{ startDate }} ~ {{ endDate }}</span>
      </div>

      <!-- Table -->
      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full min-w-max divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="w-16 px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400 sm:px-6">#</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('tokenRanking.columns.user') }}
                </th>
                <th
                  v-for="col in sortableColumns"
                  :key="col.key"
                  class="cursor-pointer select-none whitespace-nowrap px-4 py-3 text-right text-xs font-medium uppercase tracking-wider transition-colors hover:bg-gray-100 dark:hover:bg-dark-700"
                  :class="sortBy === col.key ? 'text-primary-600 dark:text-primary-400' : 'text-gray-500 dark:text-dark-400'"
                  @click="setSort(col.key)"
                >
                  {{ t(col.label) }}
                  <span v-if="sortBy === col.key" aria-hidden="true">↓</span>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="loading">
                <td :colspan="sortableColumns.length + 2" class="py-12 text-center">
                  <LoadingSpinner />
                </td>
              </tr>
              <tr v-else-if="items.length === 0">
                <td :colspan="sortableColumns.length + 2" class="py-12 text-center text-sm text-gray-400">
                  {{ t('tokenRanking.noData') }}
                </td>
              </tr>
              <tr
                v-for="(item, index) in items"
                v-else
                :key="item.user_id"
                class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40"
              >
                <td class="px-4 py-3 sm:px-6">
                  <span
                    v-if="index < 3"
                    class="inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold"
                    :class="RANK_BADGE_CLASSES[index]"
                  >{{ index + 1 }}</span>
                  <span v-else class="inline-block w-6 text-center text-sm tabular-nums text-gray-400">{{ index + 1 }}</span>
                </td>
                <td class="max-w-[260px] truncate px-4 py-3 text-sm font-medium text-gray-700 dark:text-gray-200">
                  {{ item.username || `User #${item.user_id}` }}
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ item.requests.toLocaleString() }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ fmtTokens(item.input_tokens) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ fmtTokens(item.output_tokens) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ fmtTokens(item.cache_tokens) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm font-medium tabular-nums text-gray-900 dark:text-gray-100">{{ fmtTokens(item.total_tokens) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm font-medium tabular-nums text-green-600 dark:text-green-400">${{ fmtCost(item.actual_cost) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { formatCompactNumber, formatCostFixed } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const { t } = useI18n()

interface TokenRankingItem {
  user_id: number
  username: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_tokens: number
  total_tokens: number
  actual_cost: number
}

type SortKey = 'total_tokens' | 'input_tokens' | 'output_tokens' | 'cache_tokens' | 'requests' | 'actual_cost'

const sortableColumns: { key: SortKey; label: string }[] = [
  { key: 'requests', label: 'tokenRanking.columns.requests' },
  { key: 'input_tokens', label: 'tokenRanking.columns.inputTokens' },
  { key: 'output_tokens', label: 'tokenRanking.columns.outputTokens' },
  { key: 'cache_tokens', label: 'tokenRanking.columns.cacheTokens' },
  { key: 'total_tokens', label: 'tokenRanking.columns.totalTokens' },
  { key: 'actual_cost', label: 'tokenRanking.columns.cost' },
]

const RANK_BADGE_CLASSES = [
  'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400',
  'bg-gray-200 text-gray-600 dark:bg-gray-500/20 dark:text-gray-300',
  'bg-orange-100 text-orange-700 dark:bg-orange-500/20 dark:text-orange-400',
]

const periodOptions = [
  { value: 'day', label: t('tokenRanking.periods.day') },
  { value: 'week', label: t('tokenRanking.periods.week') },
  { value: 'month', label: t('tokenRanking.periods.month') },
  { value: 'all', label: t('tokenRanking.periods.all') },
]

const limitOptions = [
  { value: 20, label: 'Top 20' },
  { value: 50, label: 'Top 50' },
  { value: 100, label: 'Top 100' },
  { value: 200, label: 'Top 200' },
]

const sortOptions = [
  { value: 'total_tokens', label: t('tokenRanking.columns.totalTokens') },
  { value: 'input_tokens', label: t('tokenRanking.columns.inputTokens') },
  { value: 'output_tokens', label: t('tokenRanking.columns.outputTokens') },
  { value: 'cache_tokens', label: t('tokenRanking.columns.cacheTokens') },
  { value: 'requests', label: t('tokenRanking.columns.requests') },
  { value: 'actual_cost', label: t('tokenRanking.columns.cost') },
]

const items = ref<TokenRankingItem[]>([])
const loading = ref(false)
const period = ref('week')
const sortBy = ref<SortKey>('total_tokens')
const limit = ref(50)
const startDate = ref('')
const endDate = ref('')

const fmtTokens = (v: number) => formatCompactNumber(v)
const fmtCost = (v: number) => formatCostFixed(v, 4)

const getDateRange = () => {
  const now = new Date()
  let start: Date
  const end = now

  switch (period.value) {
    case 'day':
      start = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      break
    case 'week': {
      const day = now.getDay()
      const diff = day === 0 ? 6 : day - 1
      start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - diff)
      break
    }
    case 'month':
      start = new Date(now.getFullYear(), now.getMonth(), 1)
      break
    default:
      start = new Date(2020, 0, 1)
  }

  startDate.value = start.toISOString().split('T')[0]
  endDate.value = end.toISOString().split('T')[0]
}

const onPeriodChange = () => {
  getDateRange()
  load()
}

const setSort = (key: SortKey) => {
  if (sortBy.value === key) return
  sortBy.value = key
  load()
}

const load = async () => {
  loading.value = true
  try {
    const params = new URLSearchParams()
    params.set('start_date', startDate.value)
    params.set('end_date', endDate.value)
    params.set('sort_by', sortBy.value)
    params.set('limit', String(limit.value))

    const { data } = await apiClient.get(`/usage/token-ranking?${params.toString()}`)
    items.value = data.users || []
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  getDateRange()
  load()
})
</script>
