# User risk score and security audit routing implementation plan

> **For the implementing agent:** REQUIRED SUB-SKILL: Use `executing-plans` to execute this plan task by task.

**Goal:** Add a persistent per-user risk score and use it to route requests through local security policy, OpenAI Moderation API, and the existing prompt audit node without sending high-confidence illegal requests upstream. When an audit node is unavailable, surface an administrator warning and allow the request to continue.

**Architecture:** Keep the existing `securityaudit.Coordinator` as the single pre-upstream audit entry point. Add a local policy evaluator before external audit, and inject a small risk-routing interface into the coordinator. Store scores in a dedicated PostgreSQL profile table with atomic score updates and exponential decay. Expose profiles through the existing admin user list/detail DTOs and render the score/level in the user management table. Preserve the current behavior when the risk router is not wired, so existing isolated coordinator and handler tests remain meaningful.

**Technology:** Go, PostgreSQL migrations, Wire, existing security-audit coordinator, Vue 3/TypeScript frontend, Vitest.

## Task 1: Create the risk profile persistence contract

**Files:**
- Create: `backend/migrations/231_user_risk_profiles.sql`
- Create: `backend/migrations/user_risk_profiles_migration_test.go`
- Create: `backend/internal/repository/user_risk_profile_repo.go`
- Modify: `backend/internal/repository/wire.go`

### Step 1: Write the migration regression test

Add a test that reads `231_user_risk_profiles.sql` and asserts that it contains:

- `user_risk_profiles` with one row per user;
- a foreign key to `users` with delete cleanup;
- a score constrained to `0..100`;
- a level/reason/timestamp/version representation;
- a unique user key and an index useful for administrator listing.
- a separate event/dedupe table with a unique `(user_id, dedupe_key)` key and delete cleanup.

Run from `backend/`:

```bash
go test ./migrations -run TestUserRiskProfilesMigration -count=1
```

The test must fail because the migration and table contract do not exist.

### Step 2: Add the migration

Create `user_risk_profiles` and `user_risk_score_events` with the repository’s existing migration conventions. Use a numeric user ID matching the existing `users.id` type, `score INTEGER NOT NULL DEFAULT 0`, a check constraint for `0..100`, a level string, the last event/decay timestamps, the last reason code, an optimistic version, and created/updated timestamps. Add the profile user uniqueness constraint, an index on `(level, score, updated_at)`, and a unique event key `(user_id, dedupe_key)` so retries are idempotent even when another event occurs between retries.

Re-run the migration test and require it to pass:

```bash
go test ./migrations -run TestUserRiskProfilesMigration -count=1
```

### Step 3: Write repository contract tests

Add repository-focused tests using the existing PostgreSQL test pattern or SQL-shape assertions if the repository package has no database harness. Cover:

- loading a missing profile as the zero profile;
- inserting/upserting a profile for a user;
- returning a batch map keyed by user ID;
- recording an event with a dedupe key without double-applying it;
- applying decay and then clamping the score.

Run the focused repository tests and confirm the initial failure before implementing:

```bash
go test ./internal/repository -run 'TestUserRiskProfile' -count=1
```

### Step 4: Implement repository operations

Implement a repository that provides:

```go
type UserRiskProfileRepository interface {
    Get(ctx context.Context, userID int64) (UserRiskProfileRecord, error)
    GetByUserIDs(ctx context.Context, userIDs []int64) (map[int64]UserRiskProfileRecord, error)
    ApplyEvent(ctx context.Context, userID int64, event UserRiskEventRecord, now time.Time) (bool, error)
}
```

Use a transaction or one atomic SQL statement for each event. Insert the dedupe event first with conflict handling, apply exponential half-life decay before adding a new delta, clamp the score to `0..100`, update the derived level, and roll back the event insert if the profile update fails. Follow the project’s existing repository error, transaction, and placeholder conventions.

Run:

```bash
go test ./internal/repository -run 'TestUserRiskProfile' -count=1
```

## Task 2: Implement score semantics and routing service

**Files:**
- Create: `backend/internal/service/user_risk_score.go`
- Create: `backend/internal/service/user_risk_score_test.go`
- Modify: `backend/internal/service/wire.go`

### Step 1: Write score and route tests

Before production code, add tests for a fake repository covering:

- scores below zero and above 100 are clamped;
- levels are low `0..29`, medium `30..59`, high `60..84`, critical `85..100`;
- a 24-hour half-life decays a score of 80 to approximately 40 after 24 hours;
- concurrent events serialize without lost updates;
- duplicate `(user, reason, dedupe key)` events within the configured window apply once;
- low-risk users with no current local signal skip external AI;
- medium-risk users route to OpenAI Moderation;
- high-risk users route to OpenAI Moderation plus prompt audit when configured;
- any current ambiguous signal routes to AI even for a low-risk profile;
- external unavailability does not increase the score.

Run the focused tests and confirm they fail because the service contract is absent:

```bash
go test ./internal/service -run 'TestUserRisk' -count=1
```

### Step 2: Implement the service

