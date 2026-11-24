-- 分组级系统提示词改为单份文本（一个分组只属于一个平台，注入位置由请求协议决定）。
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS system_prompt TEXT NOT NULL DEFAULT '';

ALTER TABLE groups
    DROP COLUMN IF EXISTS system_prompts;

COMMENT ON COLUMN groups.system_prompt IS
    '分组级系统提示词文本；注入到 system 开头（或 if_absent 时仅在客户端未提供时注入）';
