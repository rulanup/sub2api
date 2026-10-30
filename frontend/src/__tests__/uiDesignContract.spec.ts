import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (path: string): string => readFileSync(resolve(process.cwd(), path), 'utf8')

const globalStyles = readSource('src/style.css')
const themeSource = readSource('src/utils/theme.ts')
const appLayout = readSource('src/components/layout/AppLayout.vue')
const authLayout = readSource('src/components/layout/AuthLayout.vue')
const tablePageLayout = readSource('src/components/layout/TablePageLayout.vue')
const dataTable = readSource('src/components/common/DataTable.vue')
const settingsView = readSource('src/views/admin/SettingsView.vue')

describe('minimal product UI contract', () => {
  it('uses a calm product-blue token set and restrained global geometry', () => {
    expect(globalStyles).toMatch(/--color-primary-500:\s*37 99 235;/)
    expect(themeSource).toMatch(/DEFAULT_THEME_ID\s*=\s*'blue'/)
    expect(globalStyles).toMatch(
      /\.card\s*\{[^}]*@apply bg-white dark:bg-dark-800\/50;[^}]*@apply rounded-lg;[^}]*@apply shadow-none;/s
    )
    expect(globalStyles).toContain('@media (prefers-reduced-motion: reduce)')
  })

  it('keeps the application canvas free from decorative mesh backgrounds', () => {
    expect(appLayout).not.toContain('bg-mesh-gradient')
  })

  it('uses a plain authentication surface instead of glass and gradient text', () => {
    expect(authLayout).not.toMatch(/card-glass|shadow-glass|text-gradient/)
  })

  it('keeps table scrolling in the table container with native-weight scrollbars', () => {
    const tableWrapper = dataTable.match(/\.table-wrapper\s*\{[^}]*\}/s)?.[0] ?? ''
    expect(tableWrapper).toMatch(/overflow-x:\s*auto/)
    expect(dataTable).not.toMatch(/\.table-wrapper[^}]*!important/s)
    expect(dataTable).toMatch(/\.table-wrapper\.table-wrapper::-webkit-scrollbar-thumb\s*\{/s)
    expect(dataTable).not.toMatch(
      /\.is-scrollable \.sticky-col-(?:left|left-second|right)(?:::[a-z-]+)?\s*\{[^}]*linear-gradient/s
    )
  })

  it('reserves the application chrome without clipping table pages', () => {
    expect(tablePageLayout).toContain('height: calc(100vh - 64px - 7rem)')
    expect(tablePageLayout).toMatch(
      /\.table-scroll-container\s*\{[^}]*rounded-lg\b[^}]*shadow-none\b/s
    )
  })

  it('never truncates settings tab labels', () => {
    expect(settingsView).not.toMatch(
      /\.settings-tab-label\s*\{[^}]*(overflow-hidden|text-ellipsis)/s
    )
  })
})
