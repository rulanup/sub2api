<template>
  <div class="relative flex h-full overflow-hidden">
    <!-- Sidebar -->
    <aside
      class="hidden w-[280px] flex-shrink-0 flex-col border-r border-gray-200/80 bg-[#f7f7f8] dark:border-dark-800 dark:bg-[#111113] md:flex"
      :class="sidebarCollapsed ? '!w-0 !border-0 overflow-hidden' : ''"
    >
      <div class="flex items-center gap-2 p-3">
        <button
          type="button"
          class="flex flex-1 items-center justify-center gap-2 rounded-2xl bg-gray-900 px-3 py-2.5 text-sm font-medium text-white transition hover:bg-black dark:bg-white dark:text-gray-900 dark:hover:bg-gray-100"
          @click="createConversation"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
          {{ t('workbench.newConversation') }}
        </button>
      </div>
      <div class="flex-1 space-y-3 overflow-y-auto px-2 pb-3">
        <div v-if="activeConversations.length === 0 && archivedConversations.length === 0" class="px-3 py-8 text-center text-xs text-gray-400">
          {{ t('workbench.noConversations') }}
        </div>

        <div v-if="activeConversations.length > 0" class="space-y-1">
          <div
            v-for="conv in activeConversations"
            :key="conv.id"
            class="group flex w-full items-center gap-1 rounded-xl px-2 py-1.5 text-left text-sm transition"
            :class="conv.id === activeConvId
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-800 dark:text-white'
              : 'text-gray-600 hover:bg-white/70 dark:text-gray-300 dark:hover:bg-dark-800/70'"
          >
            <template v-if="editingTitleId === conv.id">
              <input
                :ref="(el) => setTitleInputRef(el)"
                v-model="editingTitle"
                class="min-w-0 flex-1 rounded-lg border border-gray-200 bg-white px-2 py-1 text-sm outline-none focus:border-gray-400 dark:border-dark-600 dark:bg-dark-900"
                @click.stop
                @keydown.enter.prevent="saveTitle(conv.id)"
                @keydown.esc.prevent="cancelEditTitle"
                @blur="saveTitle(conv.id)"
              />
            </template>
            <button
              v-else
              type="button"
              class="min-w-0 flex-1 truncate px-1 py-1 text-left"
              @click="switchConversation(conv.id)"
              @dblclick.stop="startEditTitle(conv)"
            >
              {{ conv.title }}
            </button>
            <div class="flex flex-shrink-0 items-center opacity-0 transition group-hover:opacity-100">
              <button type="button" class="rounded-md p-1 text-gray-400 hover:text-gray-700 dark:hover:text-gray-200" :title="t('workbench.renameChat')" @click.stop="startEditTitle(conv)">
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"><path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" /></svg>
              </button>
              <button type="button" class="rounded-md p-1 text-gray-400 hover:text-gray-700 dark:hover:text-gray-200" :title="t('workbench.archiveChat')" @click.stop="archiveConversation(conv.id)">
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25" /></svg>
              </button>
              <button type="button" class="rounded-md p-1 text-gray-400 hover:text-red-500" :title="t('workbench.deleteChat')" @click.stop="askDeleteConversation(conv.id)">
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
          </div>
        </div>

        <div v-if="archivedConversations.length > 0" class="space-y-1">
          <button type="button" class="flex w-full items-center justify-between px-2 py-1 text-[11px] font-medium uppercase tracking-wide text-gray-400" @click="showArchived = !showArchived">
            <span>{{ t('workbench.archived') }} ({{ archivedConversations.length }})</span>
            <span>{{ showArchived ? '▾' : '▸' }}</span>
          </button>
          <div v-if="showArchived" class="space-y-1">
            <div
              v-for="conv in archivedConversations"
              :key="conv.id"
              class="group flex w-full items-center gap-1 rounded-xl px-2 py-1.5 text-left text-sm text-gray-500 dark:text-gray-400"
              :class="conv.id === activeConvId ? 'bg-white shadow-sm dark:bg-dark-800' : 'hover:bg-white/70 dark:hover:bg-dark-800/70'"
            >
              <template v-if="editingTitleId === conv.id">
                <input
                  :ref="(el) => setTitleInputRef(el)"
                  v-model="editingTitle"
                  class="min-w-0 flex-1 rounded-lg border border-gray-200 bg-white px-2 py-1 text-sm outline-none focus:border-gray-400 dark:border-dark-600 dark:bg-dark-900"
                  @click.stop
                  @keydown.enter.prevent="saveTitle(conv.id)"
                  @keydown.esc.prevent="cancelEditTitle"
                  @blur="saveTitle(conv.id)"
                />
              </template>
              <button
                v-else
                type="button"
                class="min-w-0 flex-1 truncate px-1 py-1 text-left"
                @click="switchConversation(conv.id)"
                @dblclick.stop="startEditTitle(conv)"
              >
                {{ conv.title }}
              </button>
              <div class="flex flex-shrink-0 items-center opacity-0 transition group-hover:opacity-100">
                <button type="button" class="rounded-md p-1 text-gray-400 hover:text-gray-700 dark:hover:text-gray-200" :title="t('workbench.renameChat')" @click.stop="startEditTitle(conv)">
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"><path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" /></svg>
                </button>
                <button type="button" class="rounded-md p-1 text-gray-400 hover:text-gray-700 dark:hover:text-gray-200" :title="t('workbench.unarchiveChat')" @click.stop="unarchiveConversation(conv.id)">
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
                </button>
                <button type="button" class="rounded-md p-1 text-gray-400 hover:text-red-500" :title="t('workbench.deleteChat')" @click.stop="askDeleteConversation(conv.id)">
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <!-- Main chat -->
    <section class="relative flex min-w-0 flex-1 flex-col bg-white dark:bg-dark-950">
      <!-- Top bar -->
      <div class="absolute inset-x-0 top-0 z-10 flex items-center justify-between gap-2 px-3 py-3 sm:px-5">
        <div class="flex items-center gap-2">
          <button type="button" class="rounded-xl p-2 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800 md:hidden" @click="showMobileSidebar = true">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" /></svg>
          </button>
          <button type="button" class="hidden rounded-xl p-2 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800 md:inline-flex" @click="sidebarCollapsed = !sidebarCollapsed">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" /></svg>
          </button>
          <div class="hidden min-w-0 sm:block">
            <template v-if="activeConv && editingTitleId === activeConv.id">
              <input
                :ref="(el) => setTitleInputRef(el)"
                v-model="editingTitle"
                class="w-56 rounded-lg border border-gray-200 bg-white px-2 py-1 text-sm outline-none focus:border-gray-400 dark:border-dark-600 dark:bg-dark-900 dark:text-white"
                @keydown.enter.prevent="saveTitle(activeConv.id)"
                @keydown.esc.prevent="cancelEditTitle"
                @blur="saveTitle(activeConv.id)"
              />
            </template>
            <button
              v-else
              type="button"
              class="truncate text-left text-sm font-medium text-gray-800 hover:underline dark:text-gray-100"
              :title="t('workbench.renameChat')"
              @dblclick="activeConv && startEditTitle(activeConv)"
              @click="activeConv && startEditTitle(activeConv)"
            >
              {{ activeConv?.title || t('workbench.newConversation') }}
            </button>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-600 shadow-sm hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:hover:bg-dark-800" @click="showSettings = true">
            {{ model || t('workbench.selectModel') }}
          </button>
          <button
            v-if="activeConv && !activeConv.archived"
            type="button"
            class="rounded-full px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800"
            @click="archiveConversation(activeConv.id)"
          >
            {{ t('workbench.archiveChat') }}
          </button>
          <button
            v-else-if="activeConv?.archived"
            type="button"
            class="rounded-full px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800"
            @click="unarchiveConversation(activeConv.id)"
          >
            {{ t('workbench.unarchiveChat') }}
          </button>
          <button type="button" class="rounded-full px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800" @click="clearActiveChat">
            {{ t('workbench.clearChat') }}
          </button>
        </div>
      </div>

      <!-- Messages -->
      <div ref="messagesContainer" class="flex-1 overflow-y-auto px-3 pb-40 pt-16 sm:px-6">
        <div v-if="activeMessages.length === 0 && !streaming" class="mx-auto flex h-full max-w-3xl flex-col items-center justify-center text-center">
          <div class="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-gray-900 to-gray-700 text-2xl font-semibold text-white shadow-lg dark:from-white dark:to-gray-200 dark:text-gray-900">
            ✦
          </div>
          <h2 class="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white sm:text-4xl">
            {{ t('workbench.chatWelcomeTitle') }}
          </h2>
          <p class="mt-3 max-w-xl text-sm text-gray-500 dark:text-gray-400 sm:text-base">
            {{ t('workbench.chatWelcome') }}
          </p>
          <div class="mt-8 grid w-full max-w-2xl grid-cols-1 gap-3 sm:grid-cols-3">
            <button
              v-for="tip in chatTips"
              :key="tip"
              type="button"
              class="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-4 text-left text-sm text-gray-700 transition hover:border-gray-300 hover:bg-white hover:shadow-sm dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200 dark:hover:border-dark-600 dark:hover:bg-dark-800"
              @click="input = tip; focusInput()"
            >
              {{ tip }}
            </button>
          </div>
        </div>

        <div v-else class="mx-auto max-w-3xl space-y-8">
          <div v-for="(msg, i) in activeMessages" :key="i" class="space-y-2">
            <div v-if="msg.role === 'user'" class="flex justify-end">
              <div class="max-w-[85%] rounded-3xl bg-gray-100 px-4 py-3 text-[15px] leading-7 text-gray-900 dark:bg-dark-800 dark:text-gray-100">
                <div class="whitespace-pre-wrap break-words">{{ msg.content }}</div>
              </div>
            </div>
            <div v-else class="space-y-2">
              <div class="flex items-center gap-2 text-xs font-medium text-gray-400">
                <span class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-gray-900 text-[10px] text-white dark:bg-white dark:text-gray-900">AI</span>
                {{ t('workbench.assistant') }}
              </div>
              <div
                class="text-[15px] leading-7"
                :class="msg.content.startsWith('Error')
                  ? 'rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-red-700 dark:border-red-900/40 dark:bg-red-950/30 dark:text-red-300'
                  : 'text-gray-800 dark:text-gray-100'"
              >
                <div class="whitespace-pre-wrap break-words">{{ msg.content }}</div>
              </div>
            </div>
          </div>

          <div v-if="streaming" class="space-y-2">
            <div class="flex items-center gap-2 text-xs font-medium text-gray-400">
              <span class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-gray-900 text-[10px] text-white dark:bg-white dark:text-gray-900">AI</span>
              {{ t('workbench.assistant') }}
            </div>
            <div class="text-[15px] leading-7 text-gray-800 dark:text-gray-100">
              <div class="whitespace-pre-wrap break-words">{{ streamingContent }}<span class="ml-1 inline-block h-4 w-1.5 animate-pulse bg-gray-800 align-middle dark:bg-gray-100" /></div>
            </div>
          </div>
        </div>
      </div>

      <!-- Floating composer -->
      <div class="pointer-events-none absolute inset-x-0 bottom-0 z-20 bg-gradient-to-t from-white via-white/95 to-transparent px-3 pb-4 pt-16 dark:from-dark-950 dark:via-dark-950/95 sm:px-6">
        <div class="pointer-events-auto mx-auto max-w-3xl">
          <div class="rounded-[28px] border border-gray-200 bg-white p-2 shadow-[0_10px_40px_rgba(0,0,0,0.08)] dark:border-dark-700 dark:bg-dark-900 dark:shadow-[0_10px_40px_rgba(0,0,0,0.35)]">
            <textarea
              ref="inputRef"
              v-model="input"
              rows="1"
              class="max-h-40 w-full resize-none border-0 bg-transparent px-3 py-2.5 text-[15px] leading-6 text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-0 dark:text-white dark:placeholder-gray-500"
              style="field-sizing: content;"
              :placeholder="t('workbench.chatPlaceholder')"
              @keydown.enter.exact.prevent="sendMessage"
            />
            <div class="flex items-center justify-between gap-2 px-1 pb-1">
              <div class="flex min-w-0 items-center gap-2">
                <button type="button" class="max-w-[42%] truncate rounded-full bg-gray-100 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700" @click="showSettings = true">
                  {{ selectedKeyLabel }}
                </button>
                <button type="button" class="max-w-[42%] truncate rounded-full bg-gray-100 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700" @click="showSettings = true">
                  {{ model || t('workbench.selectModel') }}
                </button>
              </div>
              <div class="flex items-center gap-2">
                <button
                  v-if="streaming"
                  type="button"
                  class="rounded-full bg-gray-100 px-4 py-2 text-xs font-medium text-gray-700 hover:bg-gray-200 dark:bg-dark-800 dark:text-gray-200"
                  @click="stopStreaming"
                >
                  {{ t('workbench.stop') }}
                </button>
                <button
                  type="button"
                  class="inline-flex h-10 w-10 items-center justify-center rounded-full bg-gray-900 text-white transition hover:bg-black disabled:cursor-not-allowed disabled:opacity-40 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-100"
                  :disabled="!canSend"
                  @click="sendMessage"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 19.5l15-7.5-15-7.5v6l10 1.5-10 1.5v6z" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
          <p class="mt-2 text-center text-[11px] text-gray-400">{{ t('workbench.enterToSend') }}</p>
        </div>
      </div>
    </section>

    <!-- Mobile sidebar -->
    <div v-if="showMobileSidebar" class="fixed inset-0 z-40 flex md:hidden" @click.self="showMobileSidebar = false">
      <div class="absolute inset-0 bg-black/40" />
      <div class="relative m-0 flex h-full w-[82%] max-w-sm flex-col bg-[#f7f7f8] shadow-xl dark:bg-[#111113]">
        <div class="flex items-center justify-between px-4 py-3">
          <h3 class="text-sm font-semibold">{{ t('workbench.conversations') }}</h3>
          <button type="button" class="rounded-lg px-2 py-1 text-gray-500" @click="showMobileSidebar = false">✕</button>
        </div>
        <div class="px-3 pb-2">
          <button type="button" class="w-full rounded-2xl bg-gray-900 px-3 py-2.5 text-sm font-medium text-white dark:bg-white dark:text-gray-900" @click="createConversation(); showMobileSidebar = false">
            {{ t('workbench.newConversation') }}
          </button>
        </div>
        <div class="flex-1 space-y-3 overflow-y-auto px-2 pb-4">
          <button
            v-for="conv in activeConversations"
            :key="conv.id"
            type="button"
            class="flex w-full items-center justify-between rounded-xl px-3 py-2.5 text-left text-sm"
            :class="conv.id === activeConvId ? 'bg-white shadow-sm dark:bg-dark-800' : 'hover:bg-white/70 dark:hover:bg-dark-800/70'"
            @click="switchConversation(conv.id); showMobileSidebar = false"
          >
            <span class="truncate">{{ conv.title }}</span>
          </button>
          <div v-if="archivedConversations.length > 0" class="space-y-1">
            <div class="px-2 text-[11px] font-medium uppercase tracking-wide text-gray-400">{{ t('workbench.archived') }}</div>
            <button
              v-for="conv in archivedConversations"
              :key="conv.id"
              type="button"
              class="flex w-full items-center justify-between rounded-xl px-3 py-2.5 text-left text-sm text-gray-500"
              :class="conv.id === activeConvId ? 'bg-white shadow-sm dark:bg-dark-800' : 'hover:bg-white/70 dark:hover:bg-dark-800/70'"
              @click="switchConversation(conv.id); showMobileSidebar = false"
            >
              <span class="truncate">{{ conv.title }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Settings sheet -->
    <div v-if="showSettings" class="fixed inset-0 z-50 flex items-end justify-center sm:items-center" @click.self="showSettings = false">
      <div class="absolute inset-0 bg-black/40" />
      <div class="relative w-full max-w-md rounded-t-3xl bg-white p-5 shadow-2xl dark:bg-dark-900 sm:rounded-3xl">
        <div class="mx-auto mb-4 h-1.5 w-10 rounded-full bg-gray-200 dark:bg-dark-600 sm:hidden" />
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('workbench.chatSettings') }}</h3>
          <button type="button" class="rounded-lg px-2 py-1 text-gray-400 hover:text-gray-600" @click="showSettings = false">✕</button>
        </div>
        <div class="space-y-4">
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500">{{ t('workbench.chatKey') }}</label>
            <select v-model="selectedKeyId" class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2.5 text-sm dark:border-dark-700 dark:bg-dark-800" :disabled="loadingKeys">
              <option :value="null" disabled>{{ loadingKeys ? t('workbench.loadingKeys') : t('workbench.selectKey') }}</option>
              <option v-for="k in apiKeys" :key="k.id" :value="k.id">{{ k.name || `sk-...${k.key.slice(-4)}` }}</option>
            </select>
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500">{{ t('workbench.chatModel') }}</label>
            <div class="mb-2">
              <input v-model="modelSearch" type="text" class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2.5 text-sm dark:border-dark-700 dark:bg-dark-800" :placeholder="t('workbench.modelSearchPlaceholder')" />
            </div>
            <div class="max-h-56 overflow-y-auto rounded-2xl border border-gray-200 dark:border-dark-700">
              <div v-if="modelsLoading" class="px-3 py-6 text-center text-xs text-gray-400">{{ t('workbench.loadingModels') }}</div>
              <div v-else-if="groupedModels.length === 0" class="px-3 py-6 text-center text-xs text-gray-400">{{ t('workbench.noModels') }}</div>
              <template v-else>
                <div v-for="group in groupedModels" :key="group.platform">
                  <div class="sticky top-0 bg-gray-50 px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:bg-dark-800">{{ group.platform }}</div>
                  <button
                    v-for="m in group.models"
                    :key="m"
                    type="button"
                    class="flex w-full items-center px-3 py-2 text-left text-sm"
                    :class="m === model ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-dark-800'"
                    @click="selectModel(m)"
                  >
                    {{ m }}
                  </button>
                </div>
              </template>
            </div>
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500">{{ t('workbench.reasoningLevel') }}</label>
            <select v-model="reasoningEffort" class="w-full rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2.5 text-sm dark:border-dark-700 dark:bg-dark-800">
              <option value="auto">{{ t('workbench.reasoning.auto') }}</option>
              <option value="low">{{ t('workbench.reasoning.low') }}</option>
              <option value="medium">{{ t('workbench.reasoning.medium') }}</option>
              <option value="high">{{ t('workbench.reasoning.high') }}</option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <ConfirmDialog
      :show="!!pendingDeleteId"
      :title="t('workbench.deleteChatTitle')"
      :message="t('workbench.deleteChatMessage', { title: pendingDeleteTitle })"
      :confirm-text="t('common.delete')"
      danger
      @confirm="confirmDeleteConversation"
      @cancel="pendingDeleteId = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted, onBeforeUnmount, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ApiKey } from '@/types'
