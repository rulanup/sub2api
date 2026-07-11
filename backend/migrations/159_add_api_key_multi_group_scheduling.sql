-- API keys can optionally schedule across multiple groups.
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS group_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS group_schedule_strategy VARCHAR(32) NOT NULL DEFAULT 'cheapest';

UPDATE api_keys
SET group_ids = jsonb_build_array(group_id)
WHERE group_id IS NOT NULL
  AND (group_ids IS NULL OR group_ids = '[]'::jsonb);

CREATE INDEX IF NOT EXISTS idx_api_keys_group_ids_gin ON api_keys USING GIN (group_ids);
