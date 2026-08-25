package securityaudit

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/securitytext"
)

const (
	ErrorCodeNetworkSecurityPolicyViolation = "network_security_policy_violation"
	NetworkSecurityPolicyID                 = "network_security_policy"
	NetworkSecurityPolicyVersion            = 1
)

type LocalPolicyDecision struct {
	Blocked       bool
	NeedsAI       bool
	Category      string
	ReasonCode    string
	ErrorCode     string
	PolicyID      string
	PolicyVersion int
}

type localPolicyRule struct {
	category  string
	reason    string
	actions   []string
	targets   []string
	ambiguous []string
}

var localPolicyRules = []localPolicyRule{
	{
		category: "api_rate_limit_bypass", reason: "bypass_api_rate_limit",
		actions: []string{"绕过", "规避", "突破", "bypass", "evade", "circumvent"},
		targets: []string{"api限流", "限流", "速率限制", "rate limit", "rate limiting", "throttle", "throttling", "配额", "quota", "访问控制", "access control"},
	},
	{
		category: "cookie_theft", reason: "cookie_or_session_theft",
		actions:   []string{"窃取", "盗取", "偷取", "窃听", "盗走", "steal", "exfiltrate", "hijack"},
		targets:   []string{"cookie", "kookie", "session", "会话", "token", "令牌", "身份凭证"},
		ambiguous: []string{"获取", "抓取", "导出", "提取", "读取", "查看", "get", "extract", "dump", "collect"},
	},
	{
		category: "phishing", reason: "phishing_or_credential_harvesting",
		actions: []string{"制作", "搭建", "仿冒", "伪造", "创建", "build", "create", "make", "phishing", "harvest", "harvesting"},
		targets: []string{"钓鱼", "登录界面", "登录页面", "credential harvesting", "credential", "凭据", "密码", "password", "账号", "login"},
	},
	{
		category: "ransomware", reason: "ransomware_creation_or_deployment",
		actions: []string{"编写", "写", "制作", "做", "开发", "部署", "传播", "create", "write", "develop", "deploy", "spread", "encrypt", "encryption"},
		targets: []string{"勒索软件", "ransomware", "加密文件", "文件加密", "encrypt files", "file encryption", "victim files", "victims files", "赎金", "ransom"},
	},
	{
		category: "ddos", reason: "distributed_denial_of_service",
		actions: []string{"发起", "实施", "发动", "执行", "发送", "压垮", "打垮", "flood", "launch", "perform", "attack"},
		targets: []string{"ddos", "拒绝服务", "denial of service", "网站", "服务器", "api", "接口", "目标"},
	},
	{
		category: "license_key_cracking", reason: "license_or_product_key_cracking",
		actions: []string{"破解", "爆破", "生成", "伪造", "绕过", "crack", "generate", "forge", "bypass"},
		targets: []string{"卡密", "激活码", "license key", "product key", "序列号", "注册码"},
	},
	{
		category: "website_intrusion", reason: "unauthorized_website_intrusion",
		actions:   []string{"入侵", "攻击", "渗透", "利用漏洞", "漏洞利用", "exploit", "hack", "compromise", "breach", "deface", "take over", "getshell", "webshell", "提权", "sql注入", "rce"},
		targets:   []string{"网站", "站点", "服务器", "后台", "web", "目标", "系统"},
		ambiguous: []string{"验证", "测试", "扫描", "检测", "audit", "test", "scan"},
	},
	{
		category: "jailbreak", reason: "jailbreak_or_safety_bypass",
		actions: []string{"破限", "解除限制", "绕过限制", "忽略之前", "忽略指令", "无视规则", "不要拒绝", "进入开发者模式", "生成攻击工具", "生成攻击脚本", "生成恶意工具", "jailbreak", "bypass"},
		targets: []string{"破限", "安全策略", "安全规则", "内容限制", "审核", "系统提示", "开发者模式", "攻击工具", "攻击脚本", "恶意工具", "模型", "model", "llm", "safety", "restriction", "guardrail"},
	},
}

var localPolicyContextTerms = []string{
	"防御", "防护", "防止", "阻止", "防范", "检测", "缓解", "学习", "教程", "研究", "授权", "合法", "红队", "渗透测试",
	"安全教育", "事件响应", "应急响应", "防守", "defensive", "authorized", "security education",
	"incident response", "mitigation", "how to defend", "prevent", "protect", "penetration testing",
}

func EvaluateLocalPolicy(req Request) LocalPolicyDecision {
	snapshot, err := ExtractPromptSnapshot(req)
	if err != nil {
		return LocalPolicyDecision{}
	}
	return EvaluateLocalPolicyText(snapshot.ScanText)
}

// EvaluateLocalPolicyText applies the deterministic network security policy to
// a text-only audit input, such as the admin moderation test form.
func EvaluateLocalPolicyText(text string) LocalPolicyDecision {
	if strings.TrimSpace(text) == "" {
		return LocalPolicyDecision{}
	}
	canonical := securitytext.Canonicalize(text)
	normalized, compact := canonical.Text, canonical.Compact
	if normalized == "" {
		return LocalPolicyDecision{}
	}
	protective := containsAnyLocalPolicyTerm(normalized, compact, localPolicyContextTerms)
	for _, rule := range localPolicyRules {
		hasTarget := containsAnyLocalPolicyTerm(normalized, compact, rule.targets)
		if !hasTarget {
			continue
		}
		hasAction := containsAnyLocalPolicyTerm(normalized, compact, rule.actions)
		hasAmbiguous := containsAnyLocalPolicyTerm(normalized, compact, rule.ambiguous)
		if hasAction {
			if protective {
				return localPolicyReviewDecision(rule)
			}
			return localPolicyBlockDecision(rule)
		}
		if hasAmbiguous || protective {
			return localPolicyReviewDecision(rule)
		}
	}
	return LocalPolicyDecision{}
}

func localPolicyBlockDecision(rule localPolicyRule) LocalPolicyDecision {
	return LocalPolicyDecision{
		Blocked:       true,
		Category:      rule.category,
		ReasonCode:    rule.reason,
		ErrorCode:     ErrorCodeNetworkSecurityPolicyViolation,
		PolicyID:      NetworkSecurityPolicyID,
		PolicyVersion: NetworkSecurityPolicyVersion,
	}
}

func localPolicyReviewDecision(rule localPolicyRule) LocalPolicyDecision {
	return LocalPolicyDecision{
		NeedsAI:       true,
		Category:      rule.category,
		ReasonCode:    rule.reason,
		PolicyID:      NetworkSecurityPolicyID,
		PolicyVersion: NetworkSecurityPolicyVersion,
	}
}

func containsAnyLocalPolicyTerm(normalized, compact string, terms []string) bool {
	normalizedFolded := strings.ToLower(normalized)
	for _, term := range terms {
		canonical := securitytext.Canonicalize(term)
		termNormalized, termCompact := canonical.Text, canonical.Compact
		if termNormalized != "" && strings.Contains(normalizedFolded, strings.ToLower(termNormalized)) {
			return true
		}
		if termCompact != "" && strings.Contains(compact, termCompact) {
			return true
		}
	}
	return false
}
