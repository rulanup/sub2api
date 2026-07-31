<template>
  <div class="relative" ref="dropdownRef">
    <button
      @click="isOpen = !isOpen"
      class="sidebar-link mb-2 w-full"
      :class="{ 'sidebar-link-collapsed': collapsed }"
      :title="collapsed ? t('nav.themePicker') : undefined"
    >
      <Icon name="palette" class="h-5 w-5 flex-shrink-0" />
      <span
        v-if="!collapsed"
        class="sidebar-label"
        :class="{ 'sidebar-label-collapsed': collapsed }"
        :aria-hidden="collapsed ? 'true' : 'false'"
      >{{ t('nav.themePicker') }}</span>
    </button>

    <transition name="dropdown">
      <div
        v-if="isOpen"
        class="absolute bottom-0 left-14 z-50 w-72 rounded-xl border border-gray-200 bg-white p-4 shadow-xl dark:border-dark-700 dark:bg-dark-800"
        :class="collapsed ? '' : 'lg:left-0 lg:bottom-full lg:mb-2'"
      >
        <div class="mb-3 flex items-center justify-between">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('nav.themePickerTitle') }}
          </h4>
          <button
            v-if="hasCustomTheme"
            @click="resetTheme"
            class="text-xs text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400"
          >
            {{ t('nav.themeReset') }}
          </button>
        </div>

        <!-- Solid themes -->
        <p class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
          {{ t('nav.themeSolid') }}
        </p>
        <div class="mb-4 grid grid-cols-7 gap-2">
          <button
            v-for="preset in solidPresets"
            :key="preset.id"
            class="h-8 w-8 rounded-full ring-2 transition-transform hover:scale-110"
            :class="isActive(preset.id) ? 'ring-primary-500' : 'ring-transparent'"
            :style="{ backgroundColor: preset.swatch[0] }"
            :title="preset.label"
            @click="selectPreset(preset.id)"
          >
            <Icon
              v-if="isActive(preset.id)"
              name="check"
              size="sm"
              class="mx-auto text-white drop-shadow"
            />
          </button>
        </div>

        <!-- Gradient themes -->
        <p class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
          {{ t('nav.themeGradient') }}
        </p>
        <div class="mb-4 grid grid-cols-4 gap-2">
          <button
            v-for="preset in gradientPresets"
            :key="preset.id"
            class="h-8 w-8 rounded-full ring-2 transition-transform hover:scale-110"
            :class="isActive(preset.id) ? 'ring-primary-500' : 'ring-transparent'"
            :style="{ background: gradientStyle(preset) }"
            :title="preset.label"
            @click="selectPreset(preset.id)"
          >
            <Icon
              v-if="isActive(preset.id)"
              name="check"
              size="sm"
              class="mx-auto text-white drop-shadow"
            />
          </button>
        </div>

        <!-- Custom color -->
        <p class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
          {{ t('nav.themeCustom') }}
        </p>
        <div class="flex items-center gap-2">
          <label
            class="flex h-9 w-9 cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600"
            :style="{ backgroundColor: customHex || '#14b8a6' }"
          >
            <input
              v-model="colorInput"
              type="color"
              class="h-0 w-0 opacity-0"
              @input="applyCustom"
            />
          </label>
          <input
            v-model="customHex"
            type="text"
            maxlength="7"
            placeholder="#14b8a6"
            class="h-9 w-full rounded-lg border border-gray-200 bg-transparent px-3 text-sm text-gray-900 outline-none transition-colors focus:border-primary-500 dark:border-dark-600 dark:text-white"
            @keyup.enter="applyCustom"
          />
          <button
            @click="applyCustom"
            class="btn btn-primary h-9 !px-3 text-xs"
            :disabled="!isValidHexColor(customHex)"
          >
            {{ t('nav.themeApply') }}
          </button>
        </div>
        <p v-if="customError" class="mt-1 text-xs text-red-500">
          {{ customError }}
        </p>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  applyTheme,
  isValidHexColor,
  saveTheme,
  THEME_PRESETS,
  type ThemePreset,
} from '@/utils/theme'

defineProps<{ collapsed: boolean }>()

const { t } = useI18n()

const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const customHex = ref('')
const customError = ref('')
const colorInput = ref('')

const solidPresets = computed(() => THEME_PRESETS.filter((p) => p.kind === 'solid'))
const gradientPresets = computed(() => THEME_PRESETS.filter((p) => p.kind === 'gradient'))

const currentThemeId = computed(() => document.documentElement.dataset.theme ?? 'teal')
const hasCustomTheme = computed(() => currentThemeId.value === 'custom')

function isActive(id: string) {
  return currentThemeId.value === id
}

function gradientStyle(preset: ThemePreset) {
  const [from, to] = preset.swatch
  return to ? `linear-gradient(135deg, ${from}, ${to})` : from
}

function selectPreset(id: string) {
  applyTheme(id)
  saveTheme({ id })
  customError.value = ''
}

function resetTheme() {
  customHex.value = ''
  colorInput.value = ''
  selectPreset('teal')
}

function applyCustom() {
  const value = customHex.value.trim()
  if (!isValidHexColor(value)) {
    customError.value = t('nav.themeInvalidColor')
    return
  }
  customError.value = ''
  const normalized = value.startsWith('#') ? value : `#${value}`
  colorInput.value = normalized
  applyTheme('custom', normalized)
  saveTheme({ id: 'custom', custom: normalized })
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(4px) scale(0.97);
}
</style>
