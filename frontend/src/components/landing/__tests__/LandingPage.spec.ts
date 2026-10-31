import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import { nextTick } from 'vue'

import LandingPage from '../LandingPage.vue'

const landingCss = readFileSync(resolve(process.cwd(), 'src/components/landing/landing.css'), 'utf8')

const setLocale = vi.hoisted(() => vi.fn())
vi.mock('@/i18n', () => ({
  availableLocales: [
    { code: 'en', name: 'English', flag: 'US' },
    { code: 'zh', name: '中文', flag: 'CN' },
  ],
  setLocale,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh' },
    }),
  }
})

function mountLanding(setThemeMode = vi.fn()) {
  return mount(LandingPage, {
    props: {
      siteName: 'Sub2API',
      docUrl: '',
      githubUrl: 'https://github.com/rulanup/sub2api',
      currentYear: 2026,
      startPath: '/register',
      themeMode: 'system',
      setThemeMode,
    },
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
      },
    },
  })
}

describe('Sub2API landing page', () => {
  beforeEach(() => {
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
    setLocale.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the confirmed four-act information architecture', () => {
    const wrapper = mountLanding()
    expect(wrapper.findAll('[data-act]').map((section) => section.attributes('data-act'))).toEqual([
      'hero',
      'optical',
      'story',
      'final',
    ])
    expect(wrapper.find('.optical-stage').exists()).toBe(true)
    expect(wrapper.findAll('.optical-wave-layer path')).toHaveLength(47)
    expect(wrapper.find('.landing-code-window').exists()).toBe(false)
    expect(wrapper.find('.brand-story__lines').findAll('p')).toHaveLength(3)
    expect(wrapper.get('.landing-hero__identity').text()).toContain('Sub2API')
    expect(wrapper.get('.landing-hero__identity').text()).toContain('home.landing.hero.identitySuffix')
    expect(wrapper.get('.landing-scroll-cue').attributes('href')).toBe('#optical')
    expect(wrapper.find('.landing-final-cta__accent').exists()).toBe(true)
    wrapper.unmount()
  })

  it('uses one shared destination for all start actions', () => {
    const wrapper = mountLanding()
    const startLinks = wrapper
      .findAllComponents(RouterLinkStub)
      .filter((link) => link.attributes('data-testid') === 'start-link')
    expect(startLinks).toHaveLength(3)
    expect(startLinks.every((link) => link.props('to') === '/register')).toBe(true)
    wrapper.unmount()
  })

  it('falls back to the built-in developer documentation center', () => {
    const wrapper = mountLanding()

    expect(wrapper.get('.landing-header__docs').attributes('href')).toBe(
      '/developer/docs/getting-started',
    )
    expect(wrapper.findAll('.landing-footer__nav a')[0].attributes('href')).toBe(
      '/developer/docs/getting-started',
    )
    wrapper.unmount()
  })

  it('opens preferences and reports the selected appearance', async () => {
    const setThemeMode = vi.fn()
    const wrapper = mountLanding(setThemeMode)
    await wrapper.get('.landing-preferences__trigger').trigger('click')
    expect(wrapper.get('.landing-preferences__panel').exists()).toBe(true)
    await wrapper.get('[data-testid="theme-dark"]').trigger('click')
    expect(setThemeMode).toHaveBeenCalledWith('dark')
    await wrapper.findAll('.landing-preferences__options--language button')[0].trigger('click')
    expect(setLocale).toHaveBeenCalledWith('en')
    wrapper.unmount()
  })

  it('closes preferences with Escape', async () => {
    const wrapper = mountLanding()
    await wrapper.get('.landing-preferences__trigger').trigger('click')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(wrapper.get('.landing-preferences__trigger').attributes('aria-expanded')).toBe('false')
    wrapper.unmount()
  })

  it('provides a static reduced-motion fallback', () => {
    expect(landingCss).toContain('@media (prefers-reduced-motion: reduce)')
    expect(landingCss).toMatch(/\.optical-wave-layer,\s*\.optical-collapse-line\s*\{[^}]*transform:\s*none/s)
    expect(landingCss).toMatch(/\.brand-story__line\s*\{[^}]*opacity:\s*1/)
    expect(landingCss).toMatch(/\.optical-story\s*\{[^}]*height:\s*auto/)
  })
})
