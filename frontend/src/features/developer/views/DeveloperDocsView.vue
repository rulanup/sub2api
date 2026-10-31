<template>
  <div class="dev-page dev-docs-page">
    <div class="dev-reading-progress" :aria-label="t('developer.docs.progress')">
      <span :style="{ width: readingProgress + '%' }"></span>
    </div>

    <div v-if="document" class="dev-docs-layout">
      <article ref="article" class="dev-docs-article">
        <nav class="dev-breadcrumbs" :aria-label="t('developer.docs.sidebar')">
          <RouterLink to="/developer/docs/getting-started">{{ t('developer.navigation.docs') }}</RouterLink>
          <span>/</span>
          <span>{{ categoryLabel }}</span>
          <span>/</span>
          <span aria-current="page">{{ brandText(document.title[locale], siteName) }}</span>
        </nav>

        <header class="dev-docs-header">
          <p class="dev-eyebrow">{{ t('developer.docs.eyebrow') }}</p>
          <h1 class="dev-page-title">{{ brandText(document.title[locale], siteName) }}</h1>
          <p class="dev-page-subtitle">{{ brandText(document.description[locale], siteName) }}</p>
          <div class="dev-docs-meta">
            <span>{{ t('developer.docs.updated') }} {{ document.updatedAt }}</span>
            <span>{{ document.version }}</span>
            <span>{{ document.translations[locale].status }}</span>
            <span v-if="document.translations.ja.status === 'missing'" class="dev-docs-meta__pending">
              JA — {{ t('developer.docs.jaMissing') }}
            </span>
          </div>
        </header>

        <div class="dev-markdown" v-html="renderedHtml" @click="handleArticleClick"></div>

        <section class="dev-doc-feedback" :aria-label="t('developer.docs.feedback')">
          <template v-if="feedback">
            <p>{{ t('developer.docs.feedbackThanks') }}</p>
          </template>
          <template v-else>
            <p>{{ t('developer.docs.feedback') }}</p>
            <div>
              <button type="button" class="dev-secondary-button" @click="saveFeedback('helpful')">
                {{ t('developer.docs.helpful') }}
              </button>
              <button type="button" class="dev-secondary-button" @click="saveFeedback('needs-work')">
                {{ t('developer.docs.notHelpful') }}
              </button>
            </div>
          </template>
        </section>

        <nav class="dev-doc-pagination" :aria-label="t('developer.navigation.docs')">
          <RouterLink
            v-if="previousDocument"
            class="dev-doc-pagination__item"
            :to="'/developer/docs/' + previousDocument.slug"
          >
            <small>← {{ t('developer.docs.previous') }}</small>
            <span>{{ brandText(previousDocument.title[locale], siteName) }}</span>
          </RouterLink>
          <span v-else></span>
          <RouterLink
            v-if="nextDocument"
            class="dev-doc-pagination__item dev-doc-pagination__item--next"
            :to="'/developer/docs/' + nextDocument.slug"
          >
            <small>{{ t('developer.docs.next') }} →</small>
            <span>{{ brandText(nextDocument.title[locale], siteName) }}</span>
          </RouterLink>
        </nav>
      </article>

      <aside class="dev-docs-aside">
        <nav v-if="toc.length" class="dev-toc" :aria-label="t('developer.docs.contents')">
          <p>{{ t('developer.docs.contents') }}</p>
          <a
            v-for="entry in toc"
            :key="entry.id"
            :href="'#' + entry.id"
            :class="{ 'is-active': activeHeading === entry.id, 'is-deep': entry.depth === 3 }"
            @click.prevent="scrollToHeading(entry.id)"
          >
            {{ entry.title }}
          </a>
        </nav>

        <section v-if="relatedDocuments.length" class="dev-related">
          <p>{{ t('developer.docs.related') }}</p>
          <RouterLink
            v-for="related in relatedDocuments"
            :key="related.id"
            :to="'/developer/docs/' + related.slug"
          >
            {{ brandText(related.title[locale], siteName) }}
          </RouterLink>
        </section>
      </aside>
    </div>

    <section v-else class="dev-empty-state">
      <h1 class="dev-page-title">{{ t('developer.docs.notFoundTitle') }}</h1>
      <p>{{ t('developer.docs.notFoundBody') }}</p>
      <RouterLink class="dev-primary-button" to="/developer/docs/getting-started">
        {{ t('developer.navigation.docs') }}
      </RouterLink>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

import { useAppStore } from '@/stores'
import { brandMarkdown, brandText } from '../branding'
import { copyText } from '../clipboard'
import { developerCategories, developerDocuments, getDeveloperDocument } from '../content'
import { useDeveloperLocale } from '../useDeveloperLocale'

const props = defineProps<{ slug?: string }>()
const { t } = useI18n()
const { locale } = useDeveloperLocale()
const appStore = useAppStore()
const article = ref<HTMLElement | null>(null)
const readingProgress = ref(0)
const activeHeading = ref('')
const feedback = ref('')
let scrollFrame = 0
let observer: IntersectionObserver | null = null
let feedbackTimer = 0

marked.setOptions({ breaks: true, gfm: true })

