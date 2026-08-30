-- 本地内置审查策略命中后的处理动作。
-- allow：直接放行；review：标记可疑并送审查模型（无模型时放行）；block：直接拦截。
-- 默认 review，保持迁移前版本的当前行为。

INSERT INTO settings (key, value, updated_at)
VALUES ('local_audit_policy_action', 'review', NOW())
ON CONFLICT (key) DO NOTHING;
