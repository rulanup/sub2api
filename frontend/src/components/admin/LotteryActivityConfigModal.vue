<template>
  <BaseDialog :show="show" :title="t('admin.settings.features.activity.modalTitle')" width="wide" :close-on-escape="!saving" @close="close">
    <div v-if="loading" class="flex min-h-72 items-center justify-center"><LoadingSpinner /></div>
    <div v-else class="max-h-[70vh] space-y-6 overflow-y-auto pr-1">
      <div v-if="error" class="rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300">{{ error }}</div>

      <div class="flex items-center justify-between gap-4 border-b border-gray-100 pb-4 dark:border-dark-700">
        <div>
          <label id="activity-enabled-label" class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.settings.features.activity.enabled') }}</label>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.features.activity.enabledHint') }}</p>
        </div>
        <Toggle v-model="form.enabled" :aria-label="t('admin.settings.features.activity.enabled')" aria-labelledby="activity-enabled-label" />
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label for="activity-id" class="input-label">{{ t('admin.settings.features.activity.activityId') }}</label>
          <input id="activity-id" v-model="form.activity_id" class="input" maxlength="64" autocomplete="off" />
          <p class="input-hint">{{ t('admin.settings.features.activity.activityIdHint') }}</p>
        </div>
        <div>
          <label for="activity-title" class="input-label">{{ t('admin.settings.features.activity.titleLabel') }}</label>
          <input id="activity-title" v-model="form.title" class="input" />
        </div>
        <div class="sm:col-span-2">
          <label for="activity-description" class="input-label">{{ t('admin.settings.features.activity.descriptionLabel') }}</label>
          <textarea id="activity-description" v-model="form.description" class="input min-h-20"></textarea>
        </div>
        <div>
          <label for="activity-start-at" class="input-label">{{ t('admin.settings.features.activity.startAt') }}</label>
          <input id="activity-start-at" v-model="form.start_at" type="datetime-local" class="input" />
        </div>
        <div>
          <label for="activity-end-at" class="input-label">{{ t('admin.settings.features.activity.endAt') }}</label>
          <input id="activity-end-at" v-model="form.end_at" type="datetime-local" class="input" />
        </div>
        <div>
          <label for="activity-daily-limit" class="input-label">{{ t('admin.settings.features.activity.dailyLimit') }}</label>
          <input id="activity-daily-limit" v-model.number="form.daily_draw_limit" type="number" min="1" max="100" step="1" class="input" />
        </div>
        <div>
          <label for="activity-global-limit" class="input-label">{{ t('admin.settings.features.activity.globalLimit') }}</label>
          <input id="activity-global-limit" v-model.number="form.global_draw_limit" type="number" min="1" max="10000000" step="1" class="input" />
        </div>
      </div>

      <section class="border-t border-gray-100 pt-5 dark:border-dark-700">
        <div class="mb-3 flex items-center justify-between gap-3">
          <div>
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.features.activity.prizes') }}</h4>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.features.activity.prizeCount', { count: form.prizes.length }) }}</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="form.prizes.length >= 12" @click="addPrize">
            <Icon name="plus" size="sm" />
            {{ t('common.add') }}
          </button>
        </div>

        <div class="divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-600 dark:border-dark-600">
          <div v-for="(prize, index) in form.prizes" :key="index" class="space-y-3 py-4">
            <div class="flex items-center justify-between gap-3">
              <div class="inline-flex rounded border border-gray-200 p-0.5 dark:border-dark-600" role="group" :aria-label="t('admin.settings.features.activity.prizeType', { index: index + 1 })">
                <button type="button" :aria-pressed="prize.type === 'balance'" class="rounded-sm px-3 py-1.5 text-xs font-medium" :class="prize.type === 'balance' ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 dark:text-gray-300'" @click="setPrizeType(prize, 'balance')">{{ t('admin.settings.features.activity.balanceType') }}</button>
                <button type="button" :aria-pressed="prize.type === 'group'" class="rounded-sm px-3 py-1.5 text-xs font-medium" :class="prize.type === 'group' ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 dark:text-gray-300'" @click="setPrizeType(prize, 'group')">{{ t('admin.settings.features.activity.groupType') }}</button>
              </div>
              <button type="button" class="rounded p-2 text-gray-400 hover:bg-gray-100 hover:text-red-600 disabled:opacity-40 dark:hover:bg-dark-700" :disabled="form.prizes.length <= 2" :title="t('common.delete')" :aria-label="t('common.delete')" @click="removePrize(index)">
                <Icon name="trash" size="sm" />
              </button>
            </div>
            <div class="grid gap-3 sm:grid-cols-3">
              <div>
                <label :for="`activity-prize-${index}-id`" class="input-label">{{ t('admin.settings.features.activity.prizeId') }}</label>
                <input :id="`activity-prize-${index}-id`" v-model="prize.id" class="input" maxlength="64" />
              </div>
              <div>
                <label :for="`activity-prize-${index}-label`" class="input-label">{{ t('admin.settings.features.activity.prizeLabel') }}</label>
                <input
                  :id="`activity-prize-${index}-label`"
                  v-model="prize.label"
                  class="input"
                  :class="invalidPrizeLabelIndex === index ? 'border-red-500 focus:border-red-500 focus:ring-red-500' : ''"
                  :aria-invalid="invalidPrizeLabelIndex === index"
                  :aria-describedby="invalidPrizeLabelIndex === index ? `activity-prize-${index}-label-error` : undefined"
                  @input="clearPrizeLabelError(index)"
                />
                <p
                  v-if="invalidPrizeLabelIndex === index"
                  :id="`activity-prize-${index}-label-error`"
                  class="mt-1 text-xs text-red-600 dark:text-red-400"
                >
                  {{ t('admin.settings.features.activity.validation.prizeLabelAt', { index: index + 1 }) }}
                </p>
              </div>
              <div>
                <label :for="`activity-prize-${index}-weight`" class="input-label">{{ t('admin.settings.features.activity.weight') }}</label>
                <input :id="`activity-prize-${index}-weight`" v-model.number="prize.weight" type="number" min="1" step="1" class="input" />
              </div>
              <div v-if="prize.type === 'balance'" class="sm:col-span-3">
                <label :for="`activity-prize-${index}-amount`" class="input-label">{{ t('admin.settings.features.activity.amount') }}</label>
                <input :id="`activity-prize-${index}-amount`" v-model.number="prize.amount" type="number" min="0.00000001" max="1000000" step="0.00000001" class="input" />
              </div>
              <template v-else>
                <div class="sm:col-span-2">
                  <label :for="`activity-prize-${index}-group`" class="input-label">{{ t('admin.settings.features.activity.group') }}</label>
                  <select :id="`activity-prize-${index}-group`" v-model.number="prize.group_id" class="input">
                    <option :value="undefined" disabled>{{ t('admin.settings.features.activity.selectGroup') }}</option>
                    <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
                  </select>
                </div>
                <div>
                  <label :for="`activity-prize-${index}-validity`" class="input-label">{{ t('admin.settings.features.activity.validityDays') }}</label>
                  <input :id="`activity-prize-${index}-validity`" v-model.number="prize.validity_days" type="number" min="1" max="36500" step="1" class="input" />
                </div>
              </template>
            </div>
          </div>
        </div>
      </section>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="close">{{ t('common.cancel') }}</button>
      <button type="button" data-testid="activity-config-save" class="btn btn-primary" :disabled="!loaded || loading || saving" @click="save">
        <LoadingSpinner v-if="saving" class="h-4 w-4" />
        {{ t('common.save') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { nextTick, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { adminActivityAPI, type AdminActivityConfig, type AdminActivityPrize, type AdminActivityPrizeType } from '@/api/admin/activity'
import { groupsAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { configToForm, formToConfig, validateActivityConfig, type LotteryActivityConfigForm } from './lotteryActivityConfig'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; saved: [config: AdminActivityConfig] }>()
const { t } = useI18n()
const loading = ref(false)
const loaded = ref(false)
const saving = ref(false)
const error = ref('')
const groups = ref<AdminGroup[]>([])
const originalActivityId = ref('')
const invalidPrizeLabelIndex = ref<number | null>(null)
const form = reactive<LotteryActivityConfigForm>(emptyForm())
let requestGeneration = 0

function emptyForm(): LotteryActivityConfigForm {
  const start = new Date(Date.now() + 60 * 60 * 1000)
  start.setSeconds(0, 0)
  const end = new Date(start.getTime() + 7 * 24 * 60 * 60 * 1000)
  return configToForm({
    enabled: false,
    activity_id: '',
    title: '',
    description: '',
    start_at: start.toISOString(),
    end_at: end.toISOString(),
    daily_draw_limit: 1,
    global_draw_limit: 1000,
    prizes: [newPrize(1), newPrize(2)],
  })
}

function newPrize(index: number): AdminActivityPrize {
  return {
    id: `prize-${index}`,
    type: 'balance',
    label: t('admin.settings.features.activity.defaultPrizeLabel', { index }),
    weight: 1,
    amount: 1,
  }
}

function assignForm(value: LotteryActivityConfigForm): void {
  Object.assign(form, value)
  form.prizes = value.prizes.map(prize => ({ ...prize }))
}

async function load(): Promise<void> {
  const generation = ++requestGeneration
  resetState()
  loading.value = true
  error.value = ''
  try {
    const [config, allGroups] = await Promise.all([adminActivityAPI.getConfig(), groupsAPI.getAll()])
    if (generation !== requestGeneration || !props.show) return
    const usableGroups = allGroups.filter(group => group.status === 'active' && !group.is_private && group.subscription_type === 'subscription')
    groups.value = usableGroups
    const normalized = config.activity_id ? configToForm(config) : emptyForm()
    assignForm(normalized)
    originalActivityId.value = config.activity_id || ''
    loaded.value = true
  } catch (err) {
    if (generation !== requestGeneration || !props.show) return
    error.value = extractApiErrorMessage(err, t('admin.settings.features.activity.loadFailed'))
  } finally {
    if (generation === requestGeneration) loading.value = false
  }
}

function resetState(): void {
  loaded.value = false
  groups.value = []
  originalActivityId.value = ''
  error.value = ''
  invalidPrizeLabelIndex.value = null
  assignForm(emptyForm())
}

function addPrize(): void {
  if (form.prizes.length < 12) form.prizes.push(newPrize(form.prizes.length + 1))
}

function removePrize(index: number): void {
  if (form.prizes.length > 2) form.prizes.splice(index, 1)
}

function setPrizeType(prize: AdminActivityPrize, type: AdminActivityPrizeType): void {
  prize.type = type
  if (type === 'balance') {
    prize.amount = prize.amount ?? 1
    delete prize.group_id
    delete prize.validity_days
  } else {
    delete prize.amount
    prize.group_id = prize.group_id ?? groups.value[0]?.id
    prize.validity_days = prize.validity_days ?? 30
  }
}

async function save(): Promise<void> {
  if (!loaded.value || loading.value || saving.value) return
  const generation = requestGeneration
  error.value = ''
  const payload = formToConfig(form)
  const validationKey = validateActivityConfig(payload)
  if (validationKey) {
    if (validationKey === 'prizeLabel') {
      invalidPrizeLabelIndex.value = payload.prizes.findIndex(prize => !prize.label || Array.from(prize.label).length > 120)
      error.value = t('admin.settings.features.activity.validation.prizeLabelAt', { index: invalidPrizeLabelIndex.value + 1 })
      await nextTick()
      document.getElementById(`activity-prize-${invalidPrizeLabelIndex.value}-label`)?.focus()
      return
    }
    error.value = t(`admin.settings.features.activity.validation.${validationKey}`)
    return
  }
  invalidPrizeLabelIndex.value = null
  if (originalActivityId.value && originalActivityId.value !== payload.activity_id && !window.confirm(t('admin.settings.features.activity.activityIdChangeConfirm'))) return
  saving.value = true
  try {
    const saved = await adminActivityAPI.updateConfig(payload)
    if (generation !== requestGeneration || !props.show) return
    originalActivityId.value = saved.activity_id
    emit('saved', saved)
    saving.value = false
    close()
  } catch (err) {
    if (generation !== requestGeneration || !props.show) return
    error.value = extractApiErrorMessage(err, t('admin.settings.features.activity.saveFailed'))
  } finally {
    if (generation === requestGeneration) saving.value = false
  }
}

function close(): void {
  if (!saving.value) {
    requestGeneration += 1
    resetState()
    emit('close')
  }
}

function clearPrizeLabelError(index: number): void {
  if (invalidPrizeLabelIndex.value === index) invalidPrizeLabelIndex.value = null
}

watch(() => props.show, show => {
  if (show) void load()
  else {
    requestGeneration += 1
    saving.value = false
    loading.value = false
    resetState()
  }
}, { immediate: true })
</script>
