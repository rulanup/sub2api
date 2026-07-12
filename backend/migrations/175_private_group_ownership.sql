DROP INDEX IF EXISTS groups_name_active_unique;

CREATE UNIQUE INDEX IF NOT EXISTS groups_public_name_active_unique
    ON groups (name)
    WHERE deleted_at IS NULL AND is_private = false;

CREATE UNIQUE INDEX IF NOT EXISTS groups_private_owner_name_active_unique
    ON groups (owner_user_id, name)
    WHERE deleted_at IS NULL AND is_private = true;

ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_private_owner_check;
ALTER TABLE groups ADD CONSTRAINT groups_private_owner_check CHECK (
    (is_private = false AND owner_user_id IS NULL)
    OR (is_private = true AND owner_user_id IS NOT NULL)
);
