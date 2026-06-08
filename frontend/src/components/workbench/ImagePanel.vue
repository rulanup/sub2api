<template>
  <div class="flex h-full gap-4">
    <!-- Image Area -->
    <div class="flex min-w-0 flex-1 flex-col rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700">
      <!-- Generated Images -->
      <div class="flex-1 overflow-y-auto p-4">
        <div v-if="images.length === 0 && !generating" class="flex h-full items-center justify-center">
          <p class="text-sm text-gray-400 dark:text-gray-500">{{ t('workbench.imageWelcome') }}</p>
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div v-for="(img, i) in images" :key="i" class="group relative overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
            <img :src="img.url" :alt="img.prompt" class="w-full object-cover" />
            <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent p-3 opacity-0 transition-opacity group-hover:opacity-100">
              <p class="truncate text-xs text-white">{{ img.prompt }}</p>
            </div>
          </div>
        </div>
        <div v-if="generating" class="flex items-center justify-center py-8">
          <div class="flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400">
            <svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
            {{ t('workbench.generating') }}
          </div>
        </div>
      </div>

      <!-- Input -->
      <div class="border-t border-gray-200 p-4 dark:border-dark-700">
        <div class="flex gap-2">
          <textarea
            v-model="prompt"
            @keydown.enter.exact.prevent="generateImage"
            :placeholder="t('workbench.imagePlaceholder')"
            rows="1"
            class="flex-1 resize-none rounded-xl border-0 bg-gray-100 px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white dark:placeholder-gray-500"
            style="field-sizing: content; max-height: 120px;"
          />
          <button
            @click="generateImage"
            :disabled="!canGenerate"
            class="flex-shrink-0 rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ t('workbench.generate') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Settings Panel -->
    <div class="hidden w-72 flex-shrink-0 flex-col rounded-xl bg-white p-4 shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700 lg:flex">
      <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('workbench.imageSettings') }}</h3>

      <!-- API Key -->
      <div class="mb-3">
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('workbench.chatKey') }}</label>
        <select v-model="selectedKeyId" class="w-full rounded-lg border-0 bg-gray-100 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white">
          <option :value="null" disabled>{{ t('workbench.selectKey') }}</option>
          <option v-for="k in apiKeys" :key="k.id" :value="k.id">
            {{ k.name || ('sk-' + k.key.slice(-4)) }} ({{ k.key.slice(0, 10) }}...{{ k.key.slice(-4) }})
          </option>
        </select>
      </div>

      <!-- Model -->
      <div class="mb-3">
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('workbench.imageModel') }}</label>
        <select v-model="model" class="w-full rounded-lg border-0 bg-gray-100 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white">
          <option value="gpt-image-1">gpt-image-1</option>
          <option value="dall-e-3">dall-e-3</option>
          <option value="dall-e-2">dall-e-2</option>
        </select>
      </div>

      <!-- Size -->
      <div class="mb-3">
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('workbench.imageSize') }}</label>
        <select v-model="size" class="w-full rounded-lg border-0 bg-gray-100 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white">
          <option value="1024x1024">1024x1024</option>
          <option value="1536x1024">1536x1024</option>
          <option value="1024x1536">1024x1536</option>
          <option value="auto">auto</option>
        </select>
      </div>

      <!-- Quality -->
      <div class="mb-3">
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('workbench.imageQuality') }}</label>
        <select v-model="quality" class="w-full rounded-lg border-0 bg-gray-100 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:ring-blue-500 dark:bg-dark-800 dark:text-white">
          <option value="auto">auto</option>
          <option value="high">high</option>
          <option value="medium">medium</option>
          <option value="low">low</option>
        </select>
      </div>

      <!-- Actions -->
      <div class="mt-auto">
        <button @click="clearImages" class="w-full rounded-lg bg-gray-100 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700">
          {{ t('workbench.clearImages') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ApiKey } from '@/types'

const { t } = useI18n()

const props = defineProps<{
  apiKeys: ApiKey[]
  loadingKeys: boolean
}>()

interface GeneratedImage { url: string; prompt: string }

const prompt = ref('')
const images = ref<GeneratedImage[]>([])
const generating = ref(false)

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
const canGenerate = computed(() => !generating.value && prompt.value.trim() && selectedKey.value)

function clearImages() { images.value = [] }

async function generateImage() {
  if (!canGenerate.value) return
  const key = selectedKey.value!
  const p = prompt.value.trim()
  prompt.value = ''
  generating.value = true

  try {
    const resp = await fetch('/v1/images/generations', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${key.key}`,
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
      const err = await resp.text()
      images.value.unshift({ url: '', prompt: `Error: ${resp.status} - ${err}` })
      return
    }

    const data = await resp.json()
    for (const item of data.data || []) {
      images.value.unshift({ url: item.url || item.b64_json, prompt: p })
    }
  } catch (e) {
    images.value.unshift({ url: '', prompt: `Error: ${e}` })
  } finally {
    generating.value = false
  }
}
</script>
