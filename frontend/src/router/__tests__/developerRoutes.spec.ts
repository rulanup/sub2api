import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import { developerRoutes } from '../developerRoutes'

describe('developer portal routes', () => {
  it('resolves the public portal modules without requiring authentication', () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: developerRoutes,
    })

    for (const path of [
      '/developer',
      '/developer/docs/authentication',
      '/developer/api',
      '/developer/playground',
      '/developer/workspace',
      '/developer/debug',
    ]) {
      const route = router.resolve(path)
      expect(route.matched.length).toBeGreaterThan(0)
      expect(route.matched.every((record) => record.meta.requiresAuth === false)).toBe(true)
    }
  })

  it('sends the portal root to the documentation center', () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: developerRoutes,
    })

    expect(router.resolve('/developer').path).toBe('/developer')
    const rootRecord = developerRoutes.find((route) => route.path === '/developer')
    expect(rootRecord?.redirect).toBe('/developer/docs/getting-started')
  })
})