import { buildGatewayUrl } from '@/api/url'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const { t } = useI18n()

const props = defineProps<{
  apiKeys: ApiKey[]
  loadingKeys: boolean
}>()

interface ChatMessage { role: 'user' | 'assistant'; content: string }
interface Conversation {
  id: string
  title: string
  messages: ChatMessage[]
  model: string
  keyId: number | null
  createdAt: number
  archived?: boolean
  titleEdited?: boolean
}
interface ModelGroup { platform: string; models: string[] }

const STORAGE_KEY = 'sub2api_workbench_conversations'

const input = ref('')
const inputRef = ref<HTMLTextAreaElement | null>(null)
const titleInputRef = ref<HTMLInputElement | null>(null)
const streaming = ref(false)
const streamingContent = ref('')
const messagesContainer = ref<HTMLElement | null>(null)
const abortController = ref<AbortController | null>(null)
const showMobileSidebar = ref(false)
const showSettings = ref(false)
const sidebarCollapsed = ref(false)
const showArchived = ref(true)
const editingTitleId = ref<string | null>(null)
const editingTitle = ref('')
const pendingDeleteId = ref<string | null>(null)

const selectedKeyId = ref<number | null>(null)
const model = ref('gpt-4o-mini')
const reasoningEffort = ref('auto')
const modelSearch = ref('')
const modelGroups = ref<ModelGroup[]>([])
const modelsLoading = ref(false)

