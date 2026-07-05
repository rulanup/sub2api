<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 class="text-xl font-bold tracking-wide text-gray-900 dark:text-white">模型广场</h1>
          <p class="mt-1 text-xs tracking-wide text-gray-500 dark:text-gray-400">浏览全部可用模型 · 实时价格与可用分组</p>
        </div>
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <a class="rounded-md border border-gray-200 px-3 py-1.5 text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-800" href="#">QQ群</a>
          <router-link class="rounded-md border border-gray-200 px-3 py-1.5 text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-800" to="/monitor">服务检测</router-link>
          <a class="rounded-md border border-gray-200 px-3 py-1.5 text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-800" href="#">教程</a>
        </div>
      </div>

      <div class="flex flex-col gap-3 pt-4 lg:flex-row lg:items-center">
        <div class="relative flex-1">
          <svg class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
          <input
            v-model="search"
            type="text"
            placeholder="搜索模型 / 分组 / 平台..."
            class="input h-11 rounded-lg pl-9"
          />
        </div>
        <div class="whitespace-nowrap text-xs text-gray-500 dark:text-gray-400">共 {{ filteredModels.length }} 个模型</div>
      </div>

      <div class="px-1 text-xs text-gray-500 dark:text-gray-400">
        💡 点击卡片上的分组，即可按该分组倍率查看实付价
      </div>

      <div class="grid gap-4 lg:grid-cols-[212px_minmax(0,1fr)] xl:grid-cols-[220px_minmax(0,1fr)]">
        <aside class="space-y-3 lg:sticky lg:top-4 lg:self-start">
          <div class="flex items-center justify-between">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">筛选</h2>
            <button type="button" class="text-xs text-gray-500 hover:text-primary-600" @click="clearFilters">清除</button>
          </div>

          <FilterSection title="厂商" :items="platformFilters" :active="activePlatforms" @toggle="togglePlatformFilter" />
          <FilterSection title="分组" :items="groupFilters" :active="activeGroups" @toggle="toggleGroupFilter" />
          <FilterSection title="计费模式" :items="billingFilters" :active="activeBillingModes" @toggle="toggleBillingFilter" />
        </aside>

        <main>
          <div v-if="loading" class="flex items-center justify-center py-12">
            <LoadingSpinner />
          </div>

          <div v-else-if="filteredModels.length > 0" class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
            <article
              v-for="model in filteredModels"
              :key="model.name"
              class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md dark:border-dark-600 dark:bg-dark-800/80"
            >
              <div class="flex items-start gap-3">
                <div class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-gray-50 ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-600">
                  <PlatformIcon :platform="model.platform as GroupPlatform" size="sm" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <h3 class="truncate text-base font-bold tracking-wide text-gray-900 dark:text-white">{{ model.name }}</h3>
                    <button type="button" class="text-gray-400 hover:text-primary-600" title="复制模型名" @click="copyModel(model.name)">⧉</button>
                  </div>
                  <div class="mt-4 flex flex-wrap items-center gap-1.5 text-[11px]">
                    <span class="rounded-md px-1.5 py-0.5 font-medium text-gray-500 dark:text-gray-400">
                      {{ platformIcon(model.platform) }} {{ platformLabel(model.platform) }}
                    </span>
                    <span class="rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide bg-indigo-50 text-indigo-600 dark:bg-indigo-500/20 dark:text-indigo-300">{{ billingModeLabel(model.pricing?.billing_mode) }}</span>
                    <span v-if="bestMultiplier(model) !== null" class="rounded px-1.5 py-0.5 font-bold text-emerald-600 dark:text-emerald-300">
                      低至 ×{{ formatMultiplier(bestMultiplier(model)!) }}
                    </span>
                  </div>
                </div>
              </div>

              <p v-if="modelDescription(model.name)" class="mt-3 line-clamp-3 text-xs leading-5 text-gray-500 dark:text-gray-400">
                {{ modelDescription(model.name) }}
              </p>

              <div v-if="model.pricing" class="mt-4 grid grid-cols-2 gap-x-4 gap-y-1 rounded-lg bg-gray-50 px-3 py-3 text-xs dark:bg-dark-900/70">
                <PriceBox label="输入" :value="scaledPrice(model.pricing.input_price, activeMultiplier(model))" />
                <PriceBox label="输出" :value="scaledPrice(model.pricing.output_price, activeMultiplier(model))" />
                <PriceBox v-if="hasPositivePrice(model.pricing.cache_write_price)" label="缓存写入" :value="scaledPrice(model.pricing.cache_write_price, activeMultiplier(model))" />
                <PriceBox v-if="hasPositivePrice(model.pricing.cache_read_price)" label="缓存读取" :value="scaledPrice(model.pricing.cache_read_price, activeMultiplier(model))" />
                <PriceBox v-if="hasPositivePrice(model.pricing.per_request_price)" label="每请求" :value="scaledPrice(model.pricing.per_request_price, activeMultiplier(model), false)" unit="/ request" />
                <PriceBox v-if="hasPositivePrice(model.pricing.image_output_price)" label="图片" :value="scaledPrice(model.pricing.image_output_price, activeMultiplier(model), false)" unit="/ image" />
              </div>
              <div v-else class="mt-4 rounded-md bg-gray-50 px-3 py-2 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">暂无定价</div>

              <div v-if="model.groups.length > 0" class="mt-4">
                <p class="mb-2 text-[11px] font-semibold tracking-wide text-gray-500 dark:text-gray-400">可用分组</p>
                <div class="flex flex-wrap gap-1.5">
                  <button
                    v-for="group in model.groups"
                    :key="group.groupName + group.rateMultiplier"
                    type="button"
                    class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-semibold transition-colors"
                    :class="selectedGroupKey[model.name] === groupKey(group)
                      ? groupActiveClass(model.platform)
                      : groupChipClass(model.platform)"
                    @click="selectGroup(model.name, group)"
                  >
                    <span>{{ group.groupName }}</span>
                    <span class="rounded bg-black/5 px-1 text-[11px] dark:bg-white/10">{{ formatMultiplier(group.rateMultiplier) }}x</span>
                  </button>
                </div>
              </div>
            </article>
          </div>

          <div v-else class="rounded-lg border border-dashed border-gray-300 py-12 text-center text-sm text-gray-400 dark:border-dark-600 dark:text-gray-500">
            没有匹配的模型
          </div>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, type PropType } from 'vue'
