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
import { onMounted, ref } from 'vue'
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
const message = ref('')

function errorText(error: unknown, fallback: string) {
  const e = error as { response?: { data?: { message?: string } }; message?: string } | null
  return e?.response?.data?.message || e?.message || fallback
}

onMounted(async () => {
  try {
    const { data } = await apiClient.post('/canvas/session')
    canvasOrigin = data.origin
    canvasSrc.value = data.url
  } catch (error) {
    message.value = errorText(error, '画布暂不可用，请联系管理员确认画布网关已启用')
  }
})

/**
 * 把当前登录身份下发给画布：后端按 userID 幂等返回该用户的
 * infinite-canvas 专用 key，画布收到后写入自己的本地存储。
 */
async function handoff() {
  if (handedOff || !canvasOrigin) return
  const target = frame.value?.contentWindow
  if (!target) return
  try {
    const { data } = await apiClient.post('/canvas/key')
    target.postMessage(
      { type: 'sub2api.identity', userId: data.user_id, apiKey: data.api_key, apiBase: window.location.origin },
      canvasOrigin,
    )
    handedOff = true
  } catch (error) {
    // key 下发失败不阻塞画布，但要让用户知道原因（如无权使用画布分组、余额不足）。
    message.value = errorText(error, '画布 key 下发失败，模型可能无法使用')
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
