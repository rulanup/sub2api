<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">无限画布</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        指定画布专用 key 绑定的分组，可多选（例如生图分组 + 视频分组）。画布会合并拉取这些分组的模型，调用时按模型自动选分组。
        第一个勾选的是主分组，所有分组必须属于同一平台。用户打开画布时 key 会自动改绑；都不选时沿用用户可用的第一个分组。
      </p>
    </div>
    <div class="p-6">
      <div class="text-sm font-medium text-gray-700 dark:text-gray-300">画布分组</div>
      <div class="mt-2 space-y-2" data-testid="canvas-group-ids">
        <label
          v-for="group in groups"
          :key="group.id"
          class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"
          :class="{ 'opacity-50': isDisabled(group) }"
        >
          <input
            type="checkbox"
            :checked="selected.includes(group.id)"
            :disabled="loading || saving || isDisabled(group)"
            @change="toggle(group.id)"
          />
          <span>{{ group.name }}（{{ group.platform }}）</span>
          <span
            v-if="selected[0] === group.id"
            class="rounded bg-primary-50 px-1.5 text-xs text-primary-600 dark:bg-primary-900/30 dark:text-primary-300"
          >
            主分组
          </span>
        </label>
        <p v-if="!loading && !groups.length" class="text-sm text-gray-500 dark:text-gray-400">暂无分组</p>
      </div>
      <div class="mt-4 flex items-center gap-3">
        <button type="button" class="btn btn-primary" :disabled="loading || saving || !changed" @click="save">
          {{ saving ? '保存中…' : '保存' }}
        </button>
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ selected.length ? `已选 ${selected.length} 个分组` : '不指定（自动选择）' }}
        </span>
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
import { computed, onMounted, ref } from 'vue'
import { getAll } from '@/api/admin/groups'
import { getCanvasSettings, updateCanvasSettings } from '@/api/admin/canvas'

/** 画布分组设置：独立保存，不走系统设置页的整体表单。勾选顺序即绑定顺序，第一个为主分组。 */
type GroupOption = { id: number; name: string; platform: string }

const groups = ref<GroupOption[]>([])
const selected = ref<number[]>([])
const saved = ref<number[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const failed = ref(false)

// 一把 key 的分组必须同平台：选了第一个之后，其他平台的分组不可再选。
const platform = computed(() => groups.value.find((group) => group.id === selected.value[0])?.platform ?? '')
const changed = computed(() => selected.value.join(',') !== saved.value.join(','))

function isDisabled(group: GroupOption) {
  return !!platform.value && group.platform !== platform.value && !selected.value.includes(group.id)
}

function toggle(id: number) {
  selected.value = selected.value.includes(id) ? selected.value.filter((item) => item !== id) : [...selected.value, id]
}

function errorText(error: unknown, fallback: string) {
  const e = error as { response?: { data?: { message?: string } }; message?: string } | null
  return e?.response?.data?.message || e?.message || fallback
}

onMounted(async () => {
  try {
    const [list, settings] = await Promise.all([getAll(), getCanvasSettings()])
    groups.value = list.map((group) => ({ id: group.id, name: group.name, platform: String(group.platform) }))
    // 已删除或停用的分组不在列表里，无法取消勾选：加载时剔除，并提示重新保存。
    const known = new Set(groups.value.map((group) => group.id))
    const kept = settings.group_ids.filter((id) => known.has(id))
    selected.value = kept
    saved.value = [...settings.group_ids]
    if (kept.length !== settings.group_ids.length) {
      message.value = `已选的分组中有 ${settings.group_ids.length - kept.length} 个已被删除或停用，已自动移除，请确认后点保存。`
    }
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
    const settings = await updateCanvasSettings(selected.value)
    selected.value = [...settings.group_ids]
    saved.value = [...settings.group_ids]
    message.value = '已保存。用户下次打开画布时，key 会自动改绑到这些分组。'
  } catch (error) {
    failed.value = true
    message.value = errorText(error, '保存失败')
  } finally {
    saving.value = false
  }
}
</script>
