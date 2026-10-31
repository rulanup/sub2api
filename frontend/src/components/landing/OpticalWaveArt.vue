<template>
  <section
    ref="storyRef"
    id="optical"
    class="optical-story"
    data-act="optical"
    aria-labelledby="optical-title"
  >
    <div ref="stageRef" class="optical-stage">
      <div ref="ambientRef" class="optical-ambient" aria-hidden="true"></div>

      <div ref="artRef" class="optical-art" aria-hidden="true">
        <svg
          :viewBox="viewBox"
          preserveAspectRatio="xMidYMid meet"
          focusable="false"
          shape-rendering="geometricPrecision"
        >
          <defs>
            <linearGradient id="landing-optical-far" x1="0" :y1="canvasHeight * 0.1" :x2="canvasWidth" :y2="canvasHeight * 0.9" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="var(--landing-wave-3)" />
              <stop offset="0.55" stop-color="var(--landing-wave-4)" />
              <stop offset="1" stop-color="var(--landing-wave-3)" />
            </linearGradient>
            <linearGradient id="landing-optical-mid" x1="0" :y1="canvasHeight * 0.65" :x2="canvasWidth" :y2="canvasHeight * 0.2" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="var(--landing-wave-4)" />
              <stop offset="0.52" stop-color="var(--landing-wave-2)" />
              <stop offset="1" stop-color="var(--landing-wave-4)" />
            </linearGradient>
            <linearGradient id="landing-optical-front" x1="0" :y1="canvasHeight * 0.35" :x2="canvasWidth" :y2="canvasHeight * 0.7" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="var(--landing-wave-2)" />
              <stop offset="0.58" stop-color="var(--landing-wave-1)" />
              <stop offset="1" stop-color="var(--landing-wave-2)" />
            </linearGradient>
            <linearGradient id="landing-optical-accent" x1="0" :y1="coreY" :x2="canvasWidth" :y2="coreY" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="var(--landing-wave-2)" stop-opacity="0.2" />
              <stop offset="0.7" stop-color="var(--landing-wave-1)" />
              <stop offset="1" stop-color="var(--landing-wave-2)" stop-opacity="0.32" />
            </linearGradient>
            <linearGradient id="landing-optical-collapse" x1="0" :y1="coreY" :x2="canvasWidth" :y2="coreY" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="var(--landing-brand)" stop-opacity="0" />
              <stop offset="0.36" stop-color="var(--landing-brand)" stop-opacity="0.38" />
              <stop offset="0.78" stop-color="var(--landing-brand)" stop-opacity="0.82" />
              <stop offset="1" stop-color="var(--landing-brand)" stop-opacity="0" />
            </linearGradient>
            <radialGradient id="landing-optical-glow" :cx="coreX" :cy="coreY" :r="glowRadius" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="var(--landing-optical-glow)" stop-opacity="0.72" />
              <stop offset="0.46" stop-color="var(--landing-optical-glow)" stop-opacity="0.26" />
              <stop offset="1" stop-color="var(--landing-optical-glow)" stop-opacity="0" />
            </radialGradient>
            <radialGradient id="landing-optical-mask-gradient" cx="62%" cy="48%" r="68%">
              <stop offset="0" stop-color="#fff" stop-opacity="1" />
              <stop offset="0.68" stop-color="#fff" stop-opacity="0.9" />
              <stop offset="0.9" stop-color="#fff" stop-opacity="0.28" />
              <stop offset="1" stop-color="#fff" stop-opacity="0" />
            </radialGradient>
            <mask id="landing-optical-mask" maskUnits="userSpaceOnUse" x="0" y="0" :width="canvasWidth" :height="canvasHeight">
              <rect :width="canvasWidth" :height="canvasHeight" fill="url(#landing-optical-mask-gradient)" />
            </mask>
          </defs>

          <ellipse
            ref="glowRef"
            :cx="coreX"
            :cy="coreY"
            :rx="glowRadius"
            :ry="glowRadius * 0.72"
            fill="url(#landing-optical-glow)"
          />

          <g mask="url(#landing-optical-mask)">
            <g ref="backLayerRef" class="optical-wave-layer optical-wave-layer--back">
              <path
                v-for="curve in backCurves"
                :key="curve.id"
                :d="curve.path"
                fill="none"
                :stroke="curve.stroke"
                :stroke-width="curve.width"
                :opacity="curve.opacity"
                stroke-linecap="round"
                vector-effect="non-scaling-stroke"
              />
            </g>
            <g ref="midLayerRef" class="optical-wave-layer optical-wave-layer--mid">
              <path
                v-for="curve in midCurves"
                :key="curve.id"
                :d="curve.path"
                fill="none"
                :stroke="curve.stroke"
                :stroke-width="curve.width"
                :opacity="curve.opacity"
                stroke-linecap="round"
                vector-effect="non-scaling-stroke"
              />
            </g>
            <g ref="frontLayerRef" class="optical-wave-layer optical-wave-layer--front">
              <path
                v-for="curve in frontCurves"
                :key="curve.id"
                :d="curve.path"
                fill="none"
                :stroke="curve.stroke"
                :stroke-width="curve.width"
                :opacity="curve.opacity"
                stroke-linecap="round"
                vector-effect="non-scaling-stroke"
              />
            </g>
            <g ref="accentLayerRef" class="optical-wave-layer optical-wave-layer--accent">
              <path
                v-for="curve in accentCurves"
                :key="curve.id"
                :d="curve.path"
                fill="none"
                :stroke="curve.stroke"
                :stroke-width="curve.width"
                :opacity="curve.opacity"
                stroke-linecap="round"
                vector-effect="non-scaling-stroke"
              />
            </g>
          </g>

          <line
            ref="collapseLineRef"
            class="optical-collapse-line"
            x1="0"
            :y1="coreY"
            :x2="canvasWidth"
            :y2="coreY"
            stroke="url(#landing-optical-collapse)"
            stroke-width="0.9"
            vector-effect="non-scaling-stroke"
          />
        </svg>
      </div>

      <div ref="captionRef" class="landing-container optical-caption">
        <h2 id="optical-title">
          <span>{{ t('home.landing.optical.lineOne') }}</span>
          <span>{{ t('home.landing.optical.lineTwo') }}</span>
        </h2>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

