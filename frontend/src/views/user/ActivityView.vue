<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-5">
      <header class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ status?.title || t('activity.title') }}</h1>
          <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">{{ status?.description || t('activity.subtitle') }}</p>
        </div>
        <div v-if="status?.start_at && status?.end_at" class="flex shrink-0 items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <Icon name="calendar" size="sm" />
          <span>{{ formatDate(status.start_at) }} - {{ formatDate(status.end_at) }}</span>
        </div>
      </header>

      <div v-if="loading" class="card flex min-h-96 items-center justify-center">
        <LoadingSpinner />
      </div>

      <div v-else-if="loadError" class="card p-8 text-center">
        <Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" />
        <p class="mt-3 text-sm text-red-600 dark:text-red-400">{{ loadError }}</p>
        <button type="button" class="btn btn-secondary mt-5" @click="loadActivity">
          <Icon name="refresh" size="sm" />
          {{ t('activity.retry') }}
        </button>
      </div>

      <template v-else-if="status">
        <div class="grid gap-5 lg:grid-cols-[minmax(0,3fr)_minmax(280px,2fr)]">
          <section class="card min-w-0 p-4 sm:p-6">
            <div class="mb-4 grid grid-cols-2 divide-x divide-gray-200 rounded border border-gray-200 bg-gray-50 dark:divide-dark-600 dark:border-dark-600 dark:bg-dark-800">
              <div class="px-3 py-2.5 text-center">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('activity.dailyRemaining') }}</div>
                <div class="mt-0.5 text-lg font-semibold text-gray-900 dark:text-white">{{ status.daily_remaining }} / {{ status.daily_limit }}</div>
              </div>
              <div class="px-3 py-2.5 text-center">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('activity.globalRemaining') }}</div>
                <div class="mt-0.5 text-lg font-semibold text-gray-900 dark:text-white">{{ status.global_remaining }} / {{ status.global_limit }}</div>
              </div>
            </div>

            <div ref="wheelContainer" class="wheel-container mx-auto">
              <LuckyWheel
                v-if="wheelSize > 0 && wheelPrizes.length"
                ref="wheelRef"
                :width="wheelSize"
                :height="wheelSize"
                :blocks="wheelBlocks"
                :prizes="wheelPrizes"
                :buttons="wheelButtons"
                :default-style="wheelStyle"
                :default-config="wheelConfig"
                @start="startDraw"
                @end="finishDraw"
              />
              <div v-else class="flex h-full items-center justify-center text-sm text-gray-500">{{ t('activity.noEligible') }}</div>
            </div>

            <div class="mt-4 min-h-[76px] text-center" aria-live="polite">
              <div v-if="revealedResult" data-testid="activity-result" class="rounded border border-amber-300 bg-amber-50 px-4 py-3 dark:border-amber-700 dark:bg-amber-950/30">
                <div class="text-xs font-medium uppercase text-amber-700 dark:text-amber-300">{{ t('activity.won') }}</div>
                <div class="mt-0.5 text-xl font-bold text-red-700 dark:text-red-300">{{ revealedResult.prize.label }}</div>
                <div v-if="revealedResult.prize.type === 'balance' && revealedResult.balance_after != null" class="mt-1 text-xs text-gray-600 dark:text-gray-300">
                  {{ t('activity.balanceAfter', { amount: formatAmount(revealedResult.balance_after) }) }}
                </div>
                <div v-else-if="revealedResult.prize.type === 'exclusive_group_access'" class="mt-1 text-xs text-gray-600 dark:text-gray-300">
                  {{ groupRewardDetail(revealedResult) }}
                </div>
              </div>
              <div v-if="refreshWarning" data-testid="activity-refresh-warning" class="mt-2 flex items-center justify-center gap-2 text-xs text-amber-700 dark:text-amber-300">
                <span>{{ refreshWarning }}</span>
                <button type="button" class="font-medium underline" @click="refreshAfterDraw">{{ t('activity.retryRefresh') }}</button>
              </div>
              <p v-if="!revealedResult && drawError" class="pt-3 text-sm text-red-600 dark:text-red-400">{{ drawError }}</p>
              <p v-else-if="!revealedResult" class="pt-3 text-sm font-medium" :class="canDraw ? 'text-gray-700 dark:text-gray-300' : 'text-gray-500 dark:text-gray-400'">
                {{ stateMessage }}
              </p>
            </div>

            <button
              type="button"
              data-testid="activity-draw-button"
              class="btn mt-2 w-full justify-center bg-red-700 text-white hover:bg-red-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-red-700 dark:hover:bg-red-600"
              :disabled="!canDraw || drawing"
              @click="startDraw"
            >
              <LoadingSpinner v-if="drawing" class="h-4 w-4" />
              <Icon v-else name="sparkles" size="sm" />
              {{ drawing ? t('activity.drawing') : t('activity.draw') }}
            </button>
          </section>

          <aside class="space-y-5">
            <section class="card p-5">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('activity.prizes') }}</h2>
              <ul class="mt-3 divide-y divide-gray-100 dark:divide-dark-700">
                <li v-for="(prize, index) in status.prizes" :key="prize.id" class="flex items-center gap-3 py-2.5">
                  <span class="h-3 w-3 shrink-0 rounded-sm border border-black/10" :style="{ backgroundColor: segmentColors[index % segmentColors.length] }"></span>
                  <span class="min-w-0 flex-1 truncate text-sm text-gray-800 dark:text-gray-200">{{ prize.label }}</span>
                  <span class="shrink-0 text-xs text-gray-500 dark:text-gray-400">{{ prizeDetail(prize) }}</span>
                </li>
              </ul>
            </section>

            <section class="card p-5">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('activity.history') }}</h2>
              <div v-if="historyLoading && history.length === 0" class="flex justify-center py-8"><LoadingSpinner /></div>
              <div v-if="historyError" data-testid="activity-history-error" class="py-4 text-center">
                <p class="text-sm text-red-600 dark:text-red-400">{{ historyError }}</p>
                <button type="button" class="btn btn-secondary btn-sm mt-3" @click="retryHistory">{{ t('activity.retry') }}</button>
              </div>
              <p v-else-if="!historyLoading && history.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('activity.emptyHistory') }}</p>
              <ul v-if="history.length" data-testid="activity-history-list" class="mt-3 divide-y divide-gray-100 dark:divide-dark-700">
                <li v-for="item in history" :key="item.id" class="flex items-center justify-between gap-3 py-2.5">
                  <div class="min-w-0">
                    <div class="truncate text-sm font-medium text-gray-800 dark:text-gray-200">{{ item.prize.label }}</div>
                    <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ formatDate(item.created_at) }}</div>
                  </div>
                  <span class="shrink-0 text-xs text-gray-500 dark:text-gray-400">{{ prizeDetail(item.prize) }}</span>
                </li>
              </ul>
            </section>
          </aside>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { LuckyWheel } from '@lucky-canvas/vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { activityAPI, createIdempotencyKey, type ActivityDrawRecord, type ActivityPrize, type ActivityStatus } from '@/api/activity'
import { useAuthStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

interface WheelHandle { play: () => void; stop: (index?: number) => void }

const { t, locale } = useI18n()
const authStore = useAuthStore()
const wheelRef = ref<WheelHandle | null>(null)
const wheelContainer = ref<HTMLElement | null>(null)
const wheelSize = ref(0)
const status = ref<ActivityStatus | null>(null)
const history = ref<ActivityDrawRecord[]>([])
const loading = ref(true)
const historyLoading = ref(true)
const historyError = ref('')
const drawing = ref(false)
const loadError = ref('')
const drawError = ref('')
const refreshWarning = ref('')
const pendingResult = ref<ActivityDrawRecord | null>(null)
const revealedResult = ref<ActivityDrawRecord | null>(null)
const suppressFinish = ref(false)
const reducedMotion = ref(false)
let resizeObserver: ResizeObserver | null = null
let disposed = false
let drawResetTimer: number | null = null
let pendingIdempotencyKey: string | null = null

const segmentColors = ['#b91c1c', '#f4c95d', '#f8fafc', '#dc6b55', '#e5a93d', '#f1f5f9']
const wheelBlocks = [{ padding: '10px', background: '#991b1b' }, { padding: '5px', background: '#d7a72f' }]
const wheelStyle = { fontColor: '#3f2b20', fontSize: '14px', fontWeight: '600' }
const wheelConfig = computed(() => ({
  gutter: '2px',
  speed: reducedMotion.value ? 6 : 20,
  accelerationTime: reducedMotion.value ? 120 : 900,
  decelerationTime: reducedMotion.value ? 180 : 1800,
}))
const wheelButtons = computed(() => [{
  radius: '27%',
  background: canDraw.value ? '#991b1b' : '#6b7280',
  pointer: true,
  fonts: [{ text: drawing.value ? t('activity.drawing') : t('activity.draw'), top: '-12px', fontColor: '#fff', fontSize: '15px', fontWeight: '700' }],
}])
const wheelPrizes = computed(() => (status.value?.prizes || []).map((prize, index) => ({
  id: prize.id,
  background: segmentColors[index % segmentColors.length],
  fonts: [{ text: prize.label, top: '18%', lengthLimit: '72%', lineClamp: 2, fontColor: index % segmentColors.length === 0 || index % segmentColors.length === 3 ? '#fff' : '#3f2b20' }],
})))

const canDraw = computed(() => Boolean(
  status.value?.enabled &&
  status.value.state === 'active' &&
  status.value.daily_remaining > 0 &&
  status.value.global_remaining > 0 &&
  status.value.prizes.length > 0,
))

const stateMessage = computed(() => {
  if (!status.value?.enabled || status.value?.state === 'disabled') return t('activity.states.disabled')
  if (status.value.state === 'upcoming') return t('activity.states.upcoming', { date: formatDate(status.value.start_at) })
  if (status.value.state === 'ended') return t('activity.states.ended')
  if (status.value.state === 'exhausted' || status.value.global_remaining <= 0) return t('activity.states.exhausted')
  if (status.value.daily_remaining <= 0) return t('activity.states.dailyExhausted')
  if (!status.value.prizes.length) return t('activity.noEligible')
  return t('activity.ready')
})

function formatDate(value?: string): string {
  if (!value) return ''
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function formatAmount(value: number): string {
  return new Intl.NumberFormat(locale.value, { minimumFractionDigits: 2, maximumFractionDigits: 8 }).format(value)
}

function prizeDetail(prize: ActivityPrize): string {
  if (prize.type === 'balance') return prize.amount == null ? t('activity.balance') : `+$${formatAmount(prize.amount)}`
  return t('activity.validityDays', { days: prize.validity_days || 0 })
}

function groupRewardDetail(result: ActivityDrawRecord): string {
  if (result.subscription_expires_after) return t('activity.expiresAt', { date: formatDate(result.subscription_expires_after) })
  return t('activity.validityDays', { days: result.prize.validity_days || 0 })
}

async function loadHistory(): Promise<void> {
  historyLoading.value = true
  historyError.value = ''
  try {
    history.value = (await activityAPI.getHistory(10)).items
  } catch (error) {
    historyError.value = extractI18nErrorMessage(error, t, 'activity.errors', t('activity.historyLoadFailed'))
    throw error
  } finally {
    historyLoading.value = false
  }
}

function retryHistory(): void {
  void loadHistory().catch(() => undefined)
}

async function loadActivity(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    status.value = await activityAPI.getStatus()
    await loadHistory().catch(() => undefined)
  } catch (error) {
    loadError.value = extractI18nErrorMessage(error, t, 'activity.errors', t('activity.loadFailed'))
  } finally {
    loading.value = false
    await nextTick()
    updateWheelSize()
  }
}

async function startDraw(): Promise<void> {
  if (!canDraw.value || drawing.value || disposed) return
  drawing.value = true
  drawError.value = ''
  revealedResult.value = null
  pendingResult.value = null
  suppressFinish.value = false
  wheelRef.value?.play()
  const idempotencyKey = pendingIdempotencyKey ?? createIdempotencyKey()
  pendingIdempotencyKey = idempotencyKey
  try {
    const response = await activityAPI.draw(idempotencyKey)
    pendingIdempotencyKey = null
    let index = status.value?.prizes.findIndex(prize => prize.id === response.result.prize.id) ?? -1
    if (index < 0 && status.value) {
      status.value = { ...status.value, prizes: [...status.value.prizes, response.result.prize] }
      index = status.value.prizes.length - 1
      await nextTick()
    }
    pendingResult.value = response.result
    if (status.value) {
      status.value.daily_remaining = response.daily_remaining
      status.value.daily_used = response.daily_used
      status.value.global_remaining = response.global_remaining
      status.value.global_used = response.global_used
    }
    wheelRef.value?.stop(index)
  } catch (error) {
    if (isDefinitiveResponse(error)) pendingIdempotencyKey = null
    suppressFinish.value = true
    drawError.value = extractI18nErrorMessage(error, t, 'activity.errors', t('activity.drawFailed'))
    wheelRef.value?.stop(0)
    drawResetTimer = window.setTimeout(() => {
      drawing.value = false
      drawResetTimer = null
    }, reducedMotion.value ? 250 : 1900)
  }
}

function finishDraw(): void {
  if (drawResetTimer != null) {
    window.clearTimeout(drawResetTimer)
    drawResetTimer = null
  }
  drawing.value = false
  if (suppressFinish.value || !pendingResult.value || disposed) {
    pendingResult.value = null
    return
  }
  revealedResult.value = pendingResult.value
  pendingResult.value = null
  void refreshAfterDraw()
}

function isDefinitiveResponse(error: unknown): boolean {
  if (typeof error !== 'object' || error === null) return false
  const status = (error as { status?: unknown }).status
  return typeof status === 'number' && status >= 400 && status < 500
}

async function refreshAfterDraw(): Promise<void> {
  refreshWarning.value = ''
  const results = await Promise.allSettled([refreshStatus(), loadHistory(), authStore.refreshUser()])
  if (results.some(result => result.status === 'rejected')) {
    refreshWarning.value = t('activity.refreshFailed')
  }
}

async function refreshStatus(): Promise<void> {
  status.value = await activityAPI.getStatus()
}

function updateWheelSize(): void {
  const width = wheelContainer.value?.clientWidth || 0
  wheelSize.value = Math.floor(Math.min(width, 480))
}

onMounted(() => {
  reducedMotion.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  resizeObserver = new ResizeObserver(updateWheelSize)
  if (wheelContainer.value) resizeObserver.observe(wheelContainer.value)
  void loadActivity()
})

onBeforeUnmount(() => {
  disposed = true
  resizeObserver?.disconnect()
  if (drawResetTimer != null) window.clearTimeout(drawResetTimer)
  if (drawing.value) wheelRef.value?.stop(0)
})
</script>

<style scoped>
.wheel-container {
  width: min(100%, 480px);
  aspect-ratio: 1;
  contain: layout size;
}

@media (prefers-reduced-motion: reduce) {
  :deep(canvas) { scroll-behavior: auto; }
}
</style>
