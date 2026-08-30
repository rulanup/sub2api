import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import type { DOMWrapper, VueWrapper } from '@vue/test-utils'

import RiskControlView from '../RiskControlView.vue'
import type { ContentModerationConfig, UpdateContentModerationConfig } from '@/api/admin/riskControl'

const {
  getConfig,
  updateConfig,
  getStatus,
  listLogs,
  getGroups,
  getProxies,
  testAPIKeys,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  getStatus: vi.fn(),
  listLogs: vi.fn(),
  getGroups: vi.fn(),
  getProxies: vi.fn(),
  testAPIKeys: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    riskControl: {
      getConfig,
      updateConfig,
      getStatus,
      listLogs,
      testAPIKeys,
      deleteFlaggedHash: vi.fn(),
      clearFlaggedHashes: vi.fn(),
      unbanUser: vi.fn(),
    },
    groups: {
      getAll: getGroups,
    },
    proxies: {
      getAll: getProxies,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_err: unknown, fallback: string) => fallback,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.riskControl.preBlockAPIKeyLoadSummary') {
          return `同步并发 ${params?.active} / 可用 Key ${params?.available}，累计 ${params?.total} 次，worker：${params?.workerActive} / ${params?.workerTotal}`
        }
        return key.replace(/\{(\w+)\}/g, (_, token) => String(params?.[token] ?? `{${token}}`))
      },
    }),
  }
})

const baseConfig = (): ContentModerationConfig => ({
  enabled: true,
  mode: 'pre_block',
  protocol: 'openai_moderation',
  controversial_action: 'allow',
  base_url: 'https://api.openai.com',
  model: 'omni-moderation-latest',
  aliyun_region_id: 'cn-shanghai',
  aliyun_service: 'query_security_check_pro',
  aliyun_access_key_configured: false,
  aliyun_access_key_id_masked: '',
  proxy_id: null,
  api_key_configured: false,
  api_key_masked: '',
  api_key_count: 0,
  api_key_masks: [],
  api_key_statuses: [],
  timeout_ms: 3000,
  sample_rate: 100,
  all_groups: true,
  group_ids: [],
  record_non_hits: false,
  worker_count: 4,
  queue_size: 32768,
  block_status: 403,
  block_message: '内容审计命中风险规则，请调整输入后重试',
  email_on_hit: true,
  auto_ban_enabled: true,
  ban_threshold: 10,
  violation_window_hours: 720,
  retry_count: 2,
  hit_retention_days: 180,
  non_hit_retention_days: 3,
  pre_hash_check_enabled: false,
  blocked_keywords: [],
  keyword_blocking_mode: 'keyword_and_api',
  thresholds: {
    harassment: 0.98,
    sexual: 0.65,
  },
  model_filter: {
    type: 'all',
    models: [],
  },
  cyber_policy_exclude_from_ban_count: false,
  sync_abuse_detection_enabled: true,
  sync_abuse_whitelist_user_ids: [3],
  sync_abuse_rpm_limit: 10,
  sync_abuse_concurrency: 1,
  sync_abuse_disable_user: true,
  cyber_usage_detection_enabled: true,
  cyber_usage_whitelist_user_ids: [4],
  cyber_usage_ban_threshold: 3,
  cyber_usage_window_hours: 24,
})

