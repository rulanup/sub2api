<template>
  <div v-if="status?.enabled" class="card p-4">
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('checkin.title') }}</h3>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
          <template v-if="status.checked_in">
            {{ t('checkin.checkedIn') }} +${{ status.amount.toFixed(6) }}
          </template>
          <template v-else>
            {{ t('checkin.range', { min: status.min_amount.toFixed(2), max: status.max_amount.toFixed(2) }) }}
          </template>
        </p>
      </div>
      <button
        @click="handleCheckin"
        :disabled="status.checked_in || loading"
        :class="[
          'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
          status.checked_in
            ? 'cursor-default bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-400'
            : 'bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50'
        ]"
      >
        <template v-if="loading">
          <svg class="inline h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
        </template>
        <template v-else-if="status.checked_in">
          ✓ {{ t('checkin.done') }}
        </template>
        <template v-else>
          {{ t('checkin.button') }}
        </template>
      </button>
    </div>
    <!-- Success animation -->
    <Transition name="fade">
      <div v-if="showReward" class="mt-3 rounded-lg bg-green-50 p-3 dark:bg-green-900/20">
        <p class="text-sm font-medium text-green-700 dark:text-green-300">
          🎉 {{ t('checkin.reward', { amount: rewardAmount.toFixed(6) }) }}
        </p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getCheckinStatus, doCheckin, type CheckinStatus } from '@/api/checkin'

const { t } = useI18n()

const status = ref<CheckinStatus | null>(null)
const loading = ref(false)
const showReward = ref(false)
const rewardAmount = ref(0)

onMounted(async () => {
  try {
    status.value = await getCheckinStatus()
  } catch (e) {
    console.warn('Failed to load checkin status:', e)
  }
})

async function handleCheckin() {
  if (!status.value || status.value.checked_in || loading.value) return
  loading.value = true
  try {
    const result = await doCheckin()
    status.value.checked_in = true
    status.value.amount = result.amount
    rewardAmount.value = result.amount
    showReward.value = true
    setTimeout(() => { showReward.value = false }, 5000)
  } catch (e: any) {
    console.error('Checkin failed:', e)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.3s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
