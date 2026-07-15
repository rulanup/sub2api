import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ErrorPassthroughRulesModal from '../ErrorPassthroughRulesModal.vue'

const { createRule, listRules, showError } = vi.hoisted(() => ({
  createRule: vi.fn(),
  listRules: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    errorPassthrough: {
      list: listRules,
      create: createRule,
      update: vi.fn(),
      delete: vi.fn(),
      toggleEnabled: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess: vi.fn(),
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

function mountModal() {
  return mount(ErrorPassthroughRulesModal, {
    props: { show: false },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        Icon: true,
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
    listRules.mockReset().mockResolvedValue([])
    showError.mockReset()
    createRule.mockResolvedValue({})
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
