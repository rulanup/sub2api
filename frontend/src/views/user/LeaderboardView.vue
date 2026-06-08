<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('leaderboard.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('leaderboard.description') }}</p>
        </div>
        <button
          class="inline-flex items-center gap-2 rounded-lg bg-white px-3 py-2 text-sm font-medium text-gray-700 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-600 dark:hover:bg-dark-700"
          @click="loadData"
          :disabled="loading"
        >
          <svg class="h-4 w-4" :class="{ 'animate-spin': loading }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ t('leaderboard.refresh') }}
        </button>
      </div>

      <!-- Period Tabs -->
      <div class="flex items-center gap-2">
        <button
          v-for="p in periods"
          :key="p.value"
          @click="period = p.value; loadData()"
          :class="[
            'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
            period === p.value
              ? 'bg-blue-600 text-white shadow-sm'
              : 'bg-white text-gray-700 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-600 dark:hover:bg-dark-700'
          ]"
        >
          {{ p.label }}
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading && !data" class="flex items-center justify-center py-20">
        <LoadingSpinner />
      </div>

      <template v-else-if="data">
        <!-- Stats Cards -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div class="card p-4">
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('leaderboard.totalSpending') }}</p>
            <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">${{ formatCost(data.total_actual_cost) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('leaderboard.totalRequests') }}</p>
            <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ formatNumber(data.total_requests) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('leaderboard.totalTokens') }}</p>
            <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ formatTokens(data.total_tokens) }}</p>
          </div>
        </div>

        <!-- Current Ranking Period -->
        <div class="flex items-center justify-between text-sm text-gray-500 dark:text-gray-400">
          <span>{{ t('leaderboard.currentRanking') }}: {{ data.start_date }} ~ {{ data.end_date }}</span>
          <span>{{ t('leaderboard.updatedAt') }} {{ data.updated_at }}</span>
        </div>

        <!-- Ranking List -->
        <div class="card divide-y divide-gray-100 dark:divide-dark-700">
          <div
            v-for="(item, index) in data.ranking"
            :key="item.user_id"
            class="flex items-center gap-4 px-4 py-3 sm:px-6 transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/50"
            :class="{ 'bg-amber-50/50 dark:bg-amber-900/10': index < 3 }"
          >
            <!-- Rank -->
            <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center">
              <span
                v-if="index < 3"
                :class="[
                  'flex h-8 w-8 items-center justify-center rounded-full text-sm font-bold text-white',
                  index === 0 ? 'bg-amber-500' : index === 1 ? 'bg-gray-400' : 'bg-amber-700'
                ]"
              >
                {{ item.rank }}
              </span>
              <span v-else class="text-sm font-medium text-gray-500 dark:text-gray-400">
                {{ item.rank }}
              </span>
            </div>

            <!-- User Info -->
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
                {{ item.username || item.email }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ formatNumber(item.requests) }} {{ t('leaderboard.requests') }} · {{ formatTokens(item.tokens) }} {{ t('leaderboard.token') }}
              </p>
            </div>

            <!-- Cost -->
            <div class="flex-shrink-0 text-right">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">${{ formatCost(item.actual_cost) }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.spending') }}</p>
            </div>
          </div>

          <!-- Empty state -->
          <div v-if="data.ranking.length === 0" class="px-6 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('leaderboard.noRanking') }}
          </div>
        </div>

        <!-- My Ranking -->
        <div class="card p-4 sm:p-6">
          <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.myRanking') }}</h3>
          <div class="flex items-center gap-4">
            <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/30">
              <span v-if="data.my_rank.rank" class="text-lg font-bold text-blue-600 dark:text-blue-400">
                #{{ data.my_rank.rank }}
              </span>
              <span v-else class="text-lg font-bold text-gray-400">-</span>
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ user?.username || user?.email || '-' }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                <template v-if="data.my_rank.rank">
                  {{ formatNumber(data.my_rank.requests) }} {{ t('leaderboard.requests') }} · {{ formatTokens(data.my_rank.tokens) }} {{ t('leaderboard.token') }}
                </template>
                <template v-else>
                  {{ t('leaderboard.noRanking') }}
                </template>
              </p>
            </div>
            <div class="flex-shrink-0 text-right">
              <p class="text-lg font-bold text-gray-900 dark:text-white">
                ${{ formatCost(data.my_rank.actual_cost) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.spending') }}</p>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { getLeaderboard, type LeaderboardResponse } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)

const loading = ref(false)
const period = ref<'day' | 'week' | 'month'>('week')
const data = ref<LeaderboardResponse | null>(null)

const periods = computed(() => [
  { value: 'day' as const, label: t('leaderboard.daily') },
  { value: 'week' as const, label: t('leaderboard.weekly') },
  { value: 'month' as const, label: t('leaderboard.monthly') },
])

function formatCost(v: number): string {
  if (v >= 10000) return (v / 1000).toFixed(2) + 'K'
  if (v >= 1000) return v.toFixed(2)
  if (v >= 1) return v.toFixed(2)
  if (v >= 0.01) return v.toFixed(4)
  return v.toFixed(6)
}

function formatNumber(v: number): string {
  return v.toLocaleString()
}

function formatTokens(v: number): string {
  if (v >= 1e12) return (v / 1e12).toFixed(2) + 'T'
  if (v >= 1e9) return (v / 1e9).toFixed(2) + 'B'
  if (v >= 1e6) return (v / 1e6).toFixed(2) + 'M'
  if (v >= 1e3) return (v / 1e3).toFixed(2) + 'K'
  return v.toString()
}

async function loadData() {
  loading.value = true
  try {
    data.value = await getLeaderboard({ period: period.value, limit: 50 })
  } catch (e) {
    console.error('Failed to load leaderboard:', e)
  } finally {
    loading.value = false
  }
}

onMounted(() => loadData())
</script>
