<template>
  <div v-if="isEnabled" class="card p-4">
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('checkin.title') }}</h3>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
          <template v-if="isCheckedIn">
            {{ t('checkin.checkedIn') }} +${{ checkedAmount.toFixed(6) }}
          </template>
          <template v-else>
            {{ t('checkin.range', { min: minAmount.toFixed(2), max: maxAmount.toFixed(2) }) }}
          </template>
        </p>
      </div>
      <button
        @click="handleCheckin"
        :disabled="isCheckedIn || loading"
        :class="[
          'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
          isCheckedIn
            ? 'cursor-default bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-400'
            : 'bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50'
        ]"
      >
        <svg v-if="loading" class="inline h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
        <template v-else>{{ isCheckedIn ? ('✓ ' + t('checkin.done')) : t('checkin.button') }}</template>
      </button>
    </div>
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
import { getCheckinStatus, doCheckin } from '@/api/checkin'

const { t } = useI18n()

const loading = ref(false)
const showReward = ref(false)
const rewardAmount = ref(0)
const isEnabled = ref(false)
const isCheckedIn = ref(false)
const checkedAmount = ref(0)
const minAmount = ref(0.01)
const maxAmount = ref(0.10)

onMounted(async () => {
  try {
    const res = await getCheckinStatus()
    isEnabled.value = !!res?.enabled
    isCheckedIn.value = !!res?.checked_in
    checkedAmount.value = Number(res?.amount) || 0
    minAmount.value = Number(res?.min_amount) || 0.01
    maxAmount.value = Number(res?.max_amount) || 0.10
  } catch (e) {
    // Silently fail - card won't show
  }
})

async function handleCheckin() {
  if (isCheckedIn.value || loading.value) return
  loading.value = true
  try {
    const result = await doCheckin()
    const amt = Number(result?.amount) || 0
    isCheckedIn.value = true
    checkedAmount.value = amt
    rewardAmount.value = amt
    showReward.value = true
    setTimeout(() => { showReward.value = false }, 5000)
  } catch (e) {
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