import { getModelSquare, type UserAvailableChannel, type UserSupportedModelPricing } from '@/api/channels'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'

interface ModelGroup {
  channelName: string
  groupName: string
  rateMultiplier: number
}

interface ModelEntry {
  name: string
  platform: string
  pricing: UserSupportedModelPricing | null
  groups: ModelGroup[]
}

interface FilterItem {
  key: string
  label: string
  count: number
  icon?: string
}

const FilterSection = defineComponent({
  props: {
    title: { type: String, required: true },
    items: { type: Array as PropType<FilterItem[]>, required: true },
    active: { type: Array as PropType<string[]>, required: true },
  },
  emits: ['toggle'],
  setup(props, { emit }) {
    return () => h('section', { class: 'rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900' }, [
      h('h3', { class: 'mb-2 text-xs font-semibold text-gray-500 dark:text-gray-400' }, props.title),
      h('div', { class: 'space-y-1.5' }, props.items.map(item => h('button', {
        type: 'button',
        class: [
          'flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-xs transition-colors',
          props.active.includes(item.key)
            ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
            : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-800'
        ],
        onClick: () => emit('toggle', item.key)
      }, [
        h('span', { class: 'min-w-0 truncate' }, `${item.icon ? item.icon + ' ' : ''}${item.label}`),
        h('span', { class: 'ml-2 rounded bg-gray-100 px-1.5 py-0.5 text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-400' }, String(item.count))
      ])))
    ])
  }
})

const PriceBox = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
    unit: { type: String, default: '/ 1M tokens' },
  },
  setup(props) {
    return () => h('div', { class: 'grid grid-cols-[auto_1fr] items-baseline gap-x-3' }, [
      h('span', { class: 'text-[12px] text-gray-500 dark:text-gray-400' }, props.label),
      h('span', { class: 'text-right text-sm font-bold text-gray-900 dark:text-gray-100' }, props.value),
      h('span', { class: 'col-span-2 text-right text-[10px] text-gray-400 dark:text-gray-500' }, props.unit),
    ])
  }
})

const loading = ref(true)
const search = ref('')
const channels = ref<UserAvailableChannel[]>([])
const activePlatforms = ref<string[]>([])
const activeGroups = ref<string[]>([])
const activeBillingModes = ref<string[]>([])
const selectedGroupKey = ref<Record<string, string>>({})

onMounted(async () => {
  try {
    channels.value = await getModelSquare()
  } catch (error) {
    console.error('Failed to load model square data:', error)
  } finally {
    loading.value = false
  }
})