const document = computed(() => getDeveloperDocument(props.slug || 'getting-started'))
const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)
const documentIndex = computed(() =>
  document.value ? developerDocuments.findIndex((entry) => entry.id === document.value?.id) : -1,
)
const previousDocument = computed(() =>
  documentIndex.value > 0 ? developerDocuments[documentIndex.value - 1] : undefined,
)
const nextDocument = computed(() =>
  documentIndex.value >= 0 && documentIndex.value < developerDocuments.length - 1
    ? developerDocuments[documentIndex.value + 1]
    : undefined,
)
const categoryLabel = computed(() => {
  const category = developerCategories.find((entry) => entry.id === document.value?.category)
  return category ? category[locale.value] : ''
})
const relatedDocuments = computed(() =>
  (document.value?.relatedDocIds || [])
    .map((id) => developerDocuments.find((entry) => entry.id === id))
    .filter((entry): entry is NonNullable<typeof entry> => Boolean(entry)),
)

const renderedHtml = computed(() => {
  if (!document.value) return ''
  return DOMPurify.sanitize(
    marked.parse(brandMarkdown(document.value.content[locale.value], siteName.value)) as string,
  )
})

const toc = computed(() => {
  if (!document.value) return []
  const content = brandMarkdown(document.value.content[locale.value], siteName.value)
  return [...content.matchAll(/^(##|###)\s+(.+)$/gm)].map((match, index) => ({
    id: 'section-' + (index + 1),
    depth: match[1].length,
    title: match[2].replace(/[*_]/g, '').trim(),
  }))
})

function currentFeedbackKey(): string {
  return 'sub2api_developer_feedback_' + (document.value?.id || 'unknown') + '_' + locale.value
}

function loadFeedback(): void {
  feedback.value = localStorage.getItem(currentFeedbackKey()) || ''
}

function saveFeedback(value: string): void {
  feedback.value = value
  localStorage.setItem(currentFeedbackKey(), value)
  window.clearTimeout(feedbackTimer)
  feedbackTimer = window.setTimeout(() => {
    feedback.value = value
  }, 1)
}

function enhanceArticle(): void {
  const root = article.value
  if (!root) return

  root.querySelectorAll<HTMLElement>('h2, h3').forEach((heading, index) => {
    heading.id = 'section-' + (index + 1)
  })

  root.querySelectorAll<HTMLPreElement>('pre').forEach((pre) => {
    if (pre.querySelector('.dev-doc-copy')) return
    const button = window.document.createElement('button')
    button.type = 'button'
    button.className = 'dev-doc-copy'
    button.textContent = t('developer.common.copy')
    button.setAttribute('aria-label', t('developer.common.copy'))
    pre.appendChild(button)
  })

  observer?.disconnect()
  observer = new IntersectionObserver(
    (entries) => {
      const visible = entries
        .filter((entry) => entry.isIntersecting)
        .sort((left, right) => left.boundingClientRect.top - right.boundingClientRect.top)
      if (visible[0]?.target.id) activeHeading.value = visible[0].target.id
    },
    { rootMargin: '-90px 0px -65% 0px', threshold: [0, 1] },
  )
  root.querySelectorAll('h2, h3').forEach((heading) => observer?.observe(heading))
}

async function handleArticleClick(event: MouseEvent): Promise<void> {
  const target = event.target as HTMLElement
  const button = target.closest<HTMLButtonElement>('.dev-doc-copy')
  if (!button) return
  const code = button.parentElement?.querySelector('code')?.textContent || ''
  await copyText(code)
  button.textContent = t('developer.common.copied')
  window.setTimeout(() => {
    button.textContent = t('developer.common.copy')
  }, 1600)
}

function updateReadingProgress(): void {
  scrollFrame = 0
  const root = article.value
  if (!root) {
    readingProgress.value = 0
    return
  }
  const rect = root.getBoundingClientRect()
  const start = window.scrollY + rect.top - 100
  const length = Math.max(root.offsetHeight - window.innerHeight * 0.45, 1)
  const progress = ((window.scrollY - start) / length) * 100
  readingProgress.value = Math.max(0, Math.min(100, progress))
}

function scheduleReadingProgress(): void {
  if (scrollFrame) return
  scrollFrame = window.requestAnimationFrame(updateReadingProgress)
}

function scrollToHeading(id: string): void {
  const heading = window.document.getElementById(id)
  if (!heading) return
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  heading.scrollIntoView({ behavior: reduced ? 'auto' : 'smooth', block: 'start' })
  activeHeading.value = id
}

async function refreshDocument(): Promise<void> {
  activeHeading.value = ''
  loadFeedback()
  await nextTick()
  enhanceArticle()
  updateReadingProgress()
}

watch([() => props.slug, locale], refreshDocument)

onMounted(() => {
  window.addEventListener('scroll', scheduleReadingProgress, { passive: true })
  window.addEventListener('resize', scheduleReadingProgress, { passive: true })
  refreshDocument()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', scheduleReadingProgress)
  window.removeEventListener('resize', scheduleReadingProgress)
  if (scrollFrame) window.cancelAnimationFrame(scrollFrame)
  window.clearTimeout(feedbackTimer)
  observer?.disconnect()
})
</script>
