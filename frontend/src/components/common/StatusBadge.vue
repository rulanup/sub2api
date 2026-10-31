<template>
  <span
    :class="[
      'inline-flex h-6 items-center gap-1.5 whitespace-nowrap rounded-md border-[0.5px] px-2 text-xs font-medium',
      variantClass
    ]"
  >
    <span class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true"></span>
    <span>{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  label: string
}>()

const variantClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
    case 'enabled':
      return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/70 dark:bg-emerald-950/40 dark:text-emerald-300'
    case 'disabled':
    case 'inactive':
    case 'warning':
    case 'pending':
      return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800/70 dark:bg-amber-950/40 dark:text-amber-300'
    case 'error':
    case 'danger':
    case 'failed':
      return 'border-red-200 bg-red-50 text-red-700 dark:border-red-800/70 dark:bg-red-950/40 dark:text-red-300'
    case 'info':
    case 'running':
      return 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800/70 dark:bg-blue-950/40 dark:text-blue-300'
    default:
      return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300'
  }
})
</script>