const models = computed<ModelEntry[]>(() => {
  const map = new Map<string, ModelEntry>()
  for (const channel of channels.value) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        const key = `${section.platform}:${model.name}`.toLowerCase()
        if (!map.has(key)) {
          map.set(key, {
            name: model.name,
            platform: model.platform || section.platform,
            pricing: model.pricing,
            groups: [],
          })
        }
        const entry = map.get(key)!
        if (!entry.pricing && model.pricing) entry.pricing = model.pricing
        for (const group of section.groups) {
          entry.groups.push({
            channelName: channel.name,
            groupName: group.name,
            rateMultiplier: group.rate_multiplier,
          })
        }
      }
    }
  }

  return [...map.values()].map(model => {
    const seen = new Set<string>()
    return {
      ...model,
      groups: model.groups.filter(group => {
        const key = groupKey(group)
        if (seen.has(key)) return false
        seen.add(key)
        return true
      }).sort((a, b) => a.rateMultiplier - b.rateMultiplier || a.groupName.localeCompare(b.groupName))
    }
  })
})

const filteredModels = computed(() => {
  const query = search.value.trim().toLowerCase()
  return models.value
    .filter(model => activePlatforms.value.length === 0 || activePlatforms.value.includes(model.platform))
    .filter(model => activeGroups.value.length === 0 || model.groups.some(group => activeGroups.value.includes(group.groupName)))
    .filter(model => activeBillingModes.value.length === 0 || activeBillingModes.value.includes(model.pricing?.billing_mode || 'unknown'))
    .filter(model => {
      if (!query) return true
      return model.name.toLowerCase().includes(query) ||
        model.platform.toLowerCase().includes(query) ||
        platformLabel(model.platform).toLowerCase().includes(query) ||
        model.groups.some(group => group.groupName.toLowerCase().includes(query) || group.channelName.toLowerCase().includes(query))
    })
    .sort((a, b) => bestMultiplierValue(a) - bestMultiplierValue(b) || a.platform.localeCompare(b.platform) || a.name.localeCompare(b.name))
})

const platformFilters = computed<FilterItem[]>(() => {
  const counts = new Map<string, number>()
  for (const model of models.value) counts.set(model.platform, (counts.get(model.platform) || 0) + 1)
  return [...counts.entries()]
    .sort((a, b) => platformLabel(a[0]).localeCompare(platformLabel(b[0])))
    .map(([platform, count]) => ({ key: platform, label: platformLabel(platform), icon: platformIcon(platform), count }))
})

const groupFilters = computed<FilterItem[]>(() => {
  const counts = new Map<string, { count: number; multiplier: number }>()
  for (const model of models.value) {
    for (const group of model.groups) {
      const current = counts.get(group.groupName)
      counts.set(group.groupName, {
        count: (current?.count || 0) + 1,
        multiplier: current ? Math.min(current.multiplier, group.rateMultiplier) : group.rateMultiplier,
      })
    }
  }
  return [...counts.entries()]
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([groupName, info]) => ({ key: groupName, label: `${groupName} ×${formatMultiplier(info.multiplier)}`, count: info.count }))
})

const billingFilters = computed<FilterItem[]>(() => {
  const counts = new Map<string, number>()
  for (const model of models.value) {
    const mode = model.pricing?.billing_mode || 'unknown'
    counts.set(mode, (counts.get(mode) || 0) + 1)
  }
  return [...counts.entries()].map(([mode, count]) => ({ key: mode, label: billingModeLabel(mode), count }))
})

function toggleList(list: string[], key: string): string[] {
  return list.includes(key) ? list.filter(item => item !== key) : [...list, key]
}

function togglePlatformFilter(key: string) {
  activePlatforms.value = toggleList(activePlatforms.value, key)
}

function toggleGroupFilter(key: string) {
  activeGroups.value = toggleList(activeGroups.value, key)
}

function toggleBillingFilter(key: string) {
  activeBillingModes.value = toggleList(activeBillingModes.value, key)
}

function clearFilters() {
  search.value = ''
  activePlatforms.value = []
  activeGroups.value = []
  activeBillingModes.value = []
}

function groupKey(group: ModelGroup): string {
  return `${group.channelName}|${group.groupName}|${group.rateMultiplier}`
}

function selectGroup(modelName: string, group: ModelGroup) {
  const key = groupKey(group)
  selectedGroupKey.value[modelName] = selectedGroupKey.value[modelName] === key ? '' : key
}