type WaveLayer = 'back' | 'mid' | 'front' | 'accent'

type WaveSpec = {
  layer: WaveLayer
  count: number
  amplitude: number
  pull: number
  width: number
  opacity: number
  gradient: string
}

type WaveCurve = {
  id: string
  path: string
  stroke: string
  width: number
  opacity: number
  layer: WaveLayer
}

type Point = { x: number; y: number }

const compactSpecs: WaveSpec[] = [
  { layer: 'back', count: 13, amplitude: 38, pull: 0.16, width: 0.55, opacity: 0.2, gradient: 'far' },
  { layer: 'mid', count: 13, amplitude: 54, pull: 0.34, width: 0.78, opacity: 0.38, gradient: 'mid' },
  { layer: 'front', count: 5, amplitude: 68, pull: 0.5, width: 1.05, opacity: 0.56, gradient: 'front' },
  { layer: 'accent', count: 2, amplitude: 78, pull: 0.62, width: 1.12, opacity: 0.7, gradient: 'accent' },
]

const desktopSpecs: WaveSpec[] = [
  { layer: 'back', count: 18, amplitude: 42, pull: 0.16, width: 0.55, opacity: 0.19, gradient: 'far' },
  { layer: 'mid', count: 17, amplitude: 58, pull: 0.34, width: 0.8, opacity: 0.38, gradient: 'mid' },
  { layer: 'front', count: 8, amplitude: 72, pull: 0.5, width: 1.08, opacity: 0.58, gradient: 'front' },
  { layer: 'accent', count: 4, amplitude: 82, pull: 0.64, width: 1.18, opacity: 0.72, gradient: 'accent' },
]

