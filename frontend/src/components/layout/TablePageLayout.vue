<template>
  <div class="table-page-layout min-w-0" :class="{ 'mobile-mode': isMobile }">
    <!-- 固定区域：操作按钮 -->
    <div v-if="$slots.actions" class="layout-section-fixed">
      <slot name="actions" />
    </div>

    <!-- 固定区域：搜索和过滤器 -->
    <div v-if="$slots.filters" class="layout-section-fixed">
      <slot name="filters" />
    </div>

    <!-- 滚动区域：表格 -->
    <div class="layout-section-scrollable">
      <div class="table-page-card">
        <div class="table-scroll-container">
          <slot name="table" />
        </div>

        <!-- 分页器与表格共用一个产品外壳，避免两者之间出现无意义留白。 -->
        <div v-if="$slots.pagination" class="table-pagination">
          <slot name="pagination" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const isMobile = ref(false)

const checkMobile = () => {
  isMobile.value = window.innerWidth < 1024
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
/* 桌面端：Flexbox 布局 */
.table-page-layout {
  @apply flex min-w-0 flex-col gap-4;
}

.layout-section-fixed {
  @apply flex-shrink-0;
}

.layout-section-scrollable {
  @apply flex min-h-0 flex-col;
}

/* 外壳随真实行数增长；超过可用视口范围后只滚动表体。 */
.table-page-card {
  @apply overflow-hidden rounded-[10px] border-[0.5px] border-gray-200 bg-white shadow-none dark:border-dark-700 dark:bg-dark-900;
}

.table-scroll-container {
  @apply min-w-0 overflow-hidden bg-white dark:bg-dark-900;
}

.table-pagination {
  @apply border-t-[0.5px] border-gray-200 dark:border-dark-700;
}

.table-scroll-container :deep(.table-wrapper) {
  @apply overflow-x-auto overflow-y-auto;
  max-height: clamp(20rem, calc(100dvh - 18rem), 45rem);
  scrollbar-gutter: stable;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content; /* 关键：确保表格宽度根据内容撑开，从而触发横向滚动 */
  display: table; /* 使用标准 table 布局以支持 sticky 列 */
}

.table-scroll-container :deep(thead) {
  @apply bg-gray-50 dark:bg-dark-800;
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply h-11 border-b-[0.5px] border-gray-200 px-4 py-0 text-left text-xs font-medium normal-case tracking-normal text-gray-500 dark:border-dark-700 dark:text-dark-300;
}

.table-scroll-container :deep(td) {
  @apply h-[52px] border-b-[0.5px] border-gray-100 px-4 py-3 text-sm text-gray-700 dark:border-dark-800 dark:text-gray-300;
}

/* 移动端：恢复正常滚动 */
.table-page-layout.mobile-mode {
  @apply h-auto;
}

.table-page-layout.mobile-mode .table-page-card {
  @apply overflow-visible rounded-none border-0 bg-transparent;
}

.table-page-layout.mobile-mode .table-scroll-container {
  @apply overflow-visible bg-transparent;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(table) {
  @apply flex-none;
  display: table;
  min-width: 100%;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(.table-wrapper) {
  max-height: none;
  overflow-x: auto;
  overflow-y: auto;
}

.table-page-layout.mobile-mode .table-pagination {
  @apply mt-3 overflow-hidden rounded-[10px] border-[0.5px] bg-white dark:bg-dark-900;
}
</style>
