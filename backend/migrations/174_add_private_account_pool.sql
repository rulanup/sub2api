-- 添加 user_id 到 accounts 表，支持私人号池
-- user_id 为 NULL 表示系统公共账号，非 NULL 表示用户私人账号
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id) WHERE user_id IS NOT NULL;

-- 添加 is_private 字段到 groups 表，支持私人分组
ALTER TABLE groups ADD COLUMN IF NOT EXISTS is_private BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS owner_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_groups_owner_user_id ON groups(owner_user_id) WHERE owner_user_id IS NOT NULL;
