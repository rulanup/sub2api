<template>
  <AppLayout>
    <div class="mx-auto max-w-lg space-y-6">
      <!-- Header -->
      <div class="text-center">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('checkin.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('checkin.subtitle') }}</p>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <!-- Error -->
      <div v-else-if="error" class="card p-8 text-center">
        <p class="text-red-500 dark:text-red-400">{{ error }}</p>
        <button @click="loadStatus" class="mt-4 rounded-lg bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700">
          {{ t('checkin.retry') }}
        </button>
      </div>

      <!-- Disabled state -->
      <div v-else-if="!status?.enabled" class="card p-8 text-center">
        <p class="text-gray-400 dark:text-gray-500">{{ t('checkin.disabled') }}</p>
      </div>

      <!-- Checkin Content -->
      <template v-else>
        <!-- Checkin Card -->
        <div class="card overflow-hidden">
          <!-- Top decorative bar -->
          <div class="h-2 bg-gradient-to-r from-blue-500 to-purple-500"></div>

          <div class="p-6 text-center">
            <!-- Status icon -->
            <div class="mx-auto mb-4 flex h-20 w-20 items-center justify-center rounded-full"
              :class="status.checked_in ? 'bg-green-100 dark:bg-green-900/30' : 'bg-blue-100 dark:bg-blue-900/30'"
            >
              <template v-if="showReward">
                <span class="text-4xl">🎉</span>
              </template>
              <template v-else-if="status.checked_in">
                <svg class="h-10 w-10 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </template>
              <template v-else>
                <svg class="h-10 w-10 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </template>
            </div>

            <!-- Status text -->
            <template v-if="showReward">
              <h2 class="text-xl font-bold text-green-600 dark:text-green-400">{{ t('checkin.success') }}</h2>
              <p class="mt-2 text-3xl font-bold text-gray-900 dark:text-white">+${{ rewardAmount.toFixed(6) }}</p>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('checkin.addedToBalance') }}</p>
            </template>
            <template v-else-if="status.checked_in">
              <h2 class="text-xl font-bold text-green-600 dark:text-green-400">{{ t('checkin.doneToday') }}</h2>
              <p class="mt-2 text-3xl font-bold text-gray-900 dark:text-white">+${{ (status.amount || 0).toFixed(6) }}</p>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('checkin.comeBackTomorrow') }}</p>
            </template>
            <template v-else>
              <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('checkin.notCheckedIn') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('checkin.range', { min: (status.min_amount || 0).toFixed(2), max: (status.max_amount || 0).toFixed(2) }) }}
              </p>
            </template>

            <!-- Button -->
            <button
              @click="handleCheckin"
              :disabled="status.checked_in || checkingIn"
              :class="[
                'mt-6 w-full rounded-xl px-6 py-3 text-base font-semibold transition-all',
                status.checked_in
                  ? 'cursor-default bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-400'
                  : 'bg-gradient-to-r from-blue-600 to-purple-600 text-white shadow-lg hover:from-blue-700 hover:to-purple-700 hover:shadow-xl disabled:opacity-50 active:scale-[0.98]'
              ]"
            >
              <template v-if="checkingIn">
                <svg class="mr-2 inline h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
                {{ t('checkin.checkingIn') }}
              </template>
              <template v-else-if="status.checked_in">
                ✓ {{ t('checkin.done') }}
              </template>
              <template v-else>
                {{ t('checkin.button') }}
              </template>
            </button>
          </div>
        </div>

        <!-- Info -->
        <div class="card p-4">
          <h3 class="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300">{{ t('checkin.rules') }}</h3>
          <ul class="space-y-1.5 text-xs text-gray-500 dark:text-gray-400">
            <li class="flex items-start gap-2">
              <span class="mt-0.5 text-blue-500">•</span>
              {{ t('checkin.rule1') }}
            </li>
            <li class="flex items-start gap-2">
              <span class="mt-0.5 text-blue-500">•</span>
              {{ t('checkin.rule2', { min: (status.min_amount || 0).toFixed(2), max: (status.max_amount || 0).toFixed(2) }) }}
            </li>
            <li class="flex items-start gap-2">
              <span class="mt-0.5 text-blue-500">•</span>
              {{ t('checkin.rule3') }}
            </li>
          </ul>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getCheckinStatus, doCheckin, type CheckinStatus } from '@/api/checkin'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const { t } = useI18n()

const loading = ref(true)
const checkingIn = ref(false)
const status = ref<CheckinStatus | null>(null)
const showReward = ref(false)
const rewardAmount = ref(0)
const error = ref<string | null>(null)

async function loadStatus() {
  loading.value = true
  error.value = null
  try {
    status.value = await getCheckinStatus()
  } catch (e: any) {
    console.warn('Failed to load checkin status:', e)
    error.value = e?.message || 'Failed to load checkin status'
  } finally {
    loading.value = false
  }
}

onMounted(() => loadStatus())

async function handleCheckin() {
  if (!status.value || status.value.checked_in || checkingIn.value) return
  checkingIn.value = true
  try {
    const result = await doCheckin()
    status.value.checked_in = true
    status.value.amount = result.amount
    rewardAmount.value = result.amount
    showReward.value = true
  } catch (e: any) {
    console.error('Checkin failed:', e)
    error.value = e?.message || 'Check-in failed'
  } finally {
    checkingIn.value = false
  }
}
</script>
