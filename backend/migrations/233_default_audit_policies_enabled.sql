-- 内置默认审查策略开关（本地安全策略初筛、上游错误提示词记录）。
-- 初筛命中的请求交由审查模型裁决，未配置审查模型时放行。
-- 默认开启以保持既有行为；关闭后仅保留显式配置的内容审核与提示词审查链路。

INSERT INTO settings (key, value, updated_at)
VALUES ('default_audit_policies_enabled', 'true', NOW())
ON CONFLICT (key) DO NOTHING;
