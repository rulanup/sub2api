<template>
  <section
    ref="sectionRef"
    id="story"
    class="brand-story"
    :class="{ 'is-revealed': isRevealed }"
    data-act="story"
    aria-labelledby="brand-story-title"
  >
    <div class="landing-container brand-story__inner">
      <p class="brand-story__kicker">{{ t('home.landing.story.kicker') }}</p>
      <h2 id="brand-story-title" class="landing-visually-hidden">
        {{ t('home.landing.story.title') }}
      </h2>
      <div class="brand-story__lines">
        <p
          v-for="(line, index) in storyLines"
          :key="line"
          class="brand-story__line"
          :class="{ 'brand-story__line--accent': index === 2 }"
          :style="{ '--story-delay': index * 200 + 'ms' }"
        >
          {{ line }}
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const sectionRef = ref<HTMLElement | null>(null)
const isRevealed = ref(false)
let observer: IntersectionObserver | null = null

const storyLines = computed(() => [
  t('home.landing.story.lineOne'),
  t('home.landing.story.lineTwo'),
  t('home.landing.story.lineThree'),
])

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined' || !sectionRef.value) {
    isRevealed.value = true
    return
  }
  observer = new IntersectionObserver(
    (entries) => {
      if (!entries.some((entry) => entry.isIntersecting)) return
      isRevealed.value = true
      observer?.disconnect()
    },
    { threshold: 0.34, rootMargin: '0px 0px -12% 0px' },
  )
  observer.observe(sectionRef.value)
})

onBeforeUnmount(() => observer?.disconnect())
</script>
