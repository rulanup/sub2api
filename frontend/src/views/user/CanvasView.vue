<template>
  <AppLayout>
    <div class="canvas-embed">
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
 * 网关未启用时（接口报错）退回直连 3000 的单用户模式，上线前应去掉该回退。
 */
const frame = ref<HTMLIFrameElement | null>(null)
const canvasSrc = ref('')
let canvasOrigin = ''
let handedOff = false

onMounted(async () => {
  try {
    const { data } = await apiClient.post('/canvas/session')
    canvasOrigin = data.origin
    canvasSrc.value = data.url
  } catch {
    canvasOrigin = `${window.location.protocol}//${window.location.hostname}:3000`
    canvasSrc.value = `${canvasOrigin}/`
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
  } catch {
    // key 下发失败时画布退回自身渠道配置，这里不阻塞页面。
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