const runtimeStatus = () => ({
  enabled: true,
  risk_control_enabled: true,
  mode: 'pre_block',
  worker_count: 4,
  max_workers: 32,
  active_workers: 0,
  idle_workers: 4,
  queue_size: 32768,
  queue_length: 0,
  queue_usage_percent: 0,
  enqueued: 0,
  dropped: 0,
  processed: 0,
  errors: 0,
  pre_block_active: 0,
  pre_block_checked: 0,
  pre_block_allowed: 0,
  pre_block_blocked: 0,
  pre_block_errors: 0,
  pre_block_avg_latency_ms: 0,
  pre_block_api_key_active: 0,
  pre_block_api_key_available_count: 0,
  pre_block_api_key_total_calls: 0,
  pre_block_api_key_loads: [],
  api_key_statuses: [],
  flagged_hash_count: 0,
  last_cleanup_deleted_hit: 0,
  last_cleanup_deleted_non_hit: 0,
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const BaseDialogStub = defineComponent({
  props: {
    show: {
      type: Boolean,
      default: false,
    },
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})
const ModelWhitelistSelectorStub = defineComponent({
  props: {
    modelValue: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const onInput = (event: Event) => {
      const value = (event.target as HTMLInputElement).value
      emit(
        'update:modelValue',
        value
          .split(/[,\n]/)
          .map((item) => item.trim())
          .filter(Boolean)
      )
    }
    return () =>
      h('input', {
        'data-test': 'model-filter-input',
        value: (props.modelValue as string[]).join('\n'),
        onInput,
      })
  },
})
const SearchableUserAllowlistSelectorStub = defineComponent({
  inheritAttrs: false,
  props: {
    modelValue: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () => h('input', {
      'data-test': attrs['data-test'],
      value: (props.modelValue as number[]).join(','),
      onInput: (event: Event) => {
        emit('update:modelValue', (event.target as HTMLInputElement).value
          .split(',')
          .map(Number)
          .filter((id) => Number.isInteger(id) && id > 0))
      },
    })
  },
})

function mountRiskControlView() {
  return mount(RiskControlView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        BaseDialog: BaseDialogStub,
        Icon: true,
        Select: true,
        Toggle: true,
        Pagination: true,
        ModelWhitelistSelector: ModelWhitelistSelectorStub,
        ProxySelector: true,
      },
    },
  })
}

function findButtonByText(wrapper: VueWrapper, text: string): DOMWrapper<HTMLButtonElement> {
  const button = wrapper.findAll<HTMLButtonElement>('button').find((item) => item.text().includes(text))
  if (!button) {
    throw new Error(`button not found: ${text}`)
  }
  return button
}

