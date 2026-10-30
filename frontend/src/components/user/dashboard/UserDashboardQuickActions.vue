<template>
  <div class="card">
    <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <h2 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('dashboard.quickActions') }}</h2>
    </div>
    <div class="divide-y divide-gray-100 dark:divide-dark-700">
      <button @click="router.push('/keys')" class="group flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/50">
        <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-400">
          <Icon name="key" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-950 dark:text-white">{{ t('dashboard.createApiKey') }}</p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.generateNewKey') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-gray-600 dark:text-dark-500 dark:group-hover:text-gray-300"
        />
      </button>

      <button @click="router.push('/usage')" class="group flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/50">
        <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-400">
          <Icon name="chart" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-950 dark:text-white">{{ t('dashboard.viewUsage') }}</p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.checkDetailedLogs') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-gray-600 dark:text-dark-500 dark:group-hover:text-gray-300"
        />
      </button>

      <button v-if="canUseBatchImage" @click="router.push('/batch-image')" class="group flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/50">
        <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-400">
          <Icon name="sparkles" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-950 dark:text-white">{{ t('dashboard.batchImageAgent') }}</p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.batchImageAgentDesc') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-gray-600 dark:text-dark-500 dark:group-hover:text-gray-300"
        />
      </button>

      <button @click="router.push('/redeem')" class="group flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/50">
        <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-400">
          <Icon name="gift" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-950 dark:text-white">{{ t('dashboard.redeemCode') }}</p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.addBalanceWithCode') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-gray-600 dark:text-dark-500 dark:group-hover:text-gray-300"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
const router = useRouter()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