Define the service-facing profile, route, and event types. Use a 24-hour default half-life and the threshold mapping above. Provide methods equivalent to:

```go
Route(ctx context.Context, userID int64, currentSignal bool, promptAuditConfigured bool) (UserRiskRoute, error)
Record(ctx context.Context, userID int64, event UserRiskEvent) error
GetForUsers(ctx context.Context, userIDs []int64) (map[int64]UserRiskProfile, error)
```

Define the coordinator-facing `RiskScoreRouter`, `RiskRoute`, and `RiskEvent` contracts in `backend/internal/service` using only primitive request metadata; the security-audit coordinator already depends on this service boundary for its legacy adapter. Make the score service implement that interface without importing handler packages; bind it in Wire. The coordinator passes `req.UserID`, the local-policy signal, and whether prompt audit is configured, then supplies a request/stage/category dedupe key when recording an event.

Route selection must be deterministic:

- low + no current signal: skip external AI;
- medium: Moderation API;
- high: Moderation API and prompt audit when available/configured;
- critical: all available audit stages;
- current signal always enables at least Moderation API.

Treat a missing profile as score 0/low. Do not auto-ban or expose the raw score to ordinary user endpoints. Add service providers and Wire bindings, then regenerate the server injector.

Run:

```bash
go test ./internal/service -run 'TestUserRisk' -count=1
go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server
go test ./cmd/server ./internal/service ./internal/repository -count=1
```

## Task 3: Add the local network security policy

**Files:**
- Create: `backend/internal/securityaudit/prompt_local_policy.go`
- Create: `backend/internal/securityaudit/prompt_local_policy_test.go`
- Modify: `backend/internal/handler/security_audit_errors.go` if a new error mapping is needed

### Step 1: Write policy tests

Add table-driven tests for normalized prompt snapshots. The policy must block high-confidence requests for:

- jailbreak/破限 or safety bypass;
- DDoS or denial-of-service execution;
- cracking license/product keys;
- unauthorized website intrusion or exploitation;
- stealing cookies, sessions, or authentication tokens;
- phishing pages or credential-harvesting interfaces;
- ransomware or data-encryption extortion;
- bypassing API rate limits or access controls.

Add non-blocking tests for benign security education, defensive browser-security learning, authorized penetration testing, incident response, and a neutral “how to obtain cookies” question without an action/target combination. Verify Unicode/case/whitespace/obfuscation normalization, category/reason-code assignment, no raw prompt evidence in the decision, and that ambiguous but non-blocking signals set `NeedsAI`.

Run before implementation:

```bash
go test ./internal/securityaudit -run 'TestLocalSecurityPolicy' -count=1
```

### Step 2: Implement conservative local evaluation

Use `ExtractPromptSnapshot` and normalize Unicode, case, whitespace, punctuation, and common obfuscation before matching. Require action-plus-target combinations for a block to reduce false positives, and recognize defensive/authorized context as an allow/AI-review signal. Return a stable policy ID, version, category, and reason code, but never return matched text.

Use the client-facing block code `network_security_policy_violation` and message `请求被网络安全策略拦截` through the existing protocol-specific error helpers. Keep this decision local and before account selection, billing, retries, or upstream dispatch.

Run:

```bash
go test ./internal/securityaudit -run 'TestLocalSecurityPolicy' -count=1
go test ./internal/handler -run 'Test.*SecurityAudit|Test.*Audit.*Order' -count=1
```

## Task 4: Route audit stages through risk and make node failure non-blocking