describe('admin RiskControlView', () => {
  beforeEach(() => {
    getConfig.mockReset()
    updateConfig.mockReset()
    getStatus.mockReset()
    listLogs.mockReset()
    getGroups.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    getConfig.mockResolvedValue(baseConfig())
    getStatus.mockResolvedValue(runtimeStatus())
    listLogs.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 1 })
    getGroups.mockResolvedValue([])
    getProxies.mockResolvedValue([])
    testAPIKeys.mockReset()
    updateConfig.mockImplementation(async (payload: UpdateContentModerationConfig) => ({
      ...baseConfig(),
      ...payload,
      model_filter: payload.model_filter ?? baseConfig().model_filter,
      api_key_configured: false,
      api_key_masked: '',
      api_key_count: 0,
      api_key_masks: [],
      api_key_statuses: [],
      aliyun_access_key_configured: payload.clear_aliyun_credentials
        ? false
        : Boolean(payload.aliyun_access_key_id || baseConfig().aliyun_access_key_configured),
      aliyun_access_key_id_masked: payload.aliyun_access_key_id ? 'LTAI...test' : baseConfig().aliyun_access_key_id_masked,
    }))
  })

  it('shows Aliyun fields with defaults and hides model and the API key pool', async () => {
    const wrapper = mountRiskControlView()
    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')

    const vm = wrapper.vm as typeof wrapper.vm & {
      configForm: { protocol: string; base_url: string; aliyun_region_id: string; aliyun_service: string }
    }
    vm.configForm.protocol = 'aliyun_guardrails'
    await wrapper.vm.$nextTick()

    expect(vm.configForm.base_url).toBe('https://green-cip.cn-shanghai.aliyuncs.com')
    expect(vm.configForm.aliyun_region_id).toBe('cn-shanghai')
    expect(vm.configForm.aliyun_service).toBe('query_security_check_pro')
    expect(wrapper.find('[data-test="aliyun-credentials"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-key-pool"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="controversial-action"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="aliyun-text-only-notice"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="moderation-model"]').exists()).toBe(false)

    vm.configForm.aliyun_region_id = 'cn-beijing'
    await wrapper.vm.$nextTick()
    expect(vm.configForm.base_url).toBe('https://green-cip.cn-beijing.aliyuncs.com')

    vm.configForm.protocol = 'openai_moderation'
    await wrapper.vm.$nextTick()
    expect(vm.configForm.base_url).toBe('https://api.openai.com')
  })

  it('preserves custom endpoints while switching protocols', async () => {
    const wrapper = mountRiskControlView()
    await flushPromises()
    const vm = wrapper.vm as typeof wrapper.vm & { configForm: { protocol: string; base_url: string } }

    vm.configForm.base_url = 'https://moderation.example.com'
    vm.configForm.protocol = 'aliyun_guardrails'
    await wrapper.vm.$nextTick()
    expect(vm.configForm.base_url).toBe('https://moderation.example.com')

    vm.configForm.base_url = ''
    vm.configForm.protocol = 'openai_moderation'
    await wrapper.vm.$nextTick()
    expect(vm.configForm.base_url).toBe('https://api.openai.com')
  })

  it('does not echo saved Aliyun credentials and preserves them with empty inputs', async () => {
    getConfig.mockResolvedValue({
      ...baseConfig(),
      protocol: 'aliyun_guardrails',
      base_url: 'https://green-cip.cn-shanghai.aliyuncs.com',
      aliyun_access_key_configured: true,
      aliyun_access_key_id_masked: 'LTAI****7890',
    })
    const wrapper = mountRiskControlView()
    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')

    const idInput = wrapper.get<HTMLInputElement>('[data-test="aliyun-access-key-id"]')
    const secretInput = wrapper.get<HTMLInputElement>('[data-test="aliyun-access-key-secret"]')
    expect(idInput.element.value).toBe('')
    expect(idInput.attributes('placeholder')).toBe('LTAI****7890')
    expect(secretInput.element.type).toBe('password')
    expect(secretInput.element.value).toBe('')

    await findButtonByText(wrapper, 'admin.riskControl.saveConfig').trigger('click')
    await flushPromises()
    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      protocol: 'aliyun_guardrails',
      aliyun_access_key_id: '',
      aliyun_access_key_secret: '',
      clear_aliyun_credentials: false,
    }))
    expect(updateConfig.mock.calls[0]?.[0]).not.toHaveProperty('clear_api_key')
  })

  it('replaces Aliyun credentials and clears the Secret after saving', async () => {
    getConfig.mockResolvedValue({
      ...baseConfig(),
      protocol: 'aliyun_guardrails',
      aliyun_access_key_configured: true,
      aliyun_access_key_id_masked: 'LTAI****old',
    })
    const wrapper = mountRiskControlView()
    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    const secretInput = wrapper.get<HTMLInputElement>('[data-test="aliyun-access-key-secret"]')

    await wrapper.get('[data-test="aliyun-access-key-id"]').setValue('LTAI-replacement')
    await secretInput.setValue('replacement-secret')
    await findButtonByText(wrapper, 'admin.riskControl.saveConfig').trigger('click')

    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      aliyun_access_key_id: 'LTAI-replacement',
      aliyun_access_key_secret: 'replacement-secret',
      clear_aliyun_credentials: false,
    }))
    expect(secretInput.element.value).toBe('')
    await flushPromises()
  })

  it('marks configured Aliyun credentials for explicit clearing', async () => {
    getConfig.mockResolvedValue({
      ...baseConfig(),
      protocol: 'aliyun_guardrails',
      aliyun_access_key_configured: true,
      aliyun_access_key_id_masked: 'LTAI****7890',
    })
    const wrapper = mountRiskControlView()
    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    await findButtonByText(wrapper, 'admin.riskControl.clearAliyunCredentials').trigger('click')

    expect(wrapper.find('[data-test="aliyun-clear-pending"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="aliyun-credential-status"]').text()).toContain('admin.riskControl.aliyunCredentialsPendingClear')
    await findButtonByText(wrapper, 'admin.riskControl.saveConfig').trigger('click')
    await flushPromises()
    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      clear_aliyun_credentials: true,
      aliyun_access_key_id: '',
      aliyun_access_key_secret: '',
    }))
  })

  it('reuses saved Aliyun credentials when connection test inputs are empty', async () => {
    getConfig.mockResolvedValue({
      ...baseConfig(),
      protocol: 'aliyun_guardrails',
      base_url: 'https://green-cip.cn-shanghai.aliyuncs.com',
      aliyun_access_key_configured: true,
      aliyun_access_key_id_masked: 'LTAI****7890',
    })
    testAPIKeys.mockResolvedValueOnce({ items: [], image_count: 0 })
    const wrapper = mountRiskControlView()
    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    await wrapper.get('[data-test="test-aliyun-connection"]').trigger('click')

    expect(testAPIKeys).toHaveBeenCalledWith(expect.objectContaining({
      aliyun_access_key_id: '',
      aliyun_access_key_secret: '',
    }))
    await flushPromises()
  })

  it('tests Aliyun with the exact text-only payload and clears the Secret', async () => {
    getConfig.mockResolvedValue({
      ...baseConfig(),
      protocol: 'aliyun_guardrails',
      base_url: 'https://green-cip.cn-shanghai.aliyuncs.com',
      aliyun_access_key_configured: true,
      aliyun_access_key_id_masked: 'LTAI****7890',
    })
    testAPIKeys.mockResolvedValueOnce({ items: [], image_count: 0 })
    const wrapper = mountRiskControlView()
    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    const secretInput = wrapper.get<HTMLInputElement>('[data-test="aliyun-access-key-secret"]')
    await wrapper.get('[data-test="aliyun-access-key-id"]').setValue('LTAI-test')
    await secretInput.setValue('test-secret')
    await wrapper.get('[data-test="aliyun-test-prompt"]').setValue('test prompt')
    await wrapper.get('[data-test="test-aliyun-connection"]').trigger('click')

    expect(testAPIKeys).toHaveBeenCalledWith({
      protocol: 'aliyun_guardrails',
      controversial_action: 'allow',
      aliyun_region_id: 'cn-shanghai',
      aliyun_service: 'query_security_check_pro',
      aliyun_access_key_id: 'LTAI-test',
      aliyun_access_key_secret: 'test-secret',
      base_url: 'https://green-cip.cn-shanghai.aliyuncs.com',
      timeout_ms: 3000,
      proxy_id: 0,
      prompt: 'test prompt',
    })
    expect(secretInput.element.value).toBe('')
    await flushPromises()
  })

  it('submits Qwen3Guard protocol while using category thresholds', async () => {
    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    const vm = wrapper.vm as typeof wrapper.vm & {
      configForm: { protocol: string; model: string }
    }
    vm.configForm.protocol = 'qwen3guard_chat'
    vm.configForm.model = 'Qwen3Guard-Gen-8B'
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-test="controversial-action"]').exists()).toBe(false)
    await findButtonByText(wrapper, 'admin.riskControl.saveConfig').trigger('click')
    await flushPromises()

    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      protocol: 'qwen3guard_chat',
      model: 'Qwen3Guard-Gen-8B',
    }))
  })

  it('renders Qwen3Guard severity and categories from audit test results', async () => {
    testAPIKeys.mockResolvedValueOnce({
      items: [],
      image_count: 0,
      audit_result: {
        flagged: true,
        severity: 'unsafe',
        categories: ['jailbreak', 'illicit'],
        highest_category: 'jailbreak',
        highest_score: 1,
        composite_score: 1,
        category_scores: { jailbreak: 1 },
        thresholds: { jailbreak: 0.65 },
      },
    })
    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    const textareas = wrapper.findAll('textarea')
    await textareas[0].setValue('qwen-test-key')
    await findButtonByText(wrapper, 'admin.riskControl.testInputApiKeys').trigger('click')
    await flushPromises()

    expect(testAPIKeys).toHaveBeenCalledWith(expect.objectContaining({
      protocol: 'openai_moderation',
      controversial_action: 'allow',
    }))
    expect(wrapper.text()).toContain('jailbreak')
    expect(wrapper.text()).toContain('illicit')
  })

  it('shows a friendly message when the audit endpoint does not support the protocol', async () => {
    testAPIKeys.mockRejectedValueOnce({ response: { status: 400, data: { message: '暂不支持该接口' } } })
    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    await wrapper.findAll('textarea')[0].setValue('qwen-test-key')
    await findButtonByText(wrapper, 'admin.riskControl.testInputApiKeys').trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.riskControl.auditEndpointUnsupported')
  })

  it('saves the selected model filter mode and models', async () => {
    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()

    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    await findButtonByText(wrapper, 'admin.riskControl.tabs.scope').trigger('click')
    await findButtonByText(wrapper, 'admin.riskControl.modelFilterInclude').trigger('click')
    await wrapper.get('[data-test="model-filter-input"]').setValue('gpt-5.5, gpt-5.4')
    await findButtonByText(wrapper, 'admin.riskControl.saveConfig').trigger('click')
    await flushPromises()

    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      model_filter: {
        type: 'include',
        models: ['gpt-5.5', 'gpt-5.4'],
      },
    }))
    expect(showError).not.toHaveBeenCalled()
  })

  it('submits edited risk control thresholds when saving moderation config', async () => {
    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()

    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    await findButtonByText(wrapper, 'admin.riskControl.tabs.riskThresholds').trigger('click')
    for (const category of ['pii', 'unethical', 'jailbreak', 'copyright', 'political', 'qwen3guard']) {
      expect(wrapper.find(`[data-test="risk-threshold-${category}"]`).exists()).toBe(true)
    }
    await wrapper.get('[data-test="risk-threshold-sexual"]').setValue('72')
    await wrapper.get('[data-test="risk-threshold-harassment"]').setValue('99')
    await wrapper.get('[data-test="risk-threshold-political"]').setValue('80')
    await findButtonByText(wrapper, 'admin.riskControl.saveConfig').trigger('click')
    await flushPromises()

    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      thresholds: expect.objectContaining({
        sexual: 0.72,
        harassment: 0.99,
        political: 0.8,
        pii: 0.65,
        unethical: 0.65,
        jailbreak: 0.65,
        copyright: 0.65,
        qwen3guard: 0.65,
      }),
    }))
    expect(showError).not.toHaveBeenCalled()
  })

  it('submits abuse detection settings with numeric values and copied allowlists', async () => {
    const config = baseConfig()
    getConfig.mockResolvedValue(config)
    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          SearchableUserAllowlistSelector: SearchableUserAllowlistSelectorStub,
        },
      },
    })

    await flushPromises()
    await findButtonByText(wrapper, 'admin.riskControl.openSettings').trigger('click')
    await findButtonByText(wrapper, 'admin.riskControl.tabs.abuse').trigger('click')
    await wrapper.get('[data-test="sync-abuse-rpm"]').setValue('18')
    await wrapper.get('[data-test="sync-abuse-concurrency"]').setValue('4')
    await wrapper.get('[data-test="cyber-usage-threshold"]').setValue('6')
    await wrapper.get('[data-test="cyber-usage-window"]').setValue('72')
    await wrapper.get('[data-test="sync-abuse-allowlist"]').setValue('7,8')
    await wrapper.get('[data-test="cyber-usage-allowlist"]').setValue('9')
    await findButtonByText(wrapper, 'admin.riskControl.saveConfig').trigger('click')
    await flushPromises()

    const payload = updateConfig.mock.calls[0]?.[0]
    expect(payload).toEqual(expect.objectContaining({
      sync_abuse_detection_enabled: true,
      sync_abuse_whitelist_user_ids: [7, 8],
      sync_abuse_rpm_limit: 18,
      sync_abuse_concurrency: 4,
      sync_abuse_disable_user: true,
      cyber_usage_detection_enabled: true,
      cyber_usage_whitelist_user_ids: [9],
      cyber_usage_ban_threshold: 6,
      cyber_usage_window_hours: 72,
    }))
    expect(payload.sync_abuse_whitelist_user_ids).not.toBe(config.sync_abuse_whitelist_user_ids)
    expect(payload.cyber_usage_whitelist_user_ids).not.toBe(config.cyber_usage_whitelist_user_ids)
    expect(showError).not.toHaveBeenCalled()
  })

  it('describes worker runtime as async audit and pre-block record processing', async () => {
    getStatus.mockResolvedValue({
      ...runtimeStatus(),
      mode: 'observe',
      processed: 12,
      queue_length: 2,
    })

    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('admin.riskControl.workerStatusHint')
    expect(wrapper.text()).not.toContain('admin.riskControl.preBlockSyncStatus')
    expect(wrapper.text()).toContain('admin.riskControl.records')
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('2 / 32,768')
  })

  it('shows pre-block synchronous moderation metrics separately from worker queue', async () => {
    getStatus.mockResolvedValue({
      ...runtimeStatus(),
      pre_block_active: 2,
      pre_block_checked: 128,
      pre_block_allowed: 120,
      pre_block_blocked: 8,
      pre_block_errors: 1,
      pre_block_avg_latency_ms: 86,
      pre_block_api_key_active: 2,
      pre_block_api_key_available_count: 2,
      pre_block_api_key_total_calls: 128,
      active_workers: 3,
      worker_count: 7,
      pre_block_api_key_loads: [
        {
          index: 0,
          key_hash: 'hash-one',
          masked: 'sk-...one',
          status: 'ok',
          active: 1,
          total: 72,
          success: 70,
          errors: 2,
          avg_latency_ms: 84,
          last_latency_ms: 80,
          last_http_status: 200,
        },
        {
          index: 1,
          key_hash: 'hash-two',
          masked: 'sk-...two',
          status: 'ok',
          active: 1,
          total: 56,
          success: 56,
          errors: 0,
          avg_latency_ms: 90,
          last_latency_ms: 92,
          last_http_status: 200,
        },
      ],
    })

    const wrapper = mount(RiskControlView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          ModelWhitelistSelector: ModelWhitelistSelectorStub,
          ProxySelector: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('admin.riskControl.preBlockSyncStatus')
    expect(wrapper.text()).toContain('admin.riskControl.preBlockSyncHint')
    expect(wrapper.text()).not.toContain('admin.riskControl.workerStatus')
    expect(wrapper.text()).toContain('admin.riskControl.records')
    expect(wrapper.text()).toContain('128')
    expect(wrapper.text()).toContain('120')
    expect(wrapper.text()).toContain('8')
    expect(wrapper.text()).toContain('86 ms')
    expect(wrapper.text()).toContain('admin.riskControl.preBlockAPIKeyLoad')
    expect(wrapper.text()).toContain('sk-...one')
    expect(wrapper.text()).toContain('sk-...two')
    expect(wrapper.text()).toContain('72')
    expect(wrapper.text()).toContain('56')
    expect(wrapper.text()).toContain('同步并发 2 / 可用 Key 2，累计 128 次，worker：3 / 7')

    const runtimeCards = wrapper.get('[data-test="pre-block-runtime-cards"]')
    const syncCard = wrapper.get('[data-test="pre-block-sync-card"]')
    const apiKeyLoadCard = wrapper.get('[data-test="pre-block-api-key-load-card"]')

    expect(runtimeCards.classes()).toEqual(expect.arrayContaining([
      'grid',
      'grid-cols-1',
      'xl:grid-cols-[minmax(0,520px)_minmax(0,1fr)]',
    ]))
    expect(syncCard.element.parentElement).toBe(runtimeCards.element)
    expect(apiKeyLoadCard.element.parentElement).toBe(runtimeCards.element)
    expect(syncCard.classes()).toContain('card')
    expect(apiKeyLoadCard.classes()).toContain('card')
    expect(syncCard.get('h2').text()).toBe('admin.riskControl.preBlockSyncStatus')
    expect(syncCard.text()).toContain('admin.riskControl.preBlockSyncHint')
    expect(apiKeyLoadCard.get('h2').text()).toBe('admin.riskControl.preBlockAPIKeyLoad')
    expect(apiKeyLoadCard.text()).toContain('admin.riskControl.preBlockAPIKeyLoadHint')
    expect(wrapper.get('[data-test="pre-block-api-key-load-list"]').classes()).toEqual(expect.arrayContaining([
      'max-h-[280px]',
      'overflow-y-auto',
    ]))
  })
})
