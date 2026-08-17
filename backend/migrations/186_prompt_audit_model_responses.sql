-- Optional full downstream model responses for prompt-audit event detail.
-- The correlation key permits the response and async audit event to arrive in
-- either order while job_id provides lifecycle ownership and cascade deletion.
CREATE TABLE IF NOT EXISTS prompt_audit_model_responses (
    id            BIGSERIAL PRIMARY KEY,
    job_id        BIGINT UNIQUE NOT NULL REFERENCES prompt_audit_jobs(id) ON DELETE CASCADE,
    request_id    VARCHAR(128) NOT NULL,
    stage         VARCHAR(64) NOT NULL DEFAULT 'http',
    response_body BYTEA NOT NULL DEFAULT ''::bytea,
    truncated     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_prompt_audit_model_responses_correlation UNIQUE (request_id, stage),
    CONSTRAINT chk_prompt_audit_model_responses_body_size CHECK (octet_length(response_body) <= 65536)
);

CREATE INDEX IF NOT EXISTS idx_prompt_audit_model_responses_job
    ON prompt_audit_model_responses(job_id);
