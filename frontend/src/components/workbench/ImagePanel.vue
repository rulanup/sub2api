<template>
  <div class="relative flex h-full overflow-hidden bg-white dark:bg-dark-950">
    <div class="flex min-w-0 flex-1 flex-col">
      <div class="flex items-center justify-between gap-2 px-4 py-3 sm:px-6">
        <div>
          <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('workbench.imageGallery') }}</p>
          <p class="text-xs text-gray-400">{{ model }} · {{ selectedKeyLabel }}</p>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="rounded-full border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50 dark:border-dark-700 dark:text-gray-300 dark:hover:bg-dark-800" @click="showSettings = true">
            {{ t('workbench.imageSettings') }}
          </button>
          <button type="button" class="rounded-full px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800" :disabled="images.length === 0" @click="clearImages">
            {{ t('workbench.clearImages') }}
          </button>
        </div>
      </div>

      <div class="flex-1 overflow-y-auto px-4 pb-36 sm:px-6">
        <div v-if="images.length === 0 && !generating" class="mx-auto flex h-full max-w-3xl flex-col items-center justify-center text-center">
          <div class="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-gray-900 to-gray-700 text-2xl text-white shadow-lg dark:from-white dark:to-gray-200 dark:text-gray-900">
            ✦
          </div>
          <h2 class="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white sm:text-4xl">{{ t('workbench.imageWelcomeTitle') }}</h2>
          <p class="mt-3 max-w-xl text-sm text-gray-500 dark:text-gray-400 sm:text-base">{{ t('workbench.imageWelcome') }}</p>
        </div>

        <div v-else class="mx-auto grid max-w-5xl grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="(img, i) in images" :key="i" class="group overflow-hidden rounded-3xl border border-gray-100 bg-gray-50 shadow-sm dark:border-dark-800 dark:bg-dark-900">
            <div v-if="img.error" class="flex min-h-48 items-center justify-center p-4 text-sm text-red-600 dark:text-red-300">
              {{ img.error }}
            </div>
            <template v-else>
              <div class="relative aspect-square overflow-hidden bg-gray-100 dark:bg-dark-800">
                <img :src="img.url" :alt="img.prompt" class="h-full w-full object-cover transition duration-300 group-hover:scale-[1.02]" />
              </div>
              <div class="flex items-center justify-between gap-2 p-3">
                <p class="truncate text-xs text-gray-500 dark:text-gray-400">{{ img.prompt }}</p>
                <a :href="img.url" :download="`workbench-${i + 1}.png`" target="_blank" rel="noopener noreferrer" class="rounded-full bg-white px-2.5 py-1 text-[11px] text-gray-600 shadow-sm dark:bg-dark-800 dark:text-gray-300">
                  {{ t('workbench.download') }}
                </a>
              </div>
            </template>
          </div>
          <div v-if="generating" class="flex min-h-48 items-center justify-center rounded-3xl border border-dashed border-gray-300 dark:border-dark-700">
            <div class="flex items-center gap-3 text-sm text-gray-500">
              <svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
              {{ t('workbench.generating') }}
            </div>
          </div>
        </div>
      </div>

      <div class="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-white via-white/95 to-transparent px-3 pb-4 pt-16 dark:from-dark-950 dark:via-dark-950/95 sm:px-6">
        <div class="pointer-events-auto mx-auto max-w-3xl">
          <div class="rounded-[28px] border border-gray-200 bg-white p-2 shadow-[0_10px_40px_rgba(0,0,0,0.08)] dark:border-dark-700 dark:bg-dark-900">
            <textarea
              v-model="prompt"
              rows="1"
              class="max-h-40 w-full resize-none border-0 bg-transparent px-3 py-2.5 text-[15px] leading-6 text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-0 dark:text-white dark:placeholder-gray-500"
              style="field-sizing: content;"
              :placeholder="t('workbench.imagePlaceholder')"
              @keydown.enter.exact.prevent="generateImage"
            />
            <div class="flex items-center justify-between gap-2 px-1 pb-1">
              <button type="button" class="rounded-full bg-gray-100 px-3 py-1.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300" @click="showSettings = true">
                {{ size }} · {{ quality }}
              </button>
              <button
                type="button"
                class="inline-flex h-10 items-center justify-center rounded-full bg-gray-900 px-5 text-xs font-medium text-white disabled:opacity-40 dark:bg-white dark:text-gray-900"
                :disabled="!canGenerate"
                @click="generateImage"
              >
                {{ generating ? t('workbench.generating') : t('workbench.generate') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showSettings" class="fixed inset-0 z-50 flex items-end justify-center sm:items-center" @click.self="showSettings = false">
      <div class="absolute inset-0 bg-black/40" />
      <div class="relative w-full max-w-md rounded-t-3xl bg-white p-5 shadow-2xl dark:bg-dark-900 sm:rounded-3xl">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base font-semibold">{{ t('workbench.imageSettings') }}</h3>
          <button type="button" class="text-gray-400" @click="showSettings = false">✕</button>
        </div>
        <div class="space-y-4">
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500">{{ t('workbench.chatKey') }}</label>
            <select v-model="selectedKeyId" class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2.5 text-sm dark:border-dark-700 dark:bg-dark-800" :disabled="loadingKeys">
              <option :value="null" disabled>{{ loadingKeys ? t('workbench.loadingKeys') : t('workbench.selectKey') }}</option>
              <option v-for="k in apiKeys" :key="k.id" :value="k.id">{{ k.name || `sk-...${k.key.slice(-4)}` }}</option>
            </select>
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500">{{ t('workbench.imageModel') }}</label>
            <select v-model="model" class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2.5 text-sm dark:border-dark-700 dark:bg-dark-800">
              <option value="gpt-image-1">gpt-image-1</option>
              <option value="dall-e-3">dall-e-3</option>
              <option value="dall-e-2">dall-e-2</option>
            </select>
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500">{{ t('workbench.imageSize') }}</label>
            <select v-model="size" class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2.5 text-sm dark:border-dark-700 dark:bg-dark-800">
              <option value="1024x1024">1024x1024</option>
              <option value="1536x1024">1536x1024</option>
              <option value="1024x1536">1024x1536</option>
              <option value="auto">auto</option>
            </select>
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500">{{ t('workbench.imageQuality') }}</label>
            <select v-model="quality" class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2.5 text-sm dark:border-dark-700 dark:bg-dark-800">
              <option value="auto">auto</option>
              <option value="high">high</option>
              <option value="medium">medium</option>
              <option value="low">low</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ApiKey } from '@/types'
import { buildGatewayUrl } from '@/api/url'

const { t } = useI18n()

const props = defineProps<{
  apiKeys: ApiKey[]
  loadingKeys: boolean
}>()

interface GeneratedImage {
  url: string
  prompt: string
  error?: string
}

const prompt = ref('')
const images = ref<GeneratedImage[]>([])
const generating = ref(false)
const showSettings = ref(false)

const selectedKeyId = ref<number | null>(null)
const model = ref('gpt-image-1')
const size = ref('1024x1024')
const quality = ref('auto')

watch(() => props.apiKeys, (keys) => {
  if (keys.length > 0 && selectedKeyId.value === null) {
    selectedKeyId.value = keys[0].id
  }
}, { immediate: true })

const selectedKey = computed(() => props.apiKeys.find(k => k.id === selectedKeyId.value))
const selectedKeyLabel = computed(() => {
  const key = selectedKey.value
  if (!key) return t('workbench.selectKey')
  return key.name || `sk-...${key.key.slice(-4)}`
})
const canGenerate = computed(() => !generating.value && !!prompt.value.trim() && !!selectedKey.value)

function clearImages() {
  images.value = []
}

function normalizeImageUrl(item: { url?: string; b64_json?: string }) {
  if (item.url) return item.url
  if (item.b64_json) {
    if (item.b64_json.startsWith('data:')) return item.b64_json
    return `data:image/png;base64,${item.b64_json}`
  }
  return ''
}

async function generateImage() {
  if (!canGenerate.value) return
  const key = selectedKey.value!
  const p = prompt.value.trim()
  prompt.value = ''
  generating.value = true

  try {
    const resp = await fetch(buildGatewayUrl('/v1/images/generations'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${key.key}`,
      },
      body: JSON.stringify({
        model: model.value,
        prompt: p,
        n: 1,
        size: size.value,
        quality: quality.value,
      }),
    })

    if (!resp.ok) {
      let err = ''
      try {
        const j = await resp.json()
        err = j.error?.message || JSON.stringify(j)
      } catch {
        err = await resp.text()
      }
      images.value.unshift({ url: '', prompt: p, error: `Error ${resp.status}: ${err}` })
      return
    }

    const data = await resp.json()
    for (const item of data.data || []) {
      const url = normalizeImageUrl(item)
      if (url) images.value.unshift({ url, prompt: p })
    }
  } catch (e) {
    images.value.unshift({ url: '', prompt: p, error: `Error: ${e}` })
  } finally {
    generating.value = false
  }
}
</script>
