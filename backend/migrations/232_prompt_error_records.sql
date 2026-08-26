-- Upstream error prompt records: captures full prompt when upstream returns non-2xx.
CREATE TABLE IF NOT EXISTS prompt_error_records (
    id                      BIGSERIAL PRIMARY KEY,
    request_id              VARCHAR(128) NOT NULL DEFAULT '',
    user_id                 BIGINT REFERENCES users(id) ON DELETE SET NULL,
    username_snapshot       VARCHAR(255) NOT NULL DEFAULT '',
    user_email_snapshot     VARCHAR(320) NOT NULL DEFAULT '',
    api_key_id              BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    api_key_name_snapshot   VARCHAR(255) NOT NULL DEFAULT '',
    group_id                BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    group_name              VARCHAR(255) NOT NULL DEFAULT '',
    provider                VARCHAR(64) NOT NULL DEFAULT '',
    endpoint                VARCHAR(128) NOT NULL DEFAULT '',
    protocol                VARCHAR(64) NOT NULL DEFAULT '',
    model                   VARCHAR(255) NOT NULL DEFAULT '',
    prompt_hash             VARCHAR(64) NOT NULL DEFAULT '',
    full_prompt             TEXT NOT NULL DEFAULT '',
    prompt_length           INT NOT NULL DEFAULT 0,
    message_count           INT NOT NULL DEFAULT 0,
    error_status            INT NOT NULL DEFAULT 0,
    error_body              TEXT NOT NULL DEFAULT '',
    error_type              VARCHAR(64) NOT NULL DEFAULT 'upstream_error',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prompt_error_records_created
    ON prompt_error_records(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_error_records_user_created
    ON prompt_error_records(user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_error_records_group_created
    ON prompt_error_records(group_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_error_records_api_key_created
    ON prompt_error_records(api_key_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_error_records_model
    ON prompt_error_records(model);
CREATE INDEX IF NOT EXISTS idx_prompt_error_records_prompt_hash
    ON prompt_error_records(prompt_hash);
CREATE INDEX IF NOT EXISTS idx_prompt_error_records_error_status
    ON prompt_error_records(error_status);
CREATE INDEX IF NOT EXISTS idx_prompt_error_records_request_id
    ON prompt_error_records(request_id);
