import { readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (path: string): string => readFileSync(resolve(process.cwd(), path), 'utf8')

const readVueSources = (directory: string): Array<[string, string]> => {
  const sources: Array<[string, string]> = []

  for (const entry of readdirSync(resolve(process.cwd(), directory), { withFileTypes: true })) {
    const path = `${directory}/${entry.name}`
    if (entry.isDirectory()) {
      sources.push(...readVueSources(path))
    } else if (entry.isFile() && entry.name.endsWith('.vue')) {
      sources.push([path, readSource(path)])
    }
  }

  return sources
}

const allVueSources = readVueSources('src')

const globalStyles = readSource('src/style.css')
const themeSource = readSource('src/utils/theme.ts')
const appLayout = readSource('src/components/layout/AppLayout.vue')
const authLayout = readSource('src/components/layout/AuthLayout.vue')
const tablePageLayout = readSource('src/components/layout/TablePageLayout.vue')
const dataTable = readSource('src/components/common/DataTable.vue')
const settingsView = readSource('src/views/admin/SettingsView.vue')
const userDashboardStats = readSource('src/components/user/dashboard/UserDashboardStats.vue')
const userDashboardQuickActions = readSource(
  'src/components/user/dashboard/UserDashboardQuickActions.vue'
)
const userDashboardRecentUsage = readSource(
  'src/components/user/dashboard/UserDashboardRecentUsage.vue'
)
const profileInfoCard = readSource('src/components/user/profile/ProfileInfoCard.vue')
const profileView = readSource('src/views/user/ProfileView.vue')
const keysView = readSource('src/views/user/KeysView.vue')
const usageView = readSource('src/views/user/UsageView.vue')
const channelStatusView = readSource('src/views/user/ChannelStatusV2View.vue')
const announcementPopup = readSource('src/components/common/AnnouncementPopup.vue')
const chatPanel = readSource('src/components/workbench/ChatPanel.vue')
const imagePanel = readSource('src/components/workbench/ImagePanel.vue')
const monitorCard = readSource('src/components/user/monitor/MonitorCard.vue')
const monitorCardGrid = readSource('src/components/user/monitor/MonitorCardGrid.vue')
const subscriptionsView = readSource('src/views/user/SubscriptionsView.vue')
const modelSquareView = readSource('src/views/user/ModelSquareView.vue')
const paymentView = readSource('src/views/user/PaymentView.vue')
const opsDashboardHeader = readSource(
  'src/views/admin/ops/components/OpsDashboardHeader.vue'
)
const opsErrorDetailModal = readSource(
  'src/views/admin/ops/components/OpsErrorDetailModal.vue'
)
const opsRequestDetailsModal = readSource(
  'src/views/admin/ops/components/OpsRequestDetailsModal.vue'
)
const opsErrorLogTable = readSource('src/views/admin/ops/components/OpsErrorLogTable.vue')
const opsSystemLogTable = readSource(
  'src/views/admin/ops/components/OpsSystemLogTable.vue'
)
const keyUsageView = readSource('src/views/KeyUsageView.vue')
const batchImageGuideView = readSource('src/views/user/BatchImageGuideView.vue')
const homeView = readSource('src/views/HomeView.vue')
const setupWizardView = readSource('src/views/setup/SetupWizardView.vue')
const checkinView = readSource('src/views/user/CheckinView.vue')
const stripePaymentView = readSource('src/views/user/StripePaymentView.vue')
const stripePaymentInline = readSource('src/components/payment/StripePaymentInline.vue')
const workbenchView = readSource('src/views/user/WorkbenchView.vue')
const customPageView = readSource('src/views/user/CustomPageView.vue')
const userDashboardCharts = readSource(
  'src/components/user/dashboard/UserDashboardCharts.vue'
)
const promptAuditView = readSource('src/features/prompt-audit/PromptAuditView.vue')
const promptErrorRecordsView = readSource(
  'src/features/prompt-error-records/PromptErrorRecordsView.vue'
)
const endpointPool = readSource(
  'src/features/prompt-audit/components/EndpointPool.vue'
)
const runtimeOverview = readSource(
  'src/features/prompt-audit/components/RuntimeOverview.vue'
)
const tokenRankingView = readSource('src/views/user/TokenRankingView.vue')
const availableChannelsTable = readSource(
  'src/components/channels/AvailableChannelsTable.vue'
)
const auditLogView = readSource('src/views/admin/AuditLogView.vue')
const riskControlView = readSource('src/views/admin/RiskControlView.vue')
const backupView = readSource('src/views/admin/BackupView.vue')
const adminSubscriptionsView = readSource('src/views/admin/SubscriptionsView.vue')
const adminChannelsView = readSource('src/views/admin/ChannelsView.vue')
const adminGroupsView = readSource('src/views/admin/GroupsView.vue')
const adminAccountsView = readSource('src/views/admin/AccountsView.vue')
const accountUsageCell = readSource('src/components/account/AccountUsageCell.vue')
const openAIQuotaResetCell = readSource(
  'src/components/account/OpenAIQuotaResetCell.vue'
)
const claudeResetCreditsCell = readSource(
  'src/components/account/ClaudeResetCreditsCell.vue'
)
const usageTable = readSource('src/components/admin/usage/UsageTable.vue')
const quotaBadge = readSource('src/components/account/QuotaBadge.vue')
const capacityBadge = readSource('src/components/account/CapacityBadge.vue')
const usageProgressBar = readSource('src/components/account/UsageProgressBar.vue')
const platformTypeBadge = readSource('src/components/common/PlatformTypeBadge.vue')
const subscriptionProgressMini = readSource(
  'src/components/common/SubscriptionProgressMini.vue'
)
const versionBadge = readSource('src/components/common/VersionBadge.vue')
const platformUsageBreakdown = readSource(
  'src/components/user/PlatformUsageBreakdown.vue'
)
const userErrorRequestsTable = readSource(
  'src/components/user/UserErrorRequestsTable.vue'
)

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

  it('keeps the user dashboard focused on four core metrics', () => {
    expect(userDashboardStats).toContain('data-testid="user-core-metrics"')
    expect(userDashboardStats).not.toMatch(
      /bg-(?:emerald|green|purple|amber|indigo|violet|rose)-100/
    )
    expect(userDashboardQuickActions).not.toContain('group-hover:scale-105')
    expect(userDashboardRecentUsage).not.toContain('text-green-600')
  })

  it('uses solid profile surfaces without decorative gradients or elevated cards', () => {
    expect(profileInfoCard).not.toMatch(
      /bg-gradient|shadow-lg|rounded-\[1\.75rem\]|rounded-3xl/
    )
    expect(profileView).not.toContain('card border-primary-200 bg-primary-50')
  })

  it('uses restrained overlays and monitor surfaces on high-frequency pages', () => {
    expect(keysView).not.toContain('shadow-lg')
    expect(usageView).not.toContain('shadow-lg')
    expect(channelStatusView).not.toMatch(/rounded-3xl|backdrop-blur/)
    expect(announcementPopup).not.toMatch(/rounded-3xl|shadow-2xl/)
  })

  it('keeps workbench composer and sheets in normal document flow without heavy elevation', () => {
    expect(chatPanel).not.toMatch(/shadow-(?:lg|xl)|backdrop-blur|rounded-3xl/)
    expect(imagePanel).not.toMatch(/shadow-(?:lg|xl)|backdrop-blur|rounded-3xl/)
    expect(imagePanel).not.toContain('pointer-events-none')
  })

  it('keeps monitor cards solid and compact', () => {
    expect(monitorCard).not.toMatch(/rounded-(?:2xl|3xl)|backdrop-blur|shadow-(?:lg|xl)/)
    expect(monitorCardGrid).not.toMatch(/rounded-2xl|bg-white\/70|backdrop-blur|shadow-(?:lg|xl)/)
  })

  it('keeps subscription, model and payment pages free from decorative gradients', () => {
    for (const source of [subscriptionsView, modelSquareView, paymentView]) {
      expect(source).not.toMatch(/bg-gradient|rounded-3xl|backdrop-blur/)
    }
    expect(paymentView).not.toMatch(/shadow-(?:lg|xl|2xl)/)
  })

  it('keeps ops surfaces compact, solid and legible', () => {
    expect(opsDashboardHeader).not.toMatch(
      /rounded-3xl|shadow-xl|ring-1 ring-gray-900\/5/
    )
    expect(opsDashboardHeader).not.toMatch(/text-\[(?:9|10|11)px\]/)
    expect(opsDashboardHeader).not.toMatch(/font-black|uppercase tracking-wider/)
    expect(opsDashboardHeader).not.toContain('·')

    for (const source of [
      opsErrorDetailModal,
      opsRequestDetailsModal,
      opsErrorLogTable,
      opsSystemLogTable
    ]) {
      expect(source).not.toMatch(/rounded-3xl|shadow-(?:xl|2xl)|backdrop-blur/)
    }
    expect(opsErrorDetailModal).not.toMatch(/font-black|uppercase tracking-wider/)
    expect(opsRequestDetailsModal).not.toMatch(
      /text-\[(?:9|10|11)px\]|font-bold uppercase tracking-wider/
    )
    expect(opsSystemLogTable).not.toMatch(/text-\[(?:9|10|11)px\]/)
  })

  it('keeps key usage analytics solid, legible and product-focused', () => {
    expect(keyUsageView).not.toMatch(
      /rounded-(?:2xl|3xl)|bg-white\/90|backdrop-blur|shadow-(?:md|lg|xl|2xl)/
    )
    expect(keyUsageView).not.toMatch(/uppercase tracking-wider|text-\[(?:9|10|11)px\]/)
    expect(keyUsageView).not.toContain('#14b8a6')
  })

  it('uses restrained popovers and plain labels in the batch image guide', () => {
    expect(batchImageGuideView).not.toMatch(/shadow-(?:xl|2xl)/)
    expect(batchImageGuideView).not.toContain(' · ')
  })

  it('keeps public, setup and check-in surfaces free from decorative glass effects', () => {
    for (const source of [
      homeView,
      setupWizardView,
      checkinView,
      stripePaymentView,
      stripePaymentInline,
      customPageView
    ]) {
      expect(source).not.toMatch(
        /backdrop-blur|bg-mesh-gradient|card-glass|shadow-glass|shadow-glow|text-gradient/
      )
    }
    expect(homeView).not.toMatch(/hover:shadow-(?:lg|xl|2xl)|group-hover:scale-/)
    expect(checkinView).not.toMatch(/bg-gradient|shadow-(?:lg|xl)|active:scale-/)
  })

  it('keeps workbench and dashboard support surfaces in normal document flow', () => {
    expect(workbenchView).not.toMatch(/backdrop-blur|bg-white\/70|shadow-(?:xl|2xl)/)
    expect(userDashboardCharts).not.toMatch(/backdrop-blur|bg-white\/50/)
  })

  it('uses plain product labels throughout audit and operations surfaces', () => {
    for (const source of [
      promptAuditView,
      promptErrorRecordsView,
      endpointPool,
      runtimeOverview,
      tokenRankingView,
      auditLogView,
      riskControlView
    ]) {
      expect(source).not.toMatch(/text-\[(?:9|10|11)px\]/)
      expect(source).not.toMatch(/font-black|uppercase tracking-wider/)
    }
    expect(endpointPool).not.toMatch(/border-l-\[3px\]|bg-gradient/)
    expect(promptAuditView).not.toContain('backdrop-blur')
    expect(promptErrorRecordsView).not.toMatch(/shadow-(?:xl|2xl)/)
    expect(auditLogView).not.toContain('font-bold uppercase tracking-wider')
  })

  it('keeps high-frequency tables and badges at a legible 12px minimum', () => {
    for (const source of [
      availableChannelsTable,
      adminChannelsView,
      adminGroupsView,
      adminAccountsView,
      accountUsageCell,
      openAIQuotaResetCell,
      claudeResetCreditsCell,
      usageTable,
      quotaBadge,
      capacityBadge,
      usageProgressBar,
      platformTypeBadge,
      subscriptionProgressMini,
      versionBadge,
      platformUsageBreakdown,
      userErrorRequestsTable
    ]) {
      expect(source).not.toMatch(/text-\[(?:9|10|11)px\]/)
    }
  })

  it('keeps subscription and backup dialogs free from hover elevation and heavy shadows', () => {
    expect(subscriptionsView).not.toMatch(/hover:shadow|hover:-translate|rounded-3xl/)
    expect(adminSubscriptionsView).not.toMatch(/shadow-2xl|bg-gradient/)
    expect(backupView).not.toMatch(/shadow-2xl/)
  })

  it('applies the 12px text minimum and compact surface radius across every Vue view', () => {
    const offenders = allVueSources
      .filter(([, source]) => /text-\[(?:[0-9]|1[01])px\]|rounded-(?:2xl|3xl)/.test(source))
      .map(([path]) => path)

    expect(offenders).toEqual([])
  })

  it('reserves gradients for data encoding and keeps every page free from glass effects', () => {
    const offenders = allVueSources
      .filter(
        ([path, source]) =>
          path !== 'src/components/admin/usage/UsageTable.vue' &&
          /bg-gradient-to-|backdrop-blur|blur-(?:2xl|3xl)/.test(source)
      )
      .map(([path]) => path)

    expect(offenders).toEqual([])
  })
})
