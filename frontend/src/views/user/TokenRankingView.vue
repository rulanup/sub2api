<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header with time period tabs -->
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div class="flex items-center gap-2">
          <button
            v-for="opt in periodOptions"
            :key="opt.value"
            @click="period = opt.value; onPeriodChange()"
            class="rounded-lg px-3 py-1.5 text-sm font-medium transition-colors"
            :class="period === opt.value
              ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
              : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <!-- Summary Cards -->
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('tokenRanking.summary.totalCost') }}</p>
          <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">${{ fmtCost(summary.total_cost) }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('tokenRanking.summary.totalTokens') }}</p>
          <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">{{ fmtTokens(summary.total_tokens) }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('tokenRanking.summary.activeUsers') }}</p>
          <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">{{ summary.active_users }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('tokenRanking.summary.avgCost') }}</p>
          <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">${{ fmtCost(summary.avg_cost_per_user) }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('tokenRanking.summary.myRank') }}</p>
          <p class="mt-1 text-xl font-bold" :class="summary.my_rank ? 'text-primary-600 dark:text-primary-400' : 'text-gray-400'">
            {{ summary.my_rank ? `#${summary.my_rank}` : '-' }}
          </p>
        </div>
      </div>

      <!-- Sort buttons -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-for="opt in sortOptions"
          :key="opt.value"
          @click="sortBy = opt.value; load()"
          class="rounded-full px-3 py-1 text-xs font-medium transition-colors"
          :class="sortBy === opt.value
            ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'"
        >
          {{ opt.label }}
        </button>
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
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('tokenRanking.columns.cost') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('tokenRanking.columns.percentage') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('tokenRanking.columns.cacheHitRate') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('tokenRanking.columns.requests') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('tokenRanking.columns.tokens') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="loading">
                <td colspan="7" class="py-12 text-center">
                  <LoadingSpinner />
                </td>
              </tr>
              <tr v-else-if="items.length === 0">
                <td colspan="7" class="py-12 text-center text-sm text-gray-400">
                  {{ t('tokenRanking.noData') }}
                </td>
              </tr>
              <template v-else>
                <!-- Top 3 with medal badges -->
                <tr
                  v-for="(item, index) in top3"
                  :key="item.user_id"
                  class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40"
                >
                  <td class="px-4 py-3 sm:px-6">
                    <span
                      class="inline-flex h-7 w-7 items-center justify-center rounded-full text-sm font-bold"
                      :class="MEDAL_CLASSES[index]"
                    >{{ MEDAL_SYMBOLS[index] }}</span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex flex-col">
                      <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.username }}</span>
                    </div>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm font-bold text-gray-900 dark:text-white">${{ fmtCost(item.actual_cost) }}</td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm text-gray-500 dark:text-gray-400">
                    <div class="flex items-center justify-end gap-2">
                      <div class="h-1.5 w-16 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                        <div class="h-full rounded-full bg-primary-500" :style="{ width: item.percentage + '%' }"></div>
                      </div>
                      <span class="w-10 text-right">{{ item.percentage.toFixed(1) }}%</span>
                    </div>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm text-gray-500 dark:text-gray-400">
                    <span :class="getCacheHitClass(item.cache_hit_rate)">{{ item.cache_hit_rate.toFixed(0) }}%</span>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ item.requests.toLocaleString() }}</td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ fmtTokens(item.total_tokens) }}</td>
                </tr>

                <!-- Rest of the list -->
                <tr
                  v-for="item in rest"
                  :key="item.user_id"
                  class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40"
                >
                  <td class="px-4 py-3 sm:px-6">
                    <span class="inline-block w-6 text-center text-sm tabular-nums text-gray-400">{{ item.rank }}</span>
                  </td>
                  <td class="px-4 py-3">
                    <span class="text-sm text-gray-700 dark:text-gray-200">{{ item.username }}</span>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm font-medium tabular-nums text-gray-900 dark:text-white">${{ fmtCost(item.actual_cost) }}</td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm text-gray-500 dark:text-gray-400">
                    <div class="flex items-center justify-end gap-2">
                      <div class="h-1.5 w-16 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                        <div class="h-full rounded-full bg-primary-500" :style="{ width: item.percentage + '%' }"></div>
                      </div>
                      <span class="w-10 text-right">{{ item.percentage.toFixed(1) }}%</span>
                    </div>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm text-gray-500 dark:text-gray-400">
                    <span :class="getCacheHitClass(item.cache_hit_rate)">{{ item.cache_hit_rate.toFixed(0) }}%</span>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ item.requests.toLocaleString() }}</td>
                  <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ fmtTokens(item.total_tokens) }}</td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { formatCompactNumber, formatCostFixed } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
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
  cache_hit_rate: number
  percentage: number
  rank: number
}

interface TokenRankingSummary {
  total_cost: number
  total_tokens: number
  active_users: number
  avg_cost_per_user: number
  my_rank: number | null
  my_cost: number
}

const MEDAL_SYMBOLS = ['🥈', '🥇', '🥉']
const MEDAL_CLASSES = [
  'bg-gray-100 text-gray-700 dark:bg-gray-500/20 dark:text-gray-300',
  'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400',
  'bg-orange-100 text-orange-700 dark:bg-orange-500/20 dark:text-orange-400',
]

const periodOptions = [
  { value: 'day', label: t('tokenRanking.periods.day') },
  { value: 'week', label: t('tokenRanking.periods.week') },
  { value: 'month', label: t('tokenRanking.periods.month') },
]

const sortOptions = computed(() => [
  { value: 'actual_cost', label: t('tokenRanking.sortByCost') },
  { value: 'total_tokens', label: t('tokenRanking.sortByTokens') },
  { value: 'requests', label: t('tokenRanking.sortByRequests') },
])

const items = ref<TokenRankingItem[]>([])
const summary = ref<TokenRankingSummary>({
  total_cost: 0,
  total_tokens: 0,
  active_users: 0,
  avg_cost_per_user: 0,
  my_rank: null,
  my_cost: 0,
})
const loading = ref(false)
const period = ref('week')
const sortBy = ref('actual_cost')
const startDate = ref('')
const endDate = ref('')

const top3 = computed(() => items.value.slice(0, 3))
const rest = computed(() => items.value.slice(3))

const fmtTokens = (v: number) => formatCompactNumber(v)
const fmtCost = (v: number) => formatCostFixed(v, 2)

const getCacheHitClass = (rate: number) => {
  if (rate >= 90) return 'text-green-600 dark:text-green-400 font-medium'
  if (rate >= 70) return 'text-gray-600 dark:text-gray-400'
  return 'text-orange-500 dark:text-orange-400'
}

const getDateRange = () => {
  const now = new Date()
  let start: Date

  switch (period.value) {
    case 'day':
      start = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      break
    case 'week':
      start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 6)
      break
    case 'month':
      start = new Date(now.getFullYear(), now.getMonth(), 1)
      break
    default:
      start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 6)
  }

  startDate.value = start.toISOString().split('T')[0]
  endDate.value = now.toISOString().split('T')[0]
}

const onPeriodChange = () => {
  getDateRange()
  load()
}

const load = async () => {
  loading.value = true
  try {
    const params = new URLSearchParams()
    params.set('start_date', startDate.value)
    params.set('end_date', endDate.value)
    params.set('sort_by', sortBy.value)
    params.set('limit', '50')

    const { data } = await apiClient.get(`/usage/token-ranking?${params.toString()}`)
    items.value = data.items || []
    summary.value = data.summary || summary.value
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