const conversations = ref<Conversation[]>([])
const activeConvId = ref('')

const chatTips = computed(() => [
  t('workbench.tipExplain'),
  t('workbench.tipCode'),
  t('workbench.tipTranslate'),
])

const selectedKey = computed(() => props.apiKeys.find(k => k.id === selectedKeyId.value))
const selectedKeyLabel = computed(() => {
  const key = selectedKey.value
  if (!key) return t('workbench.selectKey')
  return key.name || `sk-...${key.key.slice(-4)}`
})
const activeConv = computed(() => conversations.value.find(c => c.id === activeConvId.value))
const activeConversations = computed(() => conversations.value.filter(c => !c.archived))
const archivedConversations = computed(() => conversations.value.filter(c => !!c.archived))
const pendingDeleteTitle = computed(() => {
  const conv = conversations.value.find(c => c.id === pendingDeleteId.value)
  return conv?.title || t('workbench.newConversation')
})
const activeMessages = computed(() => activeConv.value?.messages || [])
const canSend = computed(() => !streaming.value && !!input.value.trim() && !!selectedKey.value && !!activeConv.value)

function setTitleInputRef(el: Element | ComponentPublicInstance | null) {
  titleInputRef.value = el instanceof HTMLInputElement ? el : null
}

const groupedModels = computed(() => {
  const q = modelSearch.value.toLowerCase().trim()
  if (!q) return modelGroups.value.map(g => ({ platform: g.platform, models: g.models.slice(0, 50) }))
  return modelGroups.value
    .map(g => ({ platform: g.platform, models: g.models.filter(m => m.toLowerCase().includes(q)) }))
    .filter(g => g.models.length > 0)
})

