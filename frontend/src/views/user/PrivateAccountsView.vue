<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('privateAccounts.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('privateAccounts.description') }}</p>
        </div>
        <button @click="showCreateModal = true" class="btn btn-primary">
          <svg class="mr-1.5 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          {{ t('privateAccounts.addAccount') }}
        </button>
      </div>

      <!-- Account List -->
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="account in accounts"
          :key="account.id"
          class="card p-4 transition-all hover:shadow-md"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1">
              <div class="flex items-center gap-2">
                <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ account.name }}</span>
                <span
                  class="rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="account.status === 'active'
                    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                    : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'"
                >
                  {{ account.status === 'active' ? t('privateAccounts.active') : t('privateAccounts.disabled') }}
                </span>
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ account.platform }}</p>
            </div>
            <div class="flex items-center gap-1">
              <button
                @click="editAccount(account)"
                class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-300"
              >
                <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
                </svg>
              </button>
              <button
                @click="deleteAccount(account)"
                class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              >
                <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                </svg>
              </button>
            </div>
          </div>

          <!-- Groups -->
          <div v-if="account.group_ids && account.group_ids.length > 0" class="mt-3 flex flex-wrap gap-1">
            <span
              v-for="gid in account.group_ids"
              :key="gid"
              class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
            >
              {{ getGroupName(gid) }}
            </span>
          </div>
          <div v-else class="mt-3">
            <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('privateAccounts.noGroups') }}</span>
          </div>

          <!-- Notes -->
          <p v-if="account.notes" class="mt-2 text-xs text-gray-400 dark:text-gray-500">{{ account.notes }}</p>

          <!-- Last used -->
          <p class="mt-2 text-xs text-gray-400 dark:text-gray-500">
            {{ t('privateAccounts.lastUsed') }}: {{ account.last_used_at ? formatDate(account.last_used_at) : t('privateAccounts.never') }}
          </p>
        </div>

        <!-- Empty state -->
        <div
          v-if="accounts.length === 0 && !loading"
          class="col-span-full flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-12 dark:border-dark-600"
        >
          <svg class="mb-4 h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m6 4.125l2.25 2.25m0 0l2.25 2.25M12 13.875l2.25-2.25M12 13.875l-2.25 2.25M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('privateAccounts.empty') }}</p>
          <button @click="showCreateModal = true" class="btn btn-primary mt-4">
            {{ t('privateAccounts.addFirst') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showCreateModal || editingAccount" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-dark-800">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ editingAccount ? t('privateAccounts.editAccount') : t('privateAccounts.createAccount') }}
        </h2>

        <div class="mt-4 space-y-4">
          <div>
            <label class="input-label">{{ t('privateAccounts.name') }}</label>
            <input v-model="form.name" type="text" class="input" :placeholder="t('privateAccounts.namePlaceholder')" />
          </div>

          <div>
            <label class="input-label">{{ t('privateAccounts.platform') }}</label>
            <select v-model="form.platform" class="input">
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="gemini">Gemini</option>
            </select>
          </div>

          <div>
            <label class="input-label">{{ t('privateAccounts.apiKey') }}</label>
            <input v-model="form.api_key" type="password" class="input font-mono" :placeholder="t('privateAccounts.apiKeyPlaceholder')" />
          </div>

          <div>
            <label class="input-label">{{ t('privateAccounts.groups') }}</label>
            <div class="max-h-40 space-y-2 overflow-y-auto rounded-lg border border-gray-200 p-2 dark:border-dark-600">
              <label
                v-for="group in availableGroups"
                :key="group.id"
                class="flex cursor-pointer items-center gap-2 rounded p-1 hover:bg-gray-50 dark:hover:bg-dark-700"
              >
                <input
                  type="checkbox"
                  :value="group.id"
                  v-model="form.group_ids"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-sm text-gray-700 dark:text-gray-300">{{ group.name }}</span>
                <span class="text-xs text-gray-400">{{ group.platform }}</span>
              </label>
            </div>
          </div>

          <div>
            <label class="input-label">{{ t('privateAccounts.notes') }}</label>
            <textarea v-model="form.notes" class="input" rows="2" :placeholder="t('privateAccounts.notesPlaceholder')"></textarea>
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
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import AppLayout from '@/components/layout/AppLayout.vue'

const { t } = useI18n()

interface PrivateAccount {
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

interface Group {
  id: number
  name: string
  platform: string
}

const accounts = ref<PrivateAccount[]>([])
const availableGroups = ref<Group[]>([])
const loading = ref(false)
const saving = ref(false)
const showCreateModal = ref(false)
const editingAccount = ref<PrivateAccount | null>(null)

const form = ref({
  name: '',
  platform: 'openai',
  api_key: '',
  group_ids: [] as number[],
  notes: ''
})

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
    const { data } = await apiClient.get('/private-accounts')
    accounts.value = data || []
  } catch {
    accounts.value = []
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  try {
    const { data } = await apiClient.get('/private-accounts/available-groups')
    availableGroups.value = data || []
  } catch {
    availableGroups.value = []
  }
}

const editAccount = (account: PrivateAccount) => {
  editingAccount.value = account
  form.value = {
    name: account.name,
    platform: account.platform,
    api_key: '',
    group_ids: [...account.group_ids],
    notes: account.notes || ''
  }
}

const deleteAccount = async (account: PrivateAccount) => {
  if (!confirm(t('privateAccounts.deleteConfirm', { name: account.name }))) return
  try {
    await apiClient.delete(`/private-accounts/${account.id}`)
    await loadAccounts()
  } catch {
    // error
  }
}

const closeModal = () => {
  showCreateModal.value = false
  editingAccount.value = null
  form.value = { name: '', platform: 'openai', api_key: '', group_ids: [], notes: '' }
}

const saveAccount = async () => {
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      platform: form.value.platform,
      credentials: { api_key: form.value.api_key },
      group_ids: form.value.group_ids,
      notes: form.value.notes
    }

    if (editingAccount.value) {
      await apiClient.put(`/private-accounts/${editingAccount.value.id}`, payload)
    } else {
      await apiClient.post('/private-accounts', payload)
    }
    closeModal()
    await loadAccounts()
  } catch {
    // error
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadAccounts()
  loadGroups()
})
</script>