const initialCompact = typeof window !== 'undefined' && window.innerWidth <= 767
const isCompact = ref(initialCompact)
const canvasWidth = ref(initialCompact ? 1000 : 1600)
const canvasHeight = ref(
  typeof window === 'undefined'
    ? 900
    : Math.round(canvasWidth.value * (window.innerHeight / Math.max(window.innerWidth, 1))),
)
const coreX = computed(() => canvasWidth.value * 0.78)
const coreY = computed(() => canvasHeight.value * 0.3)
const glowRadius = computed(() => canvasWidth.value * (isCompact.value ? 0.38 : 0.28))
const viewBox = computed(() => '0 0 ' + canvasWidth.value + ' ' + canvasHeight.value)

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function smoothstep(edge0: number, edge1: number, value: number) {
  const x = clamp((value - edge0) / Math.max(0.0001, edge1 - edge0), 0, 1)
  return x * x * (3 - 2 * x)
}

function smoothPath(points: Point[]): string {
  if (points.length < 2) return ''
  let path = 'M ' + points[0].x + ' ' + points[0].y
  for (let i = 0; i < points.length - 1; i += 1) {
    const p0 = points[Math.max(0, i - 1)]
    const p1 = points[i]
    const p2 = points[i + 1]
    const p3 = points[Math.min(points.length - 1, i + 2)]
    const cp1x = p1.x + (p2.x - p0.x) / 6
    const cp1y = p1.y + (p2.y - p0.y) / 6
    const cp2x = p2.x - (p3.x - p1.x) / 6
    const cp2y = p2.y - (p3.y - p1.y) / 6
    path += ' C ' + cp1x + ' ' + cp1y + ', ' + cp2x + ' ' + cp2y + ', ' + p2.x + ' ' + p2.y
  }
  return path
}

function createWaveCurves(width: number, height: number, compact: boolean): WaveCurve[] {
  const specs = compact ? compactSpecs : desktopSpecs
  const centerX = width * 0.78
  const centerY = height * 0.3
  const sampleCount = compact ? 15 : 18
  const leftOverflow = compact ? 90 : 220
  const rightOverflow = compact ? 120 : 280
  const span = width + leftOverflow + rightOverflow
  const xStep = span / (sampleCount - 1)
  const fieldWidth = compact ? width * 0.19 : width * 0.19

  return specs.flatMap((spec, specIndex) => {
    return Array.from({ length: spec.count }, (_, index) => {
      const t = spec.count === 1 ? 0.5 : index / (spec.count - 1)
      const baseY = height * (0.055 + t * 0.89) + (specIndex - 1.5) * height * 0.006
      const phase = specIndex * 1.31 + index * 0.78
      const direction = index % 2 === 0 ? 1 : -1
      const points: Point[] = []

      for (let pointIndex = 0; pointIndex < sampleCount; pointIndex += 1) {
        const x = -leftOverflow + pointIndex * xStep
        const field = Math.exp(-Math.pow((x - centerX) / fieldWidth, 2))
        const travelWave =
          Math.sin(x / (compact ? width * 0.34 : width * 0.42) + phase) *
          spec.amplitude *
          0.24
        const localRipple =
          Math.sin((x - centerX) / (compact ? width * 0.12 : width * 0.13) + phase) *
          spec.amplitude *
          field *
          0.82
        const braid =
          Math.cos((x - centerX) / (compact ? width * 0.09 : width * 0.1) + phase * 0.7) *
          spec.amplitude *
          field *
          0.14 *
          direction
        const attraction = (centerY - baseY) * spec.pull * field
        const edgeDrift = Math.sin((pointIndex / (sampleCount - 1)) * Math.PI) * direction * 4
        points.push({
          x,
          y: baseY + travelWave + localRipple + braid + attraction + edgeDrift,
        })
      }

      return {
        id: spec.layer + '-' + specIndex + '-' + index,
        path: smoothPath(points),
        stroke: 'url(#landing-optical-' + spec.gradient + ')',
        width: spec.width,
        opacity: Math.min(0.84, spec.opacity + (index % 4) * 0.035),
        layer: spec.layer,
      }
    })
  })
}

