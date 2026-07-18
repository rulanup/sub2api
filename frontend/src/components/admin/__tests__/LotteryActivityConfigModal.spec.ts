import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import LotteryActivityConfigModal from '../LotteryActivityConfigModal.vue'
import type { AdminActivityConfig } from '@/api/admin/activity'

const { getConfig, updateConfig, getAll } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  getAll: vi.fn(),
}))

vi.mock('@/api/admin/activity', () => ({ adminActivityAPI: { getConfig, updateConfig } }))
vi.mock('@/api/admin', () => ({ groupsAPI: { getAll } }))
vi.mock('vue-i18n', async importOriginal => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const validConfig = (): AdminActivityConfig => ({
  enabled: true,
  activity_id: 'summer-2026',
  title: 'Summer Draw',
  description: '',
  start_at: '2026-07-17T08:30:00Z',
  end_at: '2026-07-20T09:45:00Z',
  daily_draw_limit: 2,
  global_draw_limit: 1000,
  prizes: [
    { id: 'credit', type: 'balance', label: 'Credit', weight: 10, amount: 1.25 },
    { id: 'access', type: 'group', label: 'Access', weight: 2, group_id: 3, validity_days: 30 },
  ],
})

function mountModal() {
  return mount(LotteryActivityConfigModal, {
    props: { show: false },
    attachTo: document.body,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Icon: true,
        LoadingSpinner: true,
        Toggle: true,
      },
    },
  })
}

describe('LotteryActivityConfigModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getConfig.mockReset()
    updateConfig.mockReset()
    getAll.mockReset()
    getConfig.mockResolvedValue(validConfig())
    getAll.mockResolvedValue([
      { id: 3, name: 'Active', status: 'active', is_private: false, subscription_type: 'subscription' },
      { id: 4, name: 'Private', status: 'active', is_private: true, subscription_type: 'subscription' },
    ])
    updateConfig.mockImplementation(async config => config)
  })

  it('saves a normalized UTC payload through the dedicated activity API', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    await wrapper.get('[data-testid="activity-config-save"]').trigger('click')
    await flushPromises()
    expect(updateConfig).toHaveBeenCalledWith(validConfig())
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('rejects an invalid configuration before PUT', async () => {
    getConfig.mockResolvedValue({ ...validConfig(), activity_id: 'INVALID ID' })
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    await wrapper.get('[data-testid="activity-config-save"]').trigger('click')
    expect(updateConfig).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('admin.settings.features.activity.validation.activityId')
  })

  it('cannot PUT stale configuration after a later GET failure', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    await wrapper.setProps({ show: false })

    getConfig.mockRejectedValueOnce(new Error('load failed'))
    await wrapper.setProps({ show: true })
    await flushPromises()
    const save = wrapper.get('[data-testid="activity-config-save"]')
    expect(save.attributes('disabled')).toBeDefined()
    await save.trigger('click')
    expect(updateConfig).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('load failed')
  })

  it('associates form labels and exposes prize type and icon button state', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.get('label[for="activity-title"]').exists()).toBe(true)
    expect(wrapper.get('#activity-title').attributes('maxlength')).toBeUndefined()
    expect(wrapper.get('[role="group"]').attributes('aria-label')).toContain('prizeType')
    expect(wrapper.get('[aria-pressed="true"]').text()).toContain('balanceType')
    expect(wrapper.get('button[aria-label="common.delete"]').exists()).toBe(true)
  })

  it('identifies and focuses the exact prize with an empty display label', async () => {
    const current = validConfig()
    getConfig.mockResolvedValue({
      ...current,
      prizes: [{ ...current.prizes[0], label: '' }, current.prizes[1]],
    })
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.get('[data-testid="activity-config-save"]').trigger('click')
    await flushPromises()

    expect(updateConfig).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('admin.settings.features.activity.validation.prizeLabelAt')
    expect(wrapper.get('#activity-prize-0-label').attributes('aria-invalid')).toBe('true')
    expect(document.activeElement?.id).toBe('activity-prize-0-label')
  })
})