onMounted(() => {
  loadConversations()
})

onBeforeUnmount(() => {
  stopStreaming()
})

async function fetchModels(key: ApiKey) {
  modelsLoading.value = true
  try {
    const resp = await fetch(buildGatewayUrl('/v1/models'), {
      headers: { Authorization: `Bearer ${key.key}` },
    })
    if (!resp.ok) {
      modelGroups.value = []
      return
    }
    const data = await resp.json()
    const models: string[] = (data.data || []).map((m: { id: string }) => m.id).sort()
    const platformMap = new Map<string, string[]>()
    for (const m of models) {
      let platform = 'other'
      const lower = m.toLowerCase()
      if (lower.includes('claude')) platform = 'anthropic'
      else if (lower.includes('gpt') || lower.includes('o1') || lower.includes('o3') || lower.includes('o4')) platform = 'openai'
      else if (lower.includes('gemini')) platform = 'gemini'
      else if (lower.includes('grok')) platform = 'xai'
      else platform = m.split(/[-_/]/)[0] || 'other'
      if (!platformMap.has(platform)) platformMap.set(platform, [])
      platformMap.get(platform)!.push(m)
    }
    modelGroups.value = [...platformMap.entries()].map(([platform, ms]) => ({ platform, models: ms }))
  } catch (e) {
    console.warn('Failed to fetch models:', e)
    modelGroups.value = []
  } finally {
    modelsLoading.value = false
  }
}

