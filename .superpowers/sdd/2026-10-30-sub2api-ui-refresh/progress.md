# SDD ledger — plan: docs/superpowers/plans/2026-10-30-sub2api-ui-refresh.md

Spec: docs/superpowers/specs/2026-10-30-sub2api-ui-refresh-design.md
Branch: codex/ui-refresh
Plan commit: cbfe14ef4

Pre-flight: shared interfaces are the global token contract (`style.css`, `tailwind.config.js`), the compatibility utility classes used by all pages, and the source-reading design contract test. No conflict found; later tasks consume tokens and compatibility classes established in Task 1.

Task 1: complete (design contract RED 5/5; token and reduced-motion assertions GREEN, 1/5 contract tests passing before layout work; commits 8c7c2791a..HEAD)
Task 2: complete (application mesh removed; header, sidebar, and theme picker simplified; contract 2/5 passing with remaining failures scoped to auth, table, and settings; `pnpm typecheck` passed)
Task 3: complete (authentication shell uses a plain neutral canvas and solid card; login, registration, and password recovery titles aligned; auth contract assertion and i18n completeness passed)
Task 4: complete (table shell uses reserved application-chrome height, neutral fixed-column boundaries, visible low-contrast scrolling, compact rows and unified 8px controls; `pnpm typecheck` passed; contract 5/6 passing and full suite 3176/3177 passing)

Ruling: Task 4 accepts the intentional settings-tab contract failure as Task 5's RED test. The failing assertion is limited to `SettingsView.vue` label truncation; all 390 other test files and all 3176 non-contract tests pass.
Task 5: complete (admin dashboard now prioritizes four core metrics with a compact operational summary; settings tabs use an untruncated underline pattern with narrow-screen scrolling; admin overlays use restrained floating surfaces; design contract 6/6, i18n 3/3, and `pnpm typecheck` passed)
Task 6: Ruling: 12px minimum font size applies to every Vue UI surface, including low-frequency account, monitoring, and model-plaza components — the contract is a product-wide readability floor, and exempting rare screens would keep the interface internally inconsistent — if wrong, very dense operational cells may need more horizontal space after real-device inspection.
Task 6: complete (all decorative Vue residuals cleared except the UsageTable latency severity data scale; recursive design contract 22/22 passed; full suite 3194/3194 passed)
Task 7: complete (final verification: `vitest run --minWorkers=1 --maxWorkers=2` → 391 files / 3195 tests passed, exit 0; i18n 3/3; `vue-tsc --noEmit` exit 0; frontend production build exit 0; Go backend build exit 0; login and representative desktop/tablet/mobile, light/dark, zh/en pages checked without console errors)
