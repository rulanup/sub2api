<template>
  <AppLayout>
    <div class="flex h-[calc(100vh-7rem)] flex-col gap-4">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('workbench.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('workbench.description') }}</p>
        </div>
        <router-link
          to="/keys"
          class="inline-flex items-center gap-1.5 rounded-lg bg-white px-3 py-2 text-sm font-medium text-gray-700 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-600 dark:hover:bg-dark-700"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
          </svg>
          {{ t('workbench.manageKeys') }}
        </router-link>
      </div>

      <!-- Tabs -->
      <div class="flex items-center gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          @click="activeTab = tab.value"
          :class="[
            'flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors',
            activeTab === tab.value
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'
          ]"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- Content -->
      <div class="min-h-0 flex-1">
        <ChatPanel v-if="activeTab === 'chat'" :api-keys="apiKeys" :loading-keys="loadingKeys" />
        <ImagePanel v-else-if="activeTab === 'image'" :api-keys="apiKeys" :loading-keys="loadingKeys" />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import type { ApiKey } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import ChatPanel from '@/components/workbench/ChatPanel.vue'
import ImagePanel from '@/components/workbench/ImagePanel.vue'

const { t } = useI18n()

const activeTab = ref<'chat' | 'image'>('chat')
const apiKeys = ref<ApiKey[]>([])
const loadingKeys = ref(false)

const tabs = computed(() => [
  { value: 'chat' as const, label: t('workbench.chat') },
  { value: 'image' as const, label: t('workbench.image') },
])

async function loadKeys() {
  loadingKeys.value = true
  try {
    const res = await keysAPI.list(1, 100, { status: 'active' })
    apiKeys.value = res.items || []
  } catch (e) {
    console.error('Failed to load API keys:', e)
  } finally {
    loadingKeys.value = false
  }
}

onMounted(() => loadKeys())
</script>
