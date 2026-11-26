<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">无限画布</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        指定画布专用 key 绑定的分组，建议只选能出图的分组。用户打开画布时 key 会自动绑定或改绑到这里；不指定时沿用用户可用的第一个分组。
      </p>
    </div>
    <div class="p-6">
      <label for="canvas-group-id" class="text-sm font-medium text-gray-700 dark:text-gray-300">画布分组</label>
      <div class="mt-2 flex items-center gap-3">
        <select
          id="canvas-group-id"
          v-model.number="groupId"
          class="input w-full"
          :disabled="loading || saving"
          data-testid="canvas-group-id"
        >
          <option :value="0">不指定（自动选择）</option>
          <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}（{{ group.platform }}）</option>
        </select>
        <button
          type="button"
          class="btn btn-primary shrink-0"
          :disabled="loading || saving || groupId === saved"
          @click="save"
        >
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
      <p
        v-if="message"
        class="mt-1.5 text-xs"
        :class="failed ? 'text-red-600 dark:text-red-400' : 'text-gray-500 dark:text-gray-400'"
      >
        {{ message }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getAll } from '@/api/admin/groups'
import { getCanvasSettings, updateCanvasSettings } from '@/api/admin/canvas'

/** 画布分组设置：独立保存，不走系统设置页的整体表单。 */
type GroupOption = { id: number; name: string; platform: string }

const groups = ref<GroupOption[]>([])
const groupId = ref(0)
const saved = ref(0)
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const failed = ref(false)

function errorText(error: unknown, fallback: string) {
  const e = error as { response?: { data?: { message?: string } }; message?: string } | null
  return e?.response?.data?.message || e?.message || fallback
}

onMounted(async () => {
  try {
    const [list, settings] = await Promise.all([getAll(), getCanvasSettings()])
    groups.value = list.map((group) => ({ id: group.id, name: group.name, platform: String(group.platform) }))
    groupId.value = saved.value = settings.group_id
  } catch (error) {
    failed.value = true
    message.value = errorText(error, '画布设置加载失败')
  } finally {
    loading.value = false
  }
})

async function save() {
  saving.value = true
  failed.value = false
  message.value = ''
  try {
    const settings = await updateCanvasSettings(groupId.value)
    groupId.value = saved.value = settings.group_id
    message.value = '已保存。用户下次打开画布时，key 会自动改绑到该分组。'
  } catch (error) {
    failed.value = true
    message.value = errorText(error, '保存失败')
  } finally {
    saving.value = false
  }
}
</script>
