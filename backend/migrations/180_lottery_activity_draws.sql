CREATE TABLE IF NOT EXISTS lottery_activity_draws (
    id                           BIGSERIAL PRIMARY KEY,
    activity_id                  VARCHAR(64) NOT NULL,
    user_id                      BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    period_key                   DATE NOT NULL,
    idempotency_hash             CHAR(64) NOT NULL,
    prize_id                     VARCHAR(64) NOT NULL,
    prize_type                   VARCHAR(20) NOT NULL,
    prize_label                  VARCHAR(120) NOT NULL,
    balance_amount               DECIMAL(20, 8),
    balance_before               DECIMAL(20, 8),
    balance_after                DECIMAL(20, 8),
    group_id                     BIGINT REFERENCES groups(id) ON DELETE RESTRICT,
    validity_days                INTEGER,
    subscription_id              BIGINT REFERENCES user_subscriptions(id) ON DELETE SET NULL,
    subscription_expires_before  TIMESTAMPTZ,
    subscription_expires_after   TIMESTAMPTZ,
    config_snapshot              JSONB NOT NULL,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lottery_activity_draws_activity_user_idempotency_hash_unique
        UNIQUE (activity_id, user_id, idempotency_hash),
    CONSTRAINT lottery_activity_draws_prize_type_check
        CHECK (prize_type IN ('balance', 'group')),
    CONSTRAINT lottery_activity_draws_award_shape_check CHECK (
        (prize_type = 'balance' AND balance_amount IS NOT NULL AND group_id IS NULL AND validity_days IS NULL)
        OR
        (prize_type = 'group' AND balance_amount IS NULL AND group_id IS NOT NULL AND validity_days IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_lottery_activity_draws_activity_total
    ON lottery_activity_draws (activity_id, id);

CREATE INDEX IF NOT EXISTS idx_lottery_activity_draws_user_day
    ON lottery_activity_draws (activity_id, user_id, period_key, id);

CREATE INDEX IF NOT EXISTS idx_lottery_activity_draws_user_history
    ON lottery_activity_draws (user_id, created_at DESC, id DESC);
