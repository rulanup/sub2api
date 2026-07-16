import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ErrorPassthroughRulesModal from '../ErrorPassthroughRulesModal.vue'

const { createRule, getWhitelist, listRules, showError, showSuccess, updateWhitelist } = vi.hoisted(() => ({
  createRule: vi.fn(),
  getWhitelist: vi.fn(),
  listRules: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  updateWhitelist: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    errorPassthrough: {
      list: listRules,
      create: createRule,
      update: vi.fn(),
      delete: vi.fn(),
      toggleEnabled: vi.fn(),
      getWhitelist,
      updateWhitelist,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const BaseDialogStub = defineComponent({
  props: {
    show: Boolean,
    title: String,
  },
  emits: ['close'],
  setup(props, { slots }) {
    return () =>
      props.show
        ? h('section', { 'data-dialog-title': props.title }, [slots.default?.(), slots.footer?.()])
        : null
  },
})

const SearchableUserAllowlistSelectorStub = defineComponent({
  name: 'SearchableUserAllowlistSelector',
  props: {
    modelValue: { type: Array, default: () => [] },
    placeholder: String,
    emptyLabel: String,
    deletedLabel: String,
    fallbackLabel: String,
    removeLabel: String,
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('div', {
      'data-testid': 'whitelist-selector',
      'data-user-ids': (props.modelValue as number[]).join(','),
      'data-placeholder': props.placeholder,
      'data-empty-label': props.emptyLabel,
      'data-deleted-label': props.deletedLabel,
      'data-fallback-label': props.fallbackLabel,
      'data-remove-label': props.removeLabel,
    }, [
      h('button', {
        'data-testid': 'emit-unnormalized-whitelist',
        onClick: () => emit('update:modelValue', [7, 2, 7, 0, -3, 1.5, 4]),
      }),
    ])
  },
})

function mountModal() {
  return mount(ErrorPassthroughRulesModal, {
    props: { show: false },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        Icon: true,
        SearchableUserAllowlistSelector: SearchableUserAllowlistSelectorStub,
      },
    },
  })
}

async function openCreateForm(wrapper: ReturnType<typeof mountModal>) {
  await wrapper.setProps({ show: true })
  await flushPromises()
  const createButton = wrapper
    .findAll('button')
    .find((button) => button.text().includes('admin.errorPassthrough.createRule'))
  expect(createButton).toBeDefined()
  await createButton!.trigger('click')
}

describe('ErrorPassthroughRulesModal', () => {
  beforeEach(() => {
    createRule.mockReset()
    getWhitelist.mockReset().mockResolvedValue({ user_ids: [] })
    listRules.mockReset().mockResolvedValue([])
    showError.mockReset()
    showSuccess.mockReset()
    updateWhitelist.mockReset().mockResolvedValue({ user_ids: [] })
    createRule.mockResolvedValue({})
  })

  it('loads rules and the global whitelist when opened', async () => {
    getWhitelist.mockResolvedValue({ user_ids: [8, 3] })
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(listRules).toHaveBeenCalledOnce()
    expect(getWhitelist).toHaveBeenCalledOnce()
    const selector = wrapper.get('[data-testid="whitelist-selector"]')
    expect(selector.attributes('data-user-ids')).toBe('8,3')
    expect(selector.attributes('data-placeholder')).toBe('admin.errorPassthrough.whitelist.searchPlaceholder')
    expect(selector.attributes('data-empty-label')).toBe('admin.errorPassthrough.whitelist.searchEmpty')
    expect(selector.attributes('data-deleted-label')).toBe('admin.errorPassthrough.whitelist.deleted')
    expect(selector.attributes('data-fallback-label')).toBe('admin.errorPassthrough.whitelist.fallback')
    expect(selector.attributes('data-remove-label')).toBe('admin.errorPassthrough.whitelist.remove')
  })

  it('normalizes whitelist IDs and adopts the canonical update response', async () => {
    getWhitelist.mockResolvedValue({ user_ids: [6] })
    updateWhitelist.mockResolvedValue({ user_ids: [2, 9] })
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.get('[data-testid="emit-unnormalized-whitelist"]').trigger('click')
    await wrapper.get('[data-testid="save-error-passthrough-whitelist"]').trigger('click')
    await flushPromises()

    expect(updateWhitelist).toHaveBeenCalledWith([2, 4, 7])
    expect(wrapper.get('[data-testid="whitelist-selector"]').attributes('data-user-ids')).toBe('2,9')
    expect(showSuccess).toHaveBeenCalledWith('admin.errorPassthrough.whitelist.saved')
  })

  it('reports whitelist load failure without interrupting rule loading', async () => {
    getWhitelist.mockRejectedValue(new Error('GET failed'))
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(listRules).toHaveBeenCalledOnce()
    expect(showError).toHaveBeenCalledWith('admin.errorPassthrough.whitelist.failedToLoad')
    expect(wrapper.find('[data-testid="whitelist-selector"]').exists()).toBe(true)
    const save = wrapper.get('[data-testid="save-error-passthrough-whitelist"]')
    expect(save.attributes('disabled')).toBeDefined()
    await save.trigger('click')
    expect(updateWhitelist).not.toHaveBeenCalled()
  })

  it('clears stale IDs and cannot save them after a failed reload', async () => {
    getWhitelist.mockResolvedValueOnce({ user_ids: [91] }).mockRejectedValueOnce(new Error('GET failed'))
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.get('[data-testid="whitelist-selector"]').attributes('data-user-ids')).toBe('91')

    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.get('[data-testid="whitelist-selector"]').attributes('data-user-ids')).toBe('')
    const save = wrapper.get('[data-testid="save-error-passthrough-whitelist"]')
    expect(save.attributes('disabled')).toBeDefined()
    await save.trigger('click')
    expect(updateWhitelist).not.toHaveBeenCalled()
  })

  it('ignores an older whitelist response after the dialog is reopened', async () => {
    let resolveFirst!: (value: { user_ids: number[] }) => void
    getWhitelist
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
      .mockResolvedValueOnce({ user_ids: [22] })
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.get('[data-testid="whitelist-selector"]').attributes('data-user-ids')).toBe('22')

    resolveFirst({ user_ids: [11] })
    await flushPromises()
    expect(wrapper.get('[data-testid="whitelist-selector"]').attributes('data-user-ids')).toBe('22')
  })

  it('waits for an in-flight whitelist save before loading after reopen', async () => {
    let resolveSave!: (value: { user_ids: number[] }) => void
    getWhitelist
      .mockResolvedValueOnce({ user_ids: [6] })
      .mockResolvedValueOnce({ user_ids: [2, 4, 7] })
    updateWhitelist.mockImplementationOnce(() => new Promise((resolve) => { resolveSave = resolve }))
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.get('[data-testid="emit-unnormalized-whitelist"]').trigger('click')
    await wrapper.get('[data-testid="save-error-passthrough-whitelist"]').trigger('click')
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    expect(getWhitelist).toHaveBeenCalledTimes(1)

    resolveSave({ user_ids: [2, 4, 7] })
    await flushPromises()

    expect(getWhitelist).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-testid="whitelist-selector"]').attributes('data-user-ids')).toBe('2,4,7')
  })

  it('reports whitelist update failure and re-enables saving', async () => {
    updateWhitelist.mockRejectedValue(new Error('PUT failed'))
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.get('[data-testid="emit-unnormalized-whitelist"]').trigger('click')
    await wrapper.get('[data-testid="save-error-passthrough-whitelist"]').trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.errorPassthrough.whitelist.failedToSave')
    expect(wrapper.get('[data-testid="save-error-passthrough-whitelist"]').attributes('disabled')).toBeUndefined()
  })

  it.each([
    {
      preset: 'authentication',
      name: 'admin.errorPassthrough.presets.authentication.name',
      errorCodes: [401],
      keywords: [],
    },
    {
      preset: 'rateLimit',
      name: 'admin.errorPassthrough.presets.rateLimit.name',
      errorCodes: [429],
      keywords: [],
    },
    {
      preset: 'securityPolicy',
      name: 'admin.errorPassthrough.presets.securityPolicy.name',
      errorCodes: [],
      keywords: ['cyber_policy'],
    },
  ])('submits the $preset preset payload after review', async ({ preset, name, errorCodes, keywords }) => {
    const wrapper = mountModal()
    await openCreateForm(wrapper)

    await wrapper.get(`[data-testid="error-preset-${preset}"]`).trigger('click')
    const message = wrapper.get<HTMLTextAreaElement>('[data-testid="custom-message"]')
    await message.setValue('Client-facing replacement')

    const submitButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.create'))
    await submitButton!.trigger('click')
    await flushPromises()

    expect(createRule).toHaveBeenCalledWith({
      name,
      enabled: true,
      priority: 0,
      error_codes: errorCodes,
      keywords,
      match_mode: 'any',
      platforms: [],
      passthrough_code: true,
      response_code: null,
      passthrough_body: false,
      custom_message: 'Client-facing replacement',
      skip_monitoring: false,
      description: null,
    })
  })

  it('rejects empty custom replacement text', async () => {
    const wrapper = mountModal()
    await openCreateForm(wrapper)
    await wrapper.get('[data-testid="error-preset-authentication"]').trigger('click')
    await wrapper.get('[data-testid="custom-message"]').setValue('   ')

    const submitButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.create'))
    await submitButton!.trigger('click')

    expect(createRule).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.errorPassthrough.customMessageRequired')
  })

  it('rejects malformed error status codes instead of silently dropping them', async () => {
    const wrapper = mountModal()
    await openCreateForm(wrapper)
    await wrapper.get('[data-testid="error-preset-rateLimit"]').trigger('click')
    const errorCodes = wrapper.get<HTMLInputElement>('input[placeholder="admin.errorPassthrough.form.errorCodesPlaceholder"]')
    await errorCodes.setValue('429, invalid')
    await wrapper.get('[data-testid="custom-message"]').setValue('Try later')

    const submitButton = wrapper.findAll('button').find((button) => button.text().includes('common.create'))
    await submitButton!.trigger('click')

    expect(createRule).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.errorPassthrough.invalidErrorCodes')
  })
})
