<template>
  <div ref="rootRef" class="landing-preferences">
    <button
      ref="triggerRef"
      type="button"
      class="landing-preferences__trigger"
      :aria-expanded="isOpen"
      aria-controls="landing-preferences-panel"
      aria-haspopup="dialog"
      :aria-label="t('home.landing.nav.preferences')"
      @click="togglePopover"
    >
      <svg class="landing-preferences__icon" viewBox="0 0 20 20" fill="none" aria-hidden="true">
        <path d="M4 6h12M4 10h12M4 14h12" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
        <circle cx="7" cy="6" r="1.6" fill="var(--landing-surface)" stroke="currentColor" stroke-width="1.2" />
        <circle cx="13" cy="10" r="1.6" fill="var(--landing-surface)" stroke="currentColor" stroke-width="1.2" />
        <circle cx="9" cy="14" r="1.6" fill="var(--landing-surface)" stroke="currentColor" stroke-width="1.2" />
      </svg>
      <span class="landing-preferences__label">{{ t('home.landing.nav.preferences') }}</span>
    </button>

    <Transition name="landing-popover">
      <div
        v-if="isOpen"
        id="landing-preferences-panel"
        ref="panelRef"
        class="landing-preferences__panel"
        role="dialog"
        aria-modal="false"
        :aria-label="t('home.landing.preferences.title')"
      >
        <section class="landing-preferences__section">
          <p class="landing-preferences__heading">{{ t('home.landing.preferences.language') }}</p>
          <div class="landing-preferences__options landing-preferences__options--language">
            <button
              v-for="localeOption in availableLocales"
              :key="localeOption.code"
              type="button"
              class="landing-preferences__option"
              :class="{ 'is-active': localeOption.code === currentLocale }"
              :aria-pressed="localeOption.code === currentLocale"
              :disabled="switchingLocale"
              @click="selectLocale(localeOption.code)"
            >
              {{ localeOption.name }}
            </button>
          </div>
        </section>

        <section class="landing-preferences__section">
          <p class="landing-preferences__heading">{{ t('home.landing.preferences.appearance') }}</p>
          <div class="landing-preferences__options landing-preferences__options--theme">
            <button
              v-for="themeOption in themeOptions"
              :key="themeOption.value"
              type="button"
              class="landing-preferences__option"
              :class="{ 'is-active': themeOption.value === themeMode }"
              :aria-pressed="themeOption.value === themeMode"
              :data-testid="'theme-' + themeOption.value"
              @click="setThemeMode(themeOption.value)"
            >
              {{ themeOption.label }}
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { availableLocales, setLocale } from '@/i18n'

type ThemeMode = 'light' | 'dark' | 'system'

defineProps<{
  themeMode: ThemeMode
  setThemeMode: (mode: ThemeMode) => void
}>()

const { locale, t } = useI18n()
const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const isOpen = ref(false)
const switchingLocale = ref(false)
const currentLocale = computed(() => locale.value)

const themeOptions = computed<Array<{ value: ThemeMode; label: string }>>(() => [
  { value: 'light', label: t('home.landing.preferences.light') },
  { value: 'dark', label: t('home.landing.preferences.dark') },
  { value: 'system', label: t('home.landing.preferences.system') },
])

function togglePopover() {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    nextTick(() => {
      panelRef.value?.querySelector<HTMLButtonElement>('button:not(:disabled)')?.focus()
    })
  }
}

function closePopover(restoreFocus = false) {
  isOpen.value = false
  if (restoreFocus) {
    nextTick(() => triggerRef.value?.focus())
  }
}

async function selectLocale(code: string) {
  if (code === currentLocale.value || switchingLocale.value) return
  switchingLocale.value = true
  try {
    await setLocale(code)
  } finally {
    switchingLocale.value = false
  }
}

function handlePointerDown(event: PointerEvent) {
  if (!isOpen.value) return
  if (rootRef.value?.contains(event.target as Node)) return
  closePopover()
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && isOpen.value) {
    event.preventDefault()
    closePopover(true)
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', handlePointerDown)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handlePointerDown)
  document.removeEventListener('keydown', handleKeydown)
})
</script>