**Files:**
- Modify: `backend/internal/securityaudit/coordinator.go`
- Modify: `backend/internal/securityaudit/prompt_types.go` or the existing shared prompt type file
- Modify: `backend/internal/securityaudit/prompt_service.go`
- Modify: `backend/internal/securityaudit/prompt_metrics.go`
- Modify: `backend/internal/securityaudit/prompt_metrics_test.go`
- Modify: `backend/internal/securityaudit/coordinator_test.go`
- Modify: `backend/internal/securityaudit/prompt_service_test.go`
- Modify: `backend/internal/securityaudit/prompt_module.go`
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/cmd/server/wire_gen.go` via Wire generation

### Step 1: Add failing coordinator tests

Add fakes for legacy audit, prompt audit, and risk routing. Test that:

- a local high-confidence policy match returns a block and neither external audit nor upstream dispatch is reached;
- a low-risk request skips external AI and is allowed;
- a current ambiguous signal routes to AI;
- a prompt/Moderation API block is returned before upstream and produces a risk event;
- a prompt node timeout/unavailability returns `AllowNextStage=true`, increments unavailable metrics, and produces an administrator-visible runtime signal;
- ordinary allow/flag decisions preserve existing protocol behavior.

Run the focused tests and confirm failure before changing coordinator behavior:

```bash
go test ./internal/securityaudit -run 'Test(Coordinator|PromptService|GuardMetrics)' -count=1
```

### Step 2: Implement risk-aware coordinator behavior

Add a `RiskScoreRouter` interface in `securityaudit` and inject it through a setter/provider so existing constructor call sites remain compatible. The coordinator order becomes:

```text
extract snapshot -> local policy -> risk route -> configured external audits -> record result -> upstream
```

For a local block, return a decision that cannot advance. For a low-risk skip, return an allow decision without calling external audit. For medium/high/critical routes, preserve the existing configured stage behavior and record only meaningful score events. Use request/stage/category/reason as the dedupe identity.

Change unavailable/invalid external audit outcomes so they:

- increment guard metrics;
- retain an admin-visible runtime warning/counter;
- set `AllowNextStage=true` and let normal traffic proceed;
- do not add risk points;
- do not become a user-facing block.

Keep legacy/content moderation fallback behavior intact when the coordinator or risk router is absent. Update all affected tests and regenerate Wire.

Run:

```bash
go test ./internal/securityaudit -run 'Test(Coordinator|PromptService|GuardMetrics)' -count=1
go test ./internal/handler -run 'Test.*SecurityAudit|Test.*Audit.*Order' -count=1
go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server
```

## Task 5: Expose risk fields in admin APIs

**Files:**
- Modify: `backend/internal/handler/dto/types.go`
- Modify: `backend/internal/handler/dto/mappers.go` if mapping is centralized there
- Modify: `backend/internal/handler/admin/user_handler.go`
- Modify: the admin handler provider file that constructs `NewUserHandler`
- Create or modify: `backend/internal/handler/admin/user_handler_risk_test.go`

### Step 1: Write handler tests

Add a fake risk-profile reader and test that:

- `GET /admin/users` includes `risk_score`, `risk_level`, `last_risk_reason_code`, and `last_risk_event_at`;
- profiles are batch loaded for the returned user IDs;
- a missing profile is rendered as score 0 and low level;
- a risk repository failure does not make the entire user list unavailable, while the handler logs the failure;
- the user detail response includes the same fields when that endpoint exists.

Run before implementation:

```bash
go test ./internal/handler/admin -run 'Test.*Risk' -count=1
```

### Step 2: Implement DTO and list enrichment

Add risk fields to `AdminUser` only; do not add them to ordinary user response DTOs. Enrich list/detail results with one batch query after loading users. Wire the concrete service through a small interface/setter so existing `NewUserHandler` test constructors do not need unrelated signature churn. Preserve list pagination, filtering, and ordering.

Run:

```bash
go test ./internal/handler/admin ./internal/handler/dto -run 'Test.*(Risk|User)' -count=1
go test ./internal/handler -count=1
```

## Task 6: Render score and administrator availability warning in the frontend

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/views/admin/UsersView.vue`
- Modify: `frontend/src/features/prompt-audit/components/RuntimeOverview.vue`
- Modify: `frontend/src/i18n/locales/zh/admin/promptAudit.ts`
- Modify: `frontend/src/i18n/locales/en/admin/promptAudit.ts`
- Add/update the nearest existing frontend tests for user management and prompt-audit runtime

### Step 1: Write frontend tests

Add tests that assert:

- the user table renders risk score and localized risk level;
- low/medium/high/critical values have distinct visual labels;
- missing fields fall back to score 0/low;
- the prompt-audit runtime view shows a warning when unavailable count is nonzero and does not show it when zero;
- the warning tells administrators that normal traffic is being allowed and audit-node recovery/configuration is needed.

Run in the frontend environment:

```bash
npm run test -- --run
```

The new assertions must fail before the UI changes.

### Step 2: Implement UI changes

Extend `AdminUser` types, add sortable risk score/level columns to `UsersView.vue`, and use existing table/i18n patterns. Add the runtime warning using the existing `guard_metrics.unavailable` data and localized Chinese/English copy. Do not display prompt contents or sensitive evidence in the UI.

Run:

```bash
npm run test -- --run
npm run type-check
npm run build
```

If WSL lacks the native Node runtime, run these commands with the workspace-bundled Node runtime in an isolated copy, as used by the existing frontend verification workflow, and record the exact command/output.

## Task 7: End-to-end verification and integration commit

**Files:**
- Modify only files required by the preceding tasks.

### Step 1: Run backend focused and full tests

From `backend/`:

```bash
go test ./internal/securityaudit ./internal/service ./internal/repository ./internal/handler/... ./migrations -count=1
go test ./... -count=1
```

Verify the security-audit order test still proves the block occurs before upstream dispatch, and verify that an unavailable audit node allows a normal request while producing the runtime/admin warning.

### Step 2: Run frontend verification

Run the full frontend test suite, type check, and production build using the available Node runtime:

```bash
npm run test -- --run
npm run type-check
npm run build
```

### Step 3: Inspect the final diff

Run:

```bash
git status --short
git diff --check
git diff --stat
git log -5 --oneline
```

Confirm that no prompt text, API keys, cookies, or matched evidence is persisted or returned to ordinary users, that only local policy blocks the listed high-confidence categories, and that upstream dispatch is impossible after a local block. Commit the implementation as one focused change with a Conventional Commit message such as:

```text
feat: add user risk scoring and security audit routing
```
