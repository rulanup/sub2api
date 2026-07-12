<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('userAccounts.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('userAccounts.description') }}</p>
        </div>
        <button @click="openCreateModal" class="btn btn-primary">
          <svg class="mr-1.5 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          {{ t('userAccounts.addAccount') }}
        </button>
      </div>

      <!-- Filters -->
      <div class="card p-4">
        <div class="flex flex-wrap items-end gap-4">
          <div>
            <label class="input-label">{{ t('userAccounts.platform') }}</label>
            <Select v-model="filters.platform" :options="platformOptions" class="w-36" @change="loadAccounts" />
          </div>
          <div>
            <label class="input-label">{{ t('userAccounts.status') }}</label>
            <Select v-model="filters.status" :options="statusOptions" class="w-32" @change="loadAccounts" />
          </div>
        </div>
      </div>

      <!-- Account Table -->
      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('userAccounts.columns.name') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('userAccounts.columns.platform') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('userAccounts.columns.status') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('userAccounts.columns.groups') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('userAccounts.columns.lastUsed') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('userAccounts.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="loading">
                <td colspan="6" class="py-12 text-center">
                  <LoadingSpinner />
                </td>
              </tr>
              <tr v-else-if="accounts.length === 0">
                <td colspan="6" class="py-12 text-center text-sm text-gray-400">
                  {{ t('userAccounts.empty') }}
                </td>
              </tr>
              <tr
                v-for="account in accounts"
                :key="account.id"
                class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40"
              >
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-medium text-gray-900 dark:text-white">{{ account.name }}</span>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
                    :class="getPlatformClass(account.platform)">
                    {{ account.platform }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
                    :class="account.status === 'active'
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                      : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'">
                    {{ account.status === 'active' ? t('userAccounts.active') : t('userAccounts.disabled') }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <div v-if="account.group_ids && account.group_ids.length > 0" class="flex flex-wrap gap-1">
                    <span
                      v-for="gid in account.group_ids.slice(0, 3)"
                      :key="gid"
                      class="rounded bg-primary-50 px-1.5 py-0.5 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
                    >
                      {{ getGroupName(gid) }}
                    </span>
                    <span v-if="account.group_ids.length > 3" class="text-xs text-gray-400">
                      +{{ account.group_ids.length - 3 }}
                    </span>
                  </div>
                  <span v-else class="text-xs text-gray-400">-</span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                  {{ account.last_used_at ? formatDate(account.last_used_at) : '-' }}
                </td>
                <td class="px-4 py-3 text-right">
                  <div class="flex items-center justify-end gap-1">
                    <button
                      @click="openEditModal(account)"
                      class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-300"
                      :title="t('common.edit')"
                    >
                      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
                      </svg>
                    </button>
                    <button
                      @click="confirmDelete(account)"
                      class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                      :title="t('common.delete')"
                    >
                      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                      </svg>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-dark-800">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ editingAccount ? t('userAccounts.editAccount') : t('userAccounts.createAccount') }}
        </h2>

        <div class="mt-4 space-y-4">
          <div>
            <label class="input-label">{{ t('userAccounts.columns.name') }}</label>
            <input v-model="form.name" type="text" class="input" :placeholder="t('userAccounts.namePlaceholder')" />
          </div>

          <div>
            <label class="input-label">{{ t('userAccounts.columns.platform') }}</label>
            <Select v-model="form.platform" :options="platformOptions.filter(p => p.value)" class="w-full" />
          </div>

          <div>
            <label class="input-label">API Key</label>
            <input v-model="form.api_key" type="password" class="input font-mono text-sm" placeholder="sk-..." />
          </div>

          <div>
            <label class="input-label">{{ t('userAccounts.bindGroups') }}</label>
            <div class="max-h-40 space-y-1 overflow-y-auto rounded-lg border border-gray-200 p-2 dark:border-dark-600">
              <label
                v-for="group in availableGroups"
                :key="group.id"
                class="flex cursor-pointer items-center gap-2 rounded p-1.5 hover:bg-gray-50 dark:hover:bg-dark-700"
              >
                <input
                  type="checkbox"
                  :value="group.id"
                  v-model="form.group_ids"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-sm text-gray-700 dark:text-gray-300">{{ group.name }}</span>
                <span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-dark-600 dark:text-gray-400">{{ group.platform }}</span>
              </label>
            </div>
          </div>

          <div>
            <label class="input-label">{{ t('userAccounts.notes') }}</label>
            <textarea v-model="form.notes" class="input" rows="2" :placeholder="t('userAccounts.notesPlaceholder')"></textarea>
          </div>
        </div>

        <div class="mt-6 flex justify-end gap-3">
          <button @click="closeModal" class="btn btn-secondary">{{ t('common.cancel') }}</button>
          <button @click="saveAccount" class="btn btn-primary" :disabled="saving">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const { t } = useI18n()

interface UserAccount {
  id: number
  name: string
  platform: string
  status: string
  group_ids: number[]
  notes: string
  created_at: string
  updated_at: string
  last_used_at: string | null
}

interface GroupInfo {
  id: number
  name: string
  platform: string
}

const accounts = ref<UserAccount[]>([])
const availableGroups = ref<GroupInfo[]>([])
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const editingAccount = ref<UserAccount | null>(null)

const filters = reactive({
  platform: '',
  status: ''
})

const form = reactive({
  name: '',
  platform: 'openai',
  api_key: '',
  group_ids: [] as number[],
  notes: ''
})

const platformOptions = [
  { value: '', label: t('userAccounts.allPlatforms') },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' }
]

const statusOptions = [
  { value: '', label: t('userAccounts.allStatus') },
  { value: 'active', label: t('userAccounts.active') },
  { value: 'disabled', label: t('userAccounts.disabled') }
]

const getPlatformClass = (platform: string) => {
  switch (platform) {
    case 'openai': return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
    case 'anthropic': return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    case 'gemini': return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    default: return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
  }
}

const getGroupName = (id: number) => {
  const group = availableGroups.value.find(g => g.id === id)
  return group ? group.name : `#${id}`
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

const loadAccounts = async () => {
  loading.value = true
  try {
    const params = new URLSearchParams()
    if (filters.platform) params.set('platform', filters.platform)
    if (filters.status) params.set('status', filters.status)
    const { data } = await apiClient.get(`/user/accounts?${params.toString()}`)
    accounts.value = data || []
  } catch {
    accounts.value = []
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  try {
    const { data } = await apiClient.get('/user/accounts/available-groups')
    availableGroups.value = data || []
  } catch {
    availableGroups.value = []
  }
}

const openCreateModal = () => {
  editingAccount.value = null
  form.name = ''
  form.platform = 'openai'
  form.api_key = ''
  form.group_ids = []
  form.notes = ''
  showModal.value = true
}

const openEditModal = (account: UserAccount) => {
  editingAccount.value = account
  form.name = account.name
  form.platform = account.platform
  form.api_key = ''
  form.group_ids = [...account.group_ids]
  form.notes = account.notes || ''
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingAccount.value = null
}

const saveAccount = async () => {
  saving.value = true
  try {
    const payload: Record<string, unknown> = {
      name: form.name,
      platform: form.platform,
      group_ids: form.group_ids,
      notes: form.notes
    }
    if (form.api_key) {
      payload.credentials = { api_key: form.api_key }
    }

    if (editingAccount.value) {
      await apiClient.put(`/user/accounts/${editingAccount.value.id}`, payload)
    } else {
      payload.credentials = { api_key: form.api_key }
      await apiClient.post('/user/accounts', payload)
    }
    closeModal()
    await loadAccounts()
  } catch {
    // error handled by apiClient
  } finally {
    saving.value = false
  }
}

const confirmDelete = async (account: UserAccount) => {
  if (!confirm(t('userAccounts.deleteConfirm', { name: account.name }))) return
  try {
    await apiClient.delete(`/user/accounts/${account.id}`)
    await loadAccounts()
  } catch {
    // error handled by apiClient
  }
}

onMounted(() => {
  loadAccounts()
  loadGroups()
})
</script>
