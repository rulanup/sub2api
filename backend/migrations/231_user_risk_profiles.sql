-- Per-user security risk state. Scores are deliberately kept separate from
-- the users table so ordinary user responses never expose this internal data.
CREATE TABLE IF NOT EXISTS user_risk_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    score INTEGER NOT NULL DEFAULT 0,
    level TEXT NOT NULL DEFAULT 'low',
    last_event_at TIMESTAMPTZ,
    last_decay_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_reason_code TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_risk_profiles_score_range CHECK (score >= 0 AND score <= 100),
    CONSTRAINT user_risk_profiles_level_valid CHECK (level IN ('low', 'medium', 'high', 'critical'))
);

CREATE INDEX IF NOT EXISTS idx_user_risk_profiles_level_score_updated
    ON user_risk_profiles (level, score, updated_at DESC);

-- The unique key makes score updates idempotent across retries and concurrent
-- workers, even if a different risk event is recorded between retries.
CREATE TABLE IF NOT EXISTS user_risk_score_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dedupe_key TEXT NOT NULL,
    reason_code TEXT NOT NULL,
    delta INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, dedupe_key)
);

CREATE INDEX IF NOT EXISTS idx_user_risk_score_events_user_created
    ON user_risk_score_events (user_id, created_at DESC);
