<template>
  <div class="dev-code">
    <div class="dev-code__header">
      <span>{{ label || language }}</span>
      <button type="button" class="dev-code__copy" @click="copyCode">
        {{ copied ? t('developer.common.copied') : t('developer.common.copy') }}
      </button>
    </div>
    <pre tabindex="0"><code v-html="highlighted"></code></pre>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { copyText } from '../clipboard'

const props = defineProps<{
  code: string
  language: string
  label?: string
}>()

const { t } = useI18n()
const copied = ref(false)
let copiedTimer = 0

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

const highlighted = computed(() => {
  const pattern =
    /("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|\b(?:const|let|var|await|async|function|return|import|from|package|func|if|else|throw|new|true|false|null|defer|type|interface)\b|\b\d+(?:\.\d+)?\b|#[^\n]*|\/\/[^\n]*)/g
  let output = ''
  let cursor = 0
  for (const match of props.code.matchAll(pattern)) {
    const index = match.index ?? 0
    output += escapeHtml(props.code.slice(cursor, index))
    const token = match[0]
    const kind = /^["']/.test(token)
      ? 'string'
      : /^(\/\/|#)/.test(token)
        ? 'comment'
        : /^\d/.test(token)
          ? 'number'
          : 'keyword'
    output += '<span class="dev-code__token dev-code__token--' + kind + '">' + escapeHtml(token) + '</span>'
    cursor = index + token.length
  }
  output += escapeHtml(props.code.slice(cursor))
  return output
})

async function copyCode(): Promise<void> {
  await copyText(props.code)
  copied.value = true
  window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => {
    copied.value = false
  }, 1800)
}
</script>
