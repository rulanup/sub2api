<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] pb-8">
      <header class="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.16em] text-primary-600 dark:text-primary-400">Security Audit</p>
          <h1 class="mt-1 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{{ t('admin.promptErrorRecords.title') }}</h1>
          <p class="mt-2 max-w-3xl text-sm text-gray-500 dark:text-dark-300">{{ t('admin.promptErrorRecords.description') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ t('common.refresh') }}</button>
        </div>
      </header>

      <!-- Filters -->
      <div class="card mb-4 p-4">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.promptErrorRecords.filters.keyword') }}</label>
            <input v-model="filters.keyword" type="text" :placeholder="t('admin.promptErrorRecords.filters.keywordPlaceholder')" class="input input-sm w-full" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.promptErrorRecords.filters.model') }}</label>
            <input v-model="filters.model" type="text" :placeholder="t('admin.promptErrorRecords.filters.modelPlaceholder')" class="input input-sm w-full" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.promptErrorRecords.filters.errorStatus') }}</label>
            <input v-model="filters.error_status" type="number" placeholder="e.g. 400" class="input input-sm w-full" />
          </div>
          <div class="flex items-end gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="showMore = !showMore">{{ showMore ? '▲ ' + t('admin.promptErrorRecords.filters.more') : '▼ ' + t('admin.promptErrorRecords.filters.more') }}</button>
          </div>
        </div>
        <div v-if="showMore" class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <input v-model="filters.group_id" placeholder="Group ID" class="input input-sm" />
          <input v-model="filters.user_id" placeholder="User ID" class="input input-sm" />
          <input v-model="filters.api_key_id" placeholder="API Key ID" class="input input-sm" />
          <input v-model="filters.request_id" :placeholder="t('admin.promptErrorRecords.filters.requestId')" class="input input-sm" />
          <input v-model="filters.prompt_hash" :placeholder="t('admin.promptErrorRecords.filters.promptHash')" class="input input-sm" />
          <input v-model="filters.start_at" type="datetime-local" class="input input-sm" />
          <input v-model="filters.end_at" type="datetime-local" class="input input-sm" />
        </div>
        <div class="mt-3 flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-primary btn-sm" :disabled="loading" @click="applyFilters">{{ t('admin.promptErrorRecords.filters.search') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="resetFilters">{{ t('admin.promptErrorRecords.filters.reset') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="exporting" @click="doExport">{{ exporting ? 'Exporting…' : t('admin.promptErrorRecords.filters.export') }}</button>
          <span v-if="total" class="ml-2 text-xs text-gray-500">Total {{ total }} · Page {{ page }}/{{ pages }}</span>
        </div>
      </div>

      <!-- Actions -->
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <span v-if="selectedIds.size" class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.promptErrorRecords.table.selected', { count: selectedIds.size }) }}</span>
        <button type="button" class="btn btn-danger btn-sm" :disabled="!selectedIds.size || loading" @click="confirmBatchDelete">{{ t('admin.promptErrorRecords.table.deleteSelected') }}</button>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="openFilterDelete">{{ t('admin.promptErrorRecords.table.deleteByFilter') }}</button>
      </div>

      <!-- Table -->
      <div class="card overflow-hidden">
        <div v-if="loading" class="p-8 text-center text-sm text-gray-500">Loading…</div>
        <div v-else-if="error" class="p-6 text-sm text-red-600">{{ error }}</div>
        <div v-else-if="!items.length" class="p-8 text-center text-sm text-gray-500">{{ t('admin.promptErrorRecords.table.empty') }}</div>
        <div v-else class="overflow-auto">
          <table class="min-w-full text-sm">
            <thead class="bg-gray-50 text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-3 py-2"><input type="checkbox" :checked="allChecked" @change="toggleAll" /></th>
                <th class="px-3 py-2 text-left">{{ t('admin.promptErrorRecords.table.time') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.promptErrorRecords.table.model') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.promptErrorRecords.table.errorStatus') }}</th>
                <th class="px-3 py-2 text-left">Request ID</th>
                <th class="px-3 py-2 text-left">{{ t('admin.promptErrorRecords.table.promptPreview') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.promptErrorRecords.table.errorBody') }}</th>
                <th class="px-3 py-2 text-left">触发用户 / User</th>
                <th class="px-3 py-2 text-right">{{ t('admin.promptErrorRecords.table.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="row in items" :key="row.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/50">
                <td class="px-3 py-2"><input type="checkbox" :checked="selectedIds.has(row.id)" @change="toggleOne(row.id)" /></td>
                <td class="px-3 py-2 whitespace-nowrap text-xs text-gray-600 dark:text-dark-300">{{ formatTime(row.created_at) }}</td>
                <td class="px-3 py-2 whitespace-nowrap font-medium">{{ row.model || '-' }}</td>
                <td class="px-3 py-2 whitespace-nowrap"><span class="rounded bg-red-50 px-2 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-950/40">{{ row.error_status }}</span></td>
                <td class="px-3 py-2 whitespace-nowrap text-xs font-mono">{{ row.request_id || '-' }}</td>
                <td class="px-3 py-2 max-w-[280px] truncate text-xs" :title="row.full_prompt">{{ truncate(row.full_prompt, 120) }}</td>
                <td class="px-3 py-2 max-w-[220px] truncate text-xs text-red-600 dark:text-red-300" :title="row.error_body">{{ truncate(row.error_body, 80) }}</td>
                <td class="px-3 py-2 text-xs">
                  <div class="space-y-0.5">
                    <div class="font-medium text-gray-900 dark:text-white flex items-center gap-1">
                      <span>{{ row.username_snapshot || '—' }}</span>
                      <span class="rounded bg-gray-100 px-1 py-0.5 text-[10px] text-gray-600 dark:bg-dark-700 dark:text-dark-300">ID:{{ row.user_id ?? '-' }}</span>
                      <button v-if="row.user_id" type="button" class="text-primary-600 hover:text-primary-700 dark:text-primary-400 text-[10px]" @click="filters.user_id = String(row.user_id); applyFilters()">筛选</button>
                    </div>
                    <div class="text-gray-500 dark:text-dark-400 truncate max-w-[200px]" :title="row.user_email_snapshot">{{ row.user_email_snapshot || '—' }}</div>
                    <div class="text-gray-500 dark:text-dark-400">API Key: {{ row.api_key_name_snapshot || '—' }} <span class="text-[10px]">#{{ row.api_key_id ?? '-' }}</span></div>
                    <div class="text-gray-500 dark:text-dark-400">分组: {{ row.group_name || '—' }} <span class="text-[10px]">#{{ row.group_id ?? '-' }}</span></div>
                  </div>
                </td>
                <td class="px-3 py-2 whitespace-nowrap text-right">
                  <button type="button" class="btn btn-secondary btn-xs mr-1" @click="openDetail(row)">{{ t('admin.promptErrorRecords.table.view') }}</button>
                  <button type="button" class="btn btn-danger btn-xs" @click="confirmSingleDelete(row)">{{ t('admin.promptErrorRecords.table.delete') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <!-- Pagination -->
        <div v-if="pages > 1" class="flex items-center justify-between border-t border-gray-100 px-4 py-3 dark:border-dark-700">
          <div class="text-xs text-gray-500">Total {{ total }}</div>
          <div class="flex items-center gap-1">
            <button type="button" class="btn btn-secondary btn-xs" :disabled="page <= 1" @click="goPage(page - 1)">Prev</button>
            <span class="px-2 text-xs">{{ page }} / {{ pages }}</span>
            <button type="button" class="btn btn-secondary btn-xs" :disabled="page >= pages" @click="goPage(page + 1)">Next</button>
          </div>
          <select v-model.number="pageSize" class="input input-xs w-24" @change="applyFilters">
            <option :value="20">20 / page</option>
            <option :value="50">50 / page</option>
            <option :value="100">100 / page</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Detail Drawer -->
    <div v-if="detail" class="fixed inset-0 z-40 flex">
      <div class="flex-1 bg-black/30" @click="detail = null"></div>
      <div class="flex w-full max-w-2xl flex-col overflow-hidden bg-white shadow-xl dark:bg-dark-900">
        <div class="flex items-center justify-between border-b border-gray-200 px-5 py-3 dark:border-dark-700">
          <h2 class="text-sm font-semibold">{{ t('admin.promptErrorRecords.detail.title') }} #{{ detail.id }}</h2>
          <button type="button" class="btn btn-secondary btn-xs" @click="detail = null">✕</button>
        </div>
        <div class="flex-1 overflow-auto p-5 text-sm">
          <div class="mb-4 rounded-lg border border-primary-200 bg-primary-50 p-3 dark:border-primary-900/50 dark:bg-primary-950/20">
            <p class="mb-2 text-xs font-semibold text-primary-800 dark:text-primary-300">触发用户 / Trigger User — 便于排查</p>
            <div class="grid grid-cols-2 gap-2 text-xs">
              <div><span class="font-semibold">用户:</span> {{ detail.username_snapshot || '—' }} <span class="rounded bg-white px-1 py-0.5 text-[10px] text-gray-600 dark:bg-dark-700">ID:{{ detail.user_id ?? '-' }}</span></div>
              <div><span class="font-semibold">邮箱:</span> <span class="truncate" :title="detail.user_email_snapshot">{{ detail.user_email_snapshot || '—' }}</span></div>
              <div><span class="font-semibold">API Key:</span> {{ detail.api_key_name_snapshot || '—' }} <span class="text-[10px]">#{{ detail.api_key_id ?? '-' }}</span></div>
              <div><span class="font-semibold">分组:</span> {{ detail.group_name || '—' }} <span class="text-[10px]">#{{ detail.group_id ?? '-' }}</span></div>
            </div>
            <div class="mt-2 flex gap-2">
              <button v-if="detail.user_id" type="button" class="btn btn-secondary btn-xs" @click="filters.user_id = String(detail.user_id); detail = null; applyFilters()">按此用户筛选</button>
              <a v-if="detail.user_id" :href="`/admin/users?search=${detail.user_id}`" target="_blank" class="btn btn-secondary btn-xs">查看用户</a>
            </div>
          </div>
          <div class="mb-4 grid grid-cols-2 gap-3 text-xs">
            <div><span class="font-semibold">{{ t('admin.promptErrorRecords.detail.requestId') }}:</span> {{ detail.request_id }}</div>
            <div><span class="font-semibold">{{ t('admin.promptErrorRecords.detail.promptHash') }}:</span> {{ detail.prompt_hash }}</div>
            <div><span class="font-semibold">{{ t('admin.promptErrorRecords.detail.provider') }}:</span> {{ detail.provider }}/{{ detail.endpoint }}</div>
            <div><span class="font-semibold">{{ t('admin.promptErrorRecords.detail.errorStatus') }}:</span> {{ detail.error_status }}</div>
            <div><span class="font-semibold">{{ t('admin.promptErrorRecords.detail.model') }}:</span> {{ detail.model }}</div>
            <div><span class="font-semibold">{{ t('admin.promptErrorRecords.detail.createdAt') }}:</span> {{ formatTime(detail.created_at) }}</div>
          </div>
          <div class="mb-4">
            <p class="mb-1 text-xs font-semibold">{{ t('admin.promptErrorRecords.detail.fullPrompt') }}</p>
            <p class="mb-1 text-xs text-gray-500">{{ t('admin.promptErrorRecords.detail.fullPromptHint') }}</p>
            <pre class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-gray-50 p-3 text-xs dark:bg-dark-800">{{ detail.full_prompt }}</pre>
          </div>
          <div>
            <p class="mb-1 text-xs font-semibold">{{ t('admin.promptErrorRecords.detail.errorBody') }}</p>
            <pre class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-red-50 p-3 text-xs text-red-700 dark:bg-red-950/30">{{ detail.error_body }}</pre>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete confirm -->
    <div v-if="confirmDialog.show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-dark-900">
        <h3 class="text-sm font-semibold">{{ t('admin.promptErrorRecords.deleteDialog.title') }}</h3>
        <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">{{ confirmDialog.message }}</p>
        <div v-if="confirmDialog.preview" class="mt-3 rounded bg-amber-50 p-3 text-xs dark:bg-amber-950/30">
          <p>{{ t('admin.promptErrorRecords.deleteDialog.matched', { count: confirmDialog.preview.matched_count }) }}</p>
          <p class="mt-1 text-gray-500">ID ≤ {{ confirmDialog.preview.snapshot_max_id }} · {{ confirmDialog.preview.filter_hash.slice(0, 8) }}…</p>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="confirmDialog.show = false">{{ t('admin.promptErrorRecords.deleteDialog.cancel') }}</button>
          <button type="button" class="btn btn-danger btn-sm" :disabled="deleting" @click="executeConfirm">{{ t('admin.promptErrorRecords.deleteDialog.confirm') }}</button>
        </div>
      </div>
    </div>

    <!-- Filter delete dialog -->
    <div v-if="filterDelete.show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-dark-900">
        <h3 class="text-sm font-semibold">{{ t('admin.promptErrorRecords.deleteDialog.byFilterTitle') }}</h3>
        <p class="mt-1 text-xs text-gray-500">{{ t('admin.promptErrorRecords.deleteDialog.byFilterDesc') }}</p>
        <p v-if="filterDelete.preview" class="mt-3 text-sm font-medium text-amber-700 dark:text-amber-300">{{ t('admin.promptErrorRecords.deleteDialog.matched', { count: filterDelete.preview.matched_count }) }}</p>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="filterDelete.loading" @click="filterDelete.show = false">{{ t('admin.promptErrorRecords.deleteDialog.cancel') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="filterDelete.loading" @click="doPreviewFilterDelete">{{ filterDelete.loading ? t('admin.promptErrorRecords.deleteDialog.previewing') : t('admin.promptErrorRecords.deleteDialog.preview') }}</button>
          <button type="button" class="btn btn-danger btn-sm" :disabled="!filterDelete.preview || deleting" @click="executeFilterDelete">{{ t('admin.promptErrorRecords.deleteDialog.confirm') }}</button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { promptErrorAPI } from './api'
import type { PromptErrorFilters, PromptErrorRecord, PromptErrorDeletePreview } from './types'

const { t } = useI18n()

const filters = reactive<PromptErrorFilters>({
  keyword: '',
  model: '',
  error_status: '',
  group_id: '',
  user_id: '',
  api_key_id: '',
  request_id: '',
  prompt_hash: '',
  start_at: '',
  end_at: '',
})
const showMore = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const pages = ref(0)
const items = ref<PromptErrorRecord[]>([])
const loading = ref(false)
const error = ref('')
const exporting = ref(false)
const deleting = ref(false)
const selectedIds = ref<Set<number>>(new Set())
const detail = ref<PromptErrorRecord | null>(null)
const confirmDialog = reactive<{ show: boolean; message: string; preview?: PromptErrorDeletePreview | null; action?: () => Promise<void> }>({ show: false, message: '', preview: null })
const filterDelete = reactive<{ show: boolean; loading: boolean; preview: PromptErrorDeletePreview | null }>({ show: false, loading: false, preview: null })

const allChecked = computed(() => items.value.length > 0 && items.value.every(r => selectedIds.value.has(r.id)))

function truncate(s: string, n: number) {
  if (!s) return '-'
  return s.length > n ? s.slice(0, n) + '…' : s
}
function formatTime(s: string) {
  if (!s) return '-'
  try { return new Date(s).toLocaleString() } catch { return s }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await promptErrorAPI.listRecords(filters as any, page.value, pageSize.value)
    items.value = data.items
    total.value = data.total
    pages.value = data.pages
  } catch (e: any) {
    error.value = e?.response?.data?.msg || e?.message || t('admin.promptErrorRecords.messages.loadFailed')
  } finally {
    loading.value = false
  }
}
function applyFilters() {
  page.value = 1
  selectedIds.value = new Set()
  load()
}
function resetFilters() {
  Object.assign(filters, { keyword: '', model: '', error_status: '', group_id: '', user_id: '', api_key_id: '', request_id: '', prompt_hash: '', start_at: '', end_at: '' })
  applyFilters()
}
function goPage(p: number) {
  page.value = p
  load()
}
function toggleOne(id: number) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id); else next.add(id)
  selectedIds.value = next
}
function toggleAll() {
  if (allChecked.value) selectedIds.value = new Set()
  else selectedIds.value = new Set(items.value.map(r => r.id))
}
function openDetail(row: PromptErrorRecord) {
  detail.value = row
}
function confirmSingleDelete(row: PromptErrorRecord) {
  confirmDialog.message = t('admin.promptErrorRecords.deleteDialog.single')
  confirmDialog.preview = null
  confirmDialog.action = async () => {
    deleting.value = true
    try { await promptErrorAPI.deleteRecord(row.id); await load(); selectedIds.value.delete(row.id) } catch (e: any) { alert(e?.response?.data?.msg || t('admin.promptErrorRecords.messages.deleteFailed')) } finally { deleting.value = false }
  }
  confirmDialog.show = true
}
function confirmBatchDelete() {
  const ids = Array.from(selectedIds.value)
  confirmDialog.message = t('admin.promptErrorRecords.deleteDialog.batch', { count: ids.length })
  confirmDialog.preview = null
  confirmDialog.action = async () => {
    deleting.value = true
    try { await promptErrorAPI.batchDeleteRecords(ids); await load(); selectedIds.value = new Set() } catch (e: any) { alert(e?.response?.data?.msg || t('admin.promptErrorRecords.messages.deleteFailed')) } finally { deleting.value = false }
  }
  confirmDialog.show = true
}
function openFilterDelete() {
  filterDelete.show = true
  filterDelete.preview = null
}
async function doPreviewFilterDelete() {
  if (!filters.start_at || !filters.end_at) { alert('请选择开始与结束时间'); return }
  filterDelete.loading = true
  try {
    const p = await promptErrorAPI.previewDelete(filters as any)
    filterDelete.preview = p
  } catch (e: any) { alert(e?.response?.data?.msg || t('admin.promptErrorRecords.messages.previewFailed')) } finally { filterDelete.loading = false }
}
async function executeConfirm() {
  if (!confirmDialog.action) return
  const act = confirmDialog.action
  confirmDialog.show = false
  await act()
}
async function executeFilterDelete() {
  if (!filterDelete.preview) return
  deleting.value = true
  try {
    await promptErrorAPI.deleteByFilter(filters as any, filterDelete.preview)
    filterDelete.show = false
    filterDelete.preview = null
    await load()
  } catch (e: any) { alert(e?.response?.data?.msg || t('admin.promptErrorRecords.messages.deleteFailed')) } finally { deleting.value = false }
}
async function doExport() {
  exporting.value = true
  try {
    const blob = await promptErrorAPI.exportCSV(filters as any)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `prompt-error-records-${new Date().toISOString().slice(0,19)}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  } catch (e: any) { alert(e?.response?.data?.msg || 'Export failed') } finally { exporting.value = false }
}

onMounted(load)
</script>
