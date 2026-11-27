<template>
  <AppLayout>
    <div class="canvas-embed">
      <p
        v-if="message"
        class="m-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200"
      >
        {{ message }}
      </p>
      <iframe
        v-if="canvasSrc"
        ref="frame"
        :src="canvasSrc"
        title="无限画布"
        class="canvas-embed__frame"
        allow="clipboard-read; clipboard-write"
        @load="handoff"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'
import AppLayout from '@/components/layout/AppLayout.vue'

/**
 * 画布经 sub2api 画布网关访问（独立端口/域名，整站反代到画布）：
 * 先用当前登录态换一张一次性票据，iframe 打开网关登录地址，网关验票后种下画布会话 cookie。
 * 网关未启用或签票失败时直接提示原因，不再回退到直连画布端口。
 */
const frame = ref<HTMLIFrameElement | null>(null)
const canvasSrc = ref('')
let canvasOrigin = ''
let handedOff = false
let lastHandoffAt = 0
let sending = false
/** 页面重新可见时最多每 30 秒重新下发一次身份。 */
const REFRESH_INTERVAL_MS = 30_000
const message = ref('')

function errorText(error: unknown, fallback: string) {
  const e = error as { response?: { data?: { message?: string } }; message?: string } | null
  return e?.response?.data?.message || e?.message || fallback
}

onMounted(async () => {
  document.addEventListener('visibilitychange', onVisible)
  try {
    const { data } = await apiClient.post('/canvas/session')
    canvasOrigin = data.origin
    canvasSrc.value = data.url
  } catch (error) {
    message.value = errorText(error, '画布暂不可用，请联系管理员确认画布网关已启用')
  }
})

onBeforeUnmount(() => document.removeEventListener('visibilitychange', onVisible))

/** iframe 加载完成后首次下发身份。 */
async function handoff() {
  if (handedOff) return
  if (await sendIdentity()) handedOff = true
}

/**
 * 页面重新可见时（如从后台标签页切回来）再下发一次身份：
 * 后台改了画布分组后，/canvas/key 会把 key 改绑到新分组，画布收到后重新拉模型列表。
 */
function onVisible() {
  if (document.visibilityState !== 'visible' || !handedOff || sending) return
  if (Date.now() - lastHandoffAt < REFRESH_INTERVAL_MS) return
  void sendIdentity()
}

/**
 * 把当前登录身份下发给画布：后端按 userID 幂等返回该用户的
 * infinite-canvas 专用 key，画布收到后写入自己的本地存储。
 */
async function sendIdentity() {
  if (!canvasOrigin) return false
  const target = frame.value?.contentWindow
  if (!target) return false
  sending = true
  try {
    const { data } = await apiClient.post('/canvas/key')
    target.postMessage(
      { type: 'sub2api.identity', userId: data.user_id, apiKey: data.api_key, apiBase: window.location.origin },
      canvasOrigin,
    )
    lastHandoffAt = Date.now()
    return true
  } catch (error) {
    // key 下发失败不阻塞画布，但要让用户知道原因（如无权使用画布分组、余额不足）。
    message.value = errorText(error, '画布 key 下发失败，模型可能无法使用')
    return false
  } finally {
    sending = false
  }
}
</script>

<style scoped>
/* 父级布局不保证有确定高度，这里按视口算，避免 iframe 塌成默认 150px。 */
.canvas-embed {
  display: flex;
  width: 100%;
  height: calc(100vh - 120px);
  min-height: 520px;
}

.canvas-embed__frame {
  height: 100%;
  width: 100%;
  border: 0;
  flex: 1;
}
</style>
