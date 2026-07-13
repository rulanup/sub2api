<template>
  <AppLayout>
    <div class="-m-4 flex h-[calc(100dvh-3.5rem)] flex-col md:-m-6 md:h-[calc(100dvh-4rem)] lg:-m-8 lg:h-[calc(100dvh-4.5rem)]">
      <div class="flex items-center justify-between gap-3 border-b border-gray-200/70 bg-white/70 px-3 py-2 backdrop-blur dark:border-dark-700 dark:bg-dark-900/70 sm:px-4">
        <div class="inline-flex rounded-full bg-gray-100 p-1 dark:bg-dark-800">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            type="button"
            class="rounded-full px-4 py-1.5 text-sm font-medium transition-all"
            :class="activeTab === tab.value
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="activeTab = tab.value"
          >
            {{ tab.label }}
          </button>
        </div>
        <div class="flex items-center gap-2">
          <span class="hidden text-xs text-gray-400 sm:inline">
            <template v-if="loadingKeys">{{ t('workbench.loadingKeys') }}</template>
            <template v-else-if="apiKeys.length === 0">{{ t('workbench.noKeysHint') }}</template>
            <template v-else>{{ t('workbench.keysReady', { count: apiKeys.length }) }}</template>
          </span>
          <router-link to="/keys" class="rounded-full px-3 py-1.5 text-xs font-medium text-gray-600 transition hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-800">
            {{ t('workbench.manageKeys') }}
          </router-link>
        </div>
      </div>

      <div class="min-h-0 flex-1 bg-white dark:bg-dark-950">
        <ChatPanel v-if="activeTab === 'chat'" :api-keys="apiKeys" :loading-keys="loadingKeys" />
        <ImagePanel v-else :api-keys="apiKeys" :loading-keys="loadingKeys" />
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
