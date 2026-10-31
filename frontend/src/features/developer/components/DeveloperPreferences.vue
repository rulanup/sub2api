<template>
  <div ref="root" class="dev-preferences">
    <button
      ref="trigger"
      type="button"
      class="dev-icon-button"
      :aria-label="t('developer.navigation.preferences')"
      :aria-expanded="open"
      aria-haspopup="dialog"
      @click="toggle"
    >
      <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
        <path d="M4 6h12M4 10h12M4 14h12" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
        <circle cx="8" cy="6" r="1.5" fill="var(--dev-surface)" stroke="currentColor" stroke-width="1.3" />
        <circle cx="13" cy="10" r="1.5" fill="var(--dev-surface)" stroke="currentColor" stroke-width="1.3" />
        <circle cx="7" cy="14" r="1.5" fill="var(--dev-surface)" stroke="currentColor" stroke-width="1.3" />
      </svg>
    </button>

    <div
      v-if="open"
      ref="panel"
      class="dev-popover"
      role="dialog"
      :aria-label="t('developer.navigation.preferences')"
      @keydown.esc.prevent.stop="close(true)"
    >
      <div class="dev-popover__section">
        <p class="dev-popover__label">{{ t('developer.common.language') }}</p>
        <div class="dev-segmented dev-segmented--vertical">
          <button
            v-for="option in languages"
            :key="option.id"
            type="button"
            class="dev-segmented__option"
            :class="{ 'is-active': locale === option.id }"
            :aria-pressed="locale === option.id"
            @click="chooseLocale(option.id)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>
      <div class="dev-popover__section">
        <p class="dev-popover__label">{{ t('developer.common.theme') }}</p>
        <div class="dev-segmented dev-segmented--vertical">
          <button
            v-for="option in themes"
            :key="option.id"
            type="button"
            class="dev-segmented__option"
            :class="{ 'is-active': mode === option.id }"
            :aria-pressed="mode === option.id"
            @click="chooseTheme(option.id)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import type { PortalLocale } from '../types'
import type { DeveloperThemeMode } from '../useDeveloperTheme'

defineProps<{
  locale: PortalLocale
  mode: DeveloperThemeMode
}>()

const emit = defineEmits<{
  'update:mode': [mode: DeveloperThemeMode]
  'change-locale': [locale: PortalLocale]
}>()

const { t } = useI18n()
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const open = ref(false)

const languages = computed(() => [
  { id: 'zh' as const, label: t('developer.common.chinese') },
  { id: 'en' as const, label: t('developer.common.english') },
])

const themes = computed(() => [
  { id: 'light' as const, label: t('developer.common.light') },
  { id: 'dark' as const, label: t('developer.common.dark') },
  { id: 'system' as const, label: t('developer.common.system') },
])

function onDocumentPointerDown(event: PointerEvent): void {
  if (!root.value?.contains(event.target as Node)) close(false)
}

function onDocumentKeyDown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && open.value) close(true)
}

function attachListeners(): void {
  document.addEventListener('pointerdown', onDocumentPointerDown)
  document.addEventListener('keydown', onDocumentKeyDown)
}

function detachListeners(): void {
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('keydown', onDocumentKeyDown)
}

async function toggle(): Promise<void> {
  open.value = !open.value
  if (open.value) {
    attachListeners()
    await nextTick()
    panel.value?.querySelector<HTMLButtonElement>('button')?.focus()
  }
}

function close(returnFocus: boolean): void {
  if (!open.value) return
  open.value = false
  detachListeners()
  if (returnFocus) trigger.value?.focus()
}

function chooseTheme(mode: DeveloperThemeMode): void {
  emit('update:mode', mode)
}

function chooseLocale(locale: PortalLocale): void {
  emit('change-locale', locale)
}

watch(open, (value) => {
  if (!value) detachListeners()
})

onBeforeUnmount(detachListeners)
</script>