const waveCurves = computed(() =>
  createWaveCurves(canvasWidth.value, canvasHeight.value, isCompact.value),
)
const backCurves = computed(() => waveCurves.value.filter((curve) => curve.layer === 'back'))
const midCurves = computed(() => waveCurves.value.filter((curve) => curve.layer === 'mid'))
const frontCurves = computed(() => waveCurves.value.filter((curve) => curve.layer === 'front'))
const accentCurves = computed(() => waveCurves.value.filter((curve) => curve.layer === 'accent'))

const storyRef = ref<HTMLElement | null>(null)
const stageRef = ref<HTMLElement | null>(null)
const ambientRef = ref<HTMLElement | null>(null)
const artRef = ref<HTMLElement | null>(null)
const captionRef = ref<HTMLElement | null>(null)
const glowRef = ref<SVGEllipseElement | null>(null)
const backLayerRef = ref<SVGGElement | null>(null)
const midLayerRef = ref<SVGGElement | null>(null)
const frontLayerRef = ref<SVGGElement | null>(null)
const accentLayerRef = ref<SVGGElement | null>(null)
const collapseLineRef = ref<SVGLineElement | null>(null)
const { t } = useI18n()
let frameId = 0
let motionQuery: MediaQueryList | null = null

function updateLayoutMetrics() {
  const stage = stageRef.value
  if (!stage) return
  const compact = window.innerWidth <= 767
  const nextWidth = compact ? 1000 : 1600
  const aspect = stage.clientHeight / Math.max(stage.clientWidth, 1)
  const nextHeight = Math.round(nextWidth * aspect)
  isCompact.value = compact
  canvasWidth.value = nextWidth
  canvasHeight.value = clamp(nextHeight, compact ? 1100 : 760, 2800)
}

function resetArtwork() {
  if (artRef.value) {
    artRef.value.style.transform = 'none'
    artRef.value.style.opacity = '1'
  }
  if (ambientRef.value) ambientRef.value.style.opacity = '0.82'
  if (captionRef.value) {
    captionRef.value.style.opacity = '1'
    captionRef.value.style.transform = 'none'
  }
  for (const layer of [backLayerRef.value, midLayerRef.value, frontLayerRef.value, accentLayerRef.value]) {
    if (!layer) continue
    layer.style.transform = 'none'
    layer.style.opacity = '1'
  }
  if (glowRef.value) glowRef.value.style.opacity = '0.75'
  if (collapseLineRef.value) collapseLineRef.value.style.opacity = '0'
}