watch(() => props.apiKeys, (keys) => {
  if (keys.length > 0 && selectedKeyId.value === null) {
    selectedKeyId.value = keys[0].id
    fetchModels(keys[0])
  }
}, { immediate: true })

watch(selectedKeyId, (keyId) => {
  const key = props.apiKeys.find(k => k.id === keyId)
  if (key) fetchModels(key)
  const conv = activeConv.value
  if (conv) {
    conv.keyId = keyId
    saveConversations()
  }
})

function selectModel(m: string) {
  model.value = m
  const conv = activeConv.value
  if (conv) {
    conv.model = m
    saveConversations()
  }
  showSettings.value = false
}

function focusInput() {
  nextTick(() => inputRef.value?.focus())
}

function genId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
}

function createConversation() {
  const conv: Conversation = {
    id: genId(),
    title: t('workbench.newConversation'),
    messages: [],
    model: model.value,
    keyId: selectedKeyId.value,
    createdAt: Date.now(),
    archived: false,
    titleEdited: false,
  }
  conversations.value.unshift(conv)
  activeConvId.value = conv.id
  saveConversations()
  focusInput()
}

function switchConversation(id: string) {
  activeConvId.value = id
  const conv = conversations.value.find(c => c.id === id)
  if (conv) {
    model.value = conv.model
    if (conv.keyId) selectedKeyId.value = conv.keyId
  }
}

