import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import SearchableUserAllowlistSelector from '../SearchableUserAllowlistSelector.vue'

const messages: Record<string, string> = {
  'admin.riskControl.userAllowlistDeleted': '(deleted)',
  'admin.riskControl.userAllowlistFallback': 'User #{id}',
  'admin.riskControl.userAllowlistRemove': 'Remove user',
  'admin.riskControl.userAllowlistSearchPlaceholder': 'Search users',
  'admin.riskControl.userAllowlistSearchEmpty': 'No users found',
  'common.loading': 'Loading',
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const message = messages[key] ?? key
      return params
        ? Object.entries(params).reduce(
            (value, [name, replacement]) => value.replace(`{${name}}`, String(replacement)),
            message,
          )
        : message
    },
  }),
}))

const mockSearchUsers = vi.fn()
const mockGetUserById = vi.fn()

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: { searchUsers: (...args: unknown[]) => mockSearchUsers(...args) },
    users: { getById: (...args: unknown[]) => mockGetUserById(...args) },
  },
}))

describe('SearchableUserAllowlistSelector', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockSearchUsers.mockReset()
    mockGetUserById.mockReset()
  })

  afterEach(() => vi.useRealTimers())

  it('hydrates selected users and preserves deleted state', async () => {
    mockGetUserById.mockResolvedValue({ id: 7, email: 'deleted@example.com', deleted_at: '2026-01-01' })
    const wrapper = mount(SearchableUserAllowlistSelector, {
      props: { modelValue: [7] },
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    expect(mockGetUserById).toHaveBeenCalledWith(7, true)
    expect(wrapper.text()).toContain('deleted@example.com')
    expect(wrapper.text()).toContain('(deleted)')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('searches, adds a result, and excludes already selected users', async () => {
    mockSearchUsers.mockResolvedValue([
      { id: 3, email: 'selected@example.com', deleted: false },
      { id: 9, email: 'new@example.com', deleted: false },
    ])
    mockGetUserById.mockResolvedValue({ id: 3, email: 'selected@example.com', deleted_at: null })
    const wrapper = mount(SearchableUserAllowlistSelector, {
      props: { modelValue: [3] },
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    const input = wrapper.get('input')
    await input.setValue('example')
    await input.trigger('input')
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(mockSearchUsers).toHaveBeenCalledWith('example')
    expect(wrapper.findAll('button').filter((button) => button.text().includes('selected@example.com'))).toHaveLength(0)
    const result = wrapper.findAll('button').find((button) => button.text().includes('new@example.com'))
    expect(result).toBeDefined()
    await result!.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[[3, 9]]])
  })

  it('keeps unresolved IDs visible and supports removal', async () => {
    mockGetUserById.mockRejectedValue(new Error('not found'))
    const wrapper = mount(SearchableUserAllowlistSelector, {
      props: { modelValue: [42] },
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('User #42')
    await wrapper.get('button[aria-label="Remove user"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[[]]])
  })

  it('closes search results after an outside click', async () => {
    mockSearchUsers.mockResolvedValue([{ id: 9, email: 'new@example.com', deleted: false }])
    const wrapper = mount(SearchableUserAllowlistSelector, {
      props: { modelValue: [] },
      attachTo: document.body,
      global: { stubs: { Icon: true } },
    })
    const input = wrapper.get('input')
    await input.setValue('new')
    await input.trigger('input')
    vi.advanceTimersByTime(300)
    await flushPromises()
    expect(wrapper.text()).toContain('new@example.com')

    document.body.click()
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).not.toContain('new@example.com')
    wrapper.unmount()
  })
})