function updateArtwork() {
  frameId = 0
  const story = storyRef.value
  const art = artRef.value
  const caption = captionRef.value
  const ambient = ambientRef.value
  if (!story || !art || !caption || !ambient) return

  if (motionQuery?.matches) {
    resetArtwork()
    return
  }

  const rect = story.getBoundingClientRect()
  const travel = Math.max(1, story.offsetHeight - window.innerHeight)
  const progress = clamp(-rect.top / travel, 0, 1)
  const expansion = smoothstep(0.02, 0.62, progress)
  const captionFade = 1 - smoothstep(0.1, 0.36, progress)
  const collapse = smoothstep(0.72, 0.98, progress)
  const returnFade = smoothstep(0.94, 1, progress)

  if (isCompact.value) {
    const translateY = (1 - expansion) * 30
    const scaleX = 0.93 + expansion * 0.07
    const scaleY = 0.84 + expansion * 0.16
    art.style.transform =
      'translate3d(' + ((1 - expansion) * 2) + '%, ' + translateY + '%, 0) scale(' +
      scaleX + ', ' + scaleY + ')'
  } else {
    const translateX = (1 - expansion) * 40
    const scaleX = 0.49 + expansion * 0.51
    const scaleY = 0.86 + expansion * 0.14
    art.style.transform =
      'translate3d(' + translateX + '%, 0, 0) scale(' + scaleX + ', ' + scaleY + ')'
  }

  art.style.opacity = String(1 - returnFade * 0.94)
  caption.style.opacity = String(captionFade)
  caption.style.transform = isCompact.value
    ? 'translate3d(0, ' + (-18 * (1 - captionFade)) + 'px, 0)'
    : 'translate3d(0, calc(-50% - ' + 22 * (1 - captionFade) + 'px), 0)'

  ambient.style.opacity = String(Math.pow(Math.sin(Math.PI * progress), 0.72))
  if (glowRef.value) glowRef.value.style.opacity = String(0.35 + Math.sin(Math.PI * progress) * 0.65)

  const drift = progress - 0.3
  if (backLayerRef.value) {
    backLayerRef.value.style.transform =
      'translate3d(' + drift * 9 + 'px, ' + -drift * 5 + 'px, 0)'
    backLayerRef.value.style.opacity = String(1 - collapse * 0.9)
  }
  if (midLayerRef.value) {
    midLayerRef.value.style.transform =
      'translate3d(' + drift * -14 + 'px, ' + drift * 4 + 'px, 0)'
    midLayerRef.value.style.opacity = String(1 - collapse * 0.94)
  }
  if (frontLayerRef.value) {
    frontLayerRef.value.style.transform =
      'translate3d(' + drift * 20 + 'px, ' + -drift * 7 + 'px, 0)'
    frontLayerRef.value.style.opacity = String(1 - collapse * 0.97)
  }
  if (accentLayerRef.value) {
    accentLayerRef.value.style.transform =
      'translate3d(' + drift * -22 + 'px, ' + drift * 5 + 'px, 0)'
    accentLayerRef.value.style.opacity = String(1 - collapse)
  }
  if (collapseLineRef.value) {
    const linePhase = smoothstep(0.72, 0.9, progress) * (1 - smoothstep(0.92, 1, progress))
    collapseLineRef.value.style.opacity = String(linePhase)
    collapseLineRef.value.style.transform = 'scaleX(' + smoothstep(0.72, 0.96, progress) + ')'
  }
}

function scheduleArtworkUpdate() {
  if (frameId) return
  if (typeof window.requestAnimationFrame !== 'function') {
    updateArtwork()
    return
  }
  frameId = window.requestAnimationFrame(updateArtwork)
}

function handleResize() {
  updateLayoutMetrics()
  scheduleArtworkUpdate()
}

function handleMotionPreferenceChange() {
  scheduleArtworkUpdate()
}

onMounted(() => {
  motionQuery =
    typeof window.matchMedia === 'function'
      ? window.matchMedia('(prefers-reduced-motion: reduce)')
      : null
  if (motionQuery && typeof motionQuery.addEventListener === 'function') {
    motionQuery.addEventListener('change', handleMotionPreferenceChange)
  } else if (motionQuery && typeof motionQuery.addListener === 'function') {
    motionQuery.addListener(handleMotionPreferenceChange)
  }
  updateLayoutMetrics()
  window.addEventListener('scroll', scheduleArtworkUpdate, { passive: true })
  window.addEventListener('resize', handleResize, { passive: true })
  scheduleArtworkUpdate()
})

onBeforeUnmount(() => {
  if (frameId) window.cancelAnimationFrame(frameId)
  window.removeEventListener('scroll', scheduleArtworkUpdate)
  window.removeEventListener('resize', handleResize)
  if (motionQuery && typeof motionQuery.removeEventListener === 'function') {
    motionQuery.removeEventListener('change', handleMotionPreferenceChange)
  } else if (motionQuery && typeof motionQuery.removeListener === 'function') {
    motionQuery.removeListener(handleMotionPreferenceChange)
  }
})
</script>