function ensureActiveConversation() {
  if (conversations.value.some(c => c.id === activeConvId.value)) return
  const next = activeConversations.value[0] || archivedConversations.value[0]
  activeConvId.value = next?.id || ''
  if (!activeConvId.value) createConversation()
  else switchConversation(activeConvId.value)
}

function askDeleteConversation(id: string) {
  pendingDeleteId.value = id
}

function confirmDeleteConversation() {
  const id = pendingDeleteId.value
  pendingDeleteId.value = null
  if (!id) return
  const idx = conversations.value.findIndex(c => c.id === id)
  if (idx < 0) return
  conversations.value.splice(idx, 1)
  if (editingTitleId.value === id) cancelEditTitle()
  if (conversations.value.length === 0) {
    activeConvId.value = ''
    createConversation()
  } else {
    ensureActiveConversation()
  }
  saveConversations()
}

function archiveConversation(id: string) {
  const conv = conversations.value.find(c => c.id === id)
  if (!conv || conv.archived) return
  conv.archived = true
  if (activeConvId.value === id) {
    const next = activeConversations.value[0]
    if (next) switchConversation(next.id)
  }
  saveConversations()
}

function unarchiveConversation(id: string) {
  const conv = conversations.value.find(c => c.id === id)
  if (!conv || !conv.archived) return
  conv.archived = false
  conversations.value = [
    conv,
    ...conversations.value.filter(c => c.id !== id),
  ]
  switchConversation(id)
  saveConversations()
}