function activeMultiplier(model: ModelEntry): number {
  const selected = model.groups.find(group => groupKey(group) === selectedGroupKey.value[model.name])
  return selected?.rateMultiplier ?? bestMultiplier(model) ?? 1
}

function bestMultiplier(model: ModelEntry): number | null {
  return model.groups.length > 0 ? Math.min(...model.groups.map(group => group.rateMultiplier)) : null
}

function bestMultiplierValue(model: ModelEntry): number {
  return bestMultiplier(model) ?? 9999
}

function platformLabel(platform: string): string {
  const normalized = platform.toLowerCase()
  if (normalized === 'openai') return 'OpenAI'
  if (normalized === 'anthropic') return 'Anthropic'
  if (normalized === 'gemini') return 'Gemini'
  if (normalized === 'zhipu') return 'ZHIPU'
  if (normalized === 'grok') return 'Grok'
  if (normalized === 'antigravity') return 'Antigravity'
  return platform
}

function platformIcon(platform: string): string {
  const normalized = platform.toLowerCase()
  if (normalized === 'openai') return '◍'
  if (normalized === 'zhipu') return '❄'
  if (normalized === 'anthropic') return '✳'
  if (normalized === 'gemini') return '✦'
  return '◆'
}

function groupChipClass(platform: string): string {
  const normalized = platform.toLowerCase()
  if (normalized === 'anthropic') return 'bg-amber-500/10 text-amber-700 hover:bg-amber-500/20 dark:bg-amber-500/15 dark:text-amber-300'
  if (normalized === 'openai') return 'bg-emerald-500/10 text-emerald-700 hover:bg-emerald-500/20 dark:bg-emerald-500/15 dark:text-emerald-300'
  if (normalized === 'gemini') return 'bg-cyan-500/10 text-cyan-700 hover:bg-cyan-500/20 dark:bg-cyan-500/15 dark:text-cyan-300'
  if (normalized === 'zhipu') return 'bg-blue-500/10 text-blue-700 hover:bg-blue-500/20 dark:bg-blue-500/15 dark:text-blue-300'
  return 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300'
}

function groupActiveClass(platform: string): string {
  const normalized = platform.toLowerCase()
  if (normalized === 'anthropic') return 'bg-amber-500/25 text-amber-800 ring-1 ring-amber-400 dark:text-amber-200'
  if (normalized === 'openai') return 'bg-emerald-500/25 text-emerald-800 ring-1 ring-emerald-400 dark:text-emerald-200'
  if (normalized === 'gemini') return 'bg-cyan-500/25 text-cyan-800 ring-1 ring-cyan-400 dark:text-cyan-200'
  if (normalized === 'zhipu') return 'bg-blue-500/25 text-blue-800 ring-1 ring-blue-400 dark:text-blue-200'
  return 'bg-primary-500/20 text-primary-700 ring-1 ring-primary-400 dark:text-primary-200'
}

function billingModeLabel(mode?: string | null): string {
  if (mode === 'per_request') return '按请求'
  if (mode === 'image') return '按图片'
  if (mode === 'unknown' || !mode) return '未知'
  return '按 TOKEN'
}

function hasPositivePrice(price: number | null | undefined): boolean {
  return typeof price === 'number' && price > 0
}

function scaledPrice(price: number | null | undefined, multiplier: number, perMillion = true): string {
  if (price === null || price === undefined) return '-'
  const value = price * multiplier * (perMillion ? 1_000_000 : 1)
  if (value === 0) return '$0'
  if (value >= 100) return `$${formatTrimmed(value, 0)}`
  if (value >= 1) return `$${formatTrimmed(value, 2)}`
  return `$${formatTrimmed(value, 4)}`
}

function formatTrimmed(value: number, digits: number): string {
  return value.toFixed(digits).replace(/\.0+$/, '').replace(/(\.\d*?)0+$/, '$1')
}

function formatMultiplier(value: number): string {
  return formatTrimmed(value, value < 1 ? 2 : 1)
}

function modelDescription(modelName: string): string {
  const normalized = modelName.toLowerCase()
  if (normalized.startsWith('glm-')) {
    return '智谱 GLM 系列国产大模型，中文理解、代码与推理表现优秀；本站经 Anthropic 兼容协议接入，可直接在 Claude Code 等客户端使用。'
  }
  return ''
}

async function copyModel(modelName: string) {
  try {
    await navigator.clipboard.writeText(modelName)
  } catch {
    // Clipboard may be blocked in non-secure contexts; ignore quietly.
  }
}
</script>
