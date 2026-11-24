-- 分组级系统提示词注入与安全审查开关（migration 253）。
-- 三态语义：NULL 表示继承全局 prompt-audit 配置，保持老分组行为不变。
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS system_prompt_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS system_prompts JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS system_prompt_mode VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS security_audit_enabled BOOLEAN;

COMMENT ON COLUMN groups.system_prompt_enabled IS
    '分组级系统提示词注入开关；NULL 继承全局 prompt-audit 配置，false 强制不注入';
COMMENT ON COLUMN groups.system_prompts IS
    '按协议的系统提示词文本，键：anthropic_messages/openai/gemini；空文本表示该协议不注入';
COMMENT ON COLUMN groups.system_prompt_mode IS
    '注入位置：prepend（默认，追加到 system 开头）或 if_absent（客户端已有 system 则不注入）';
COMMENT ON COLUMN groups.security_audit_enabled IS
    '分组级安全审查开关；NULL 按全局分组范围，false 强制跳过本地策略与审查模型';