function startEditTitle(conv: Conversation) {
  editingTitleId.value = conv.id
  editingTitle.value = conv.title
  nextTick(() => {
    titleInputRef.value?.focus()
    titleInputRef.value?.select()
  })
}

function cancelEditTitle() {
  editingTitleId.value = null
  editingTitle.value = ''
}

function saveTitle(id: string) {
  if (editingTitleId.value !== id) return
  const conv = conversations.value.find(c => c.id === id)
  if (!conv) {
    cancelEditTitle()
    return
  }
  const next = editingTitle.value.trim()
  if (next) {
    conv.title = next.slice(0, 80)
    conv.titleEdited = true
    saveConversations()
  }
  cancelEditTitle()
}

function clearActiveChat() {
  const conv = activeConv.value
  if (conv) {
    conv.messages = []
    if (!conv.titleEdited) conv.title = t('workbench.newConversation')
    saveConversations()
  }
}

function saveConversations() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations.value))
  } catch {}
}

function loadConversations() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Conversation[]
      conversations.value = (parsed || []).map(item => ({
        ...item,
        archived: !!item.archived,
        titleEdited: !!item.titleEdited,
      }))
      const preferred = activeConversations.value[0] || conversations.value[0]
      if (preferred) {
        activeConvId.value = preferred.id
        model.value = preferred.model
        if (preferred.keyId) selectedKeyId.value = preferred.keyId
      }
    }
  } catch {}
  if (conversations.value.length === 0) createConversation()
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

function stopStreaming() {
  abortController.value?.abort()
  abortController.value = null
  streaming.value = false
}

async function sendMessage() {
  if (!canSend.value) return
  const key = selectedKey.value!
  const conv = activeConv.value!
  const userMsg = input.value.trim()
  input.value = ''

  conv.messages.push({ role: 'user', content: userMsg })
  if (!conv.titleEdited && conv.messages.filter(m => m.role === 'user').length === 1) {
    conv.title = userMsg.slice(0, 40) + (userMsg.length > 40 ? '...' : '')
  }
  conv.model = model.value
  conv.keyId = selectedKeyId.value
  saveConversations()
  scrollToBottom()

  streaming.value = true
  streamingContent.value = ''
  abortController.value = new AbortController()

  const body: Record<string, unknown> = {
    model: model.value,
    messages: conv.messages.map(m => ({ role: m.role, content: m.content })),
    stream: true,
  }
  if (reasoningEffort.value !== 'auto') body.reasoning_effort = reasoningEffort.value

  try {
    const resp = await fetch(buildGatewayUrl('/v1/chat/completions'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${key.key}`,
      },
      body: JSON.stringify(body),
      signal: abortController.value.signal,
    })
    if (!resp.ok) {
      let errMsg: string
      try {
        const j = await resp.json()
        errMsg = j.error?.message || JSON.stringify(j)
      } catch {
        errMsg = await resp.text()
      }
      conv.messages.push({ role: 'assistant', content: `Error ${resp.status}: ${errMsg}` })
      streaming.value = false
      saveConversations()
      return
    }

    const reader = resp.body?.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    while (reader) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''
      for (const line of lines) {
        const trimmed = line.trim()
        if (!trimmed || !trimmed.startsWith('data: ')) continue
        const data = trimmed.slice(6)
        if (data === '[DONE]') continue
        try {
          const parsed = JSON.parse(data)
          const delta = parsed.choices?.[0]?.delta?.content
          if (delta) {
            streamingContent.value += delta
            scrollToBottom()
          }
        } catch {}
      }
    }
    if (streamingContent.value) {
      conv.messages.push({ role: 'assistant', content: streamingContent.value })
    }
  } catch (e: any) {
    if (e?.name !== 'AbortError') {
      conv.messages.push({ role: 'assistant', content: `Error: ${e}` })
    } else if (streamingContent.value) {
      conv.messages.push({ role: 'assistant', content: streamingContent.value })
    }
  } finally {
    streaming.value = false
    streamingContent.value = ''
    abortController.value = null
    saveConversations()
    scrollToBottom()
  }
}
</script>
