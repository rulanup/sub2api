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
	category       string
	reason         string
	actions        []string
	targets        []string
	ambiguous      []string
	strongActions  []string
	strongTargets  []string
	genericTargets []string
}

var localPolicyRules = []localPolicyRule{
	{
		category: "api_rate_limit_bypass", reason: "bypass_api_rate_limit",
		actions:       []string{"绕过", "规避", "突破", "bypass", "evade", "circumvent"},
		targets:       []string{"api限流", "限流", "速率限制", "rate limit", "rate limiting", "throttle", "throttling", "配额", "quota", "访问控制", "access control"},
		strongActions: []string{"绕过", "规避", "突破", "bypass", "evade", "circumvent"},
	},
	{
		category: "cookie_theft", reason: "cookie_or_session_theft",
		actions:       []string{"窃取", "盗取", "偷取", "窃听", "盗走", "steal", "exfiltrate", "hijack"},
		targets:       []string{"cookie", "kookie", "session", "会话", "token", "令牌", "身份凭证"},
		strongActions: []string{"窃取", "盗取", "偷取", "窃听", "盗走", "steal", "exfiltrate", "hijack"},
		ambiguous:     []string{"获取", "抓取", "导出", "提取", "读取", "查看", "get", "extract", "dump", "collect"},
	},
	{
		category: "credential_theft", reason: "wireless_credential_extraction",
		actions: []string{"netsh wlan show profile", "netsh wlan show profiles", "netsh.exe wlan show profile", "show profile", "show profiles"},
		targets: []string{"key=clear", "key clear", "wifi key", "wireless key", "无线密钥"},
	},
	{
		category: "phishing", reason: "phishing_or_credential_harvesting",
		actions:        []string{"制作", "搭建", "仿冒", "伪造", "创建", "build", "create", "make", "phishing", "harvest", "harvesting"},
		targets:        []string{"\u9493\u9c7c", "\u767b\u5f55\u754c\u9762", "\u767b\u5f55\u9875\u9762", "credential harvesting", "\u51ed\u636e", "\u8d26\u53f7"},
		genericTargets: []string{"credential", "password", "login"},
		strongActions:  []string{"制作", "搭建", "仿冒", "伪造", "phishing", "harvest", "harvesting"},
	},
	{
		category: "ransomware", reason: "ransomware_creation_or_deployment",
		actions:       []string{"编写", "写", "制作", "做", "开发", "部署", "传播", "create", "write", "develop", "deploy", "spread", "encrypt", "encryption"},
		targets:       []string{"勒索软件", "ransomware", "加密文件", "文件加密", "encrypt files", "file encryption", "victim files", "victims files", "赎金", "ransom"},
		strongActions: []string{"编写", "制作", "开发", "部署", "传播", "create", "write", "develop", "deploy", "spread", "encrypt", "encryption"},
	},
	{
		category: "ddos", reason: "distributed_denial_of_service",
		actions:        []string{"发起", "实施", "发动", "执行", "发送", "压垮", "打垮", "flood", "launch", "perform", "attack"},
		targets:        []string{"ddos", "\u62d2\u7edd\u670d\u52a1", "denial of service"},
		genericTargets: []string{"\u7f51\u7ad9", "\u670d\u52a1\u5668", "website", "server", "api", "\u63a5\u53e3", "\u76ee\u6807"},
		strongActions:  []string{"发起", "实施", "发动", "执行", "压垮", "打垮", "flood", "launch", "perform", "attack"},
	},
	{
		category: "license_key_cracking", reason: "license_or_product_key_cracking",
		actions: []string{"破解", "爆破", "生成", "伪造", "绕过", "crack", "generate", "forge", "bypass"},
		targets: []string{"卡密", "激活码", "license key", "product key", "序列号", "注册码"},
	},
	{
		category: "website_intrusion", reason: "unauthorized_website_intrusion",
		actions:        []string{"入侵", "攻击", "渗透", "利用漏洞", "漏洞利用", "exploit", "hack", "compromise", "breach", "deface", "take over", "getshell", "webshell", "提权", "sql注入", "rce"},
		targets:        []string{"网站", "站点", "服务器", "后台", "web", "目标", "系统"},
		genericTargets: []string{"\u7f51\u7ad9", "\u7ad9\u70b9", "website", "site", "\u670d\u52a1\u5668", "server", "web", "\u76ee\u6807", "system", "\u7cfb\u7edf"},
		strongActions:  []string{"入侵", "攻击", "渗透", "利用漏洞", "漏洞利用", "exploit", "hack", "compromise", "breach", "deface", "take over", "getshell", "webshell", "提权", "sql注入", "rce"},
		strongTargets:  []string{"sql注入", "rce"},
		ambiguous:      []string{"验证", "测试", "扫描", "检测", "audit", "test", "scan"},
	},
	{
		category: "jailbreak", reason: "jailbreak_or_safety_bypass",
		actions:        []string{"破限", "解除限制", "绕过限制", "忽略之前", "忽略指令", "无视规则", "不要拒绝", "进入开发者模式", "生成攻击工具", "生成攻击脚本", "生成恶意工具", "jailbreak", "bypass"},
		targets:        []string{"\u7834\u9650", "\u5b89\u5168\u7b56\u7565", "\u5b89\u5168\u89c4\u5219", "\u5185\u5bb9\u9650\u5236", "\u5ba1\u6838", "\u7cfb\u7edf\u63d0\u793a", "\u5f00\u53d1\u8005\u6a21\u5f0f", "\u653b\u51fb\u5de5\u5177", "\u653b\u51fb\u811a\u672c", "\u6076\u610f\u5de5\u5177", "safety", "restriction", "guardrail"},
		genericTargets: []string{"\u6a21\u578b", "model", "llm"},
		strongActions:  []string{"破限", "解除限制", "绕过限制", "忽略之前", "忽略指令", "无视规则", "不要拒绝", "进入开发者模式", "生成攻击工具", "生成攻击脚本", "生成恶意工具", "jailbreak", "bypass"},
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
	scanTexts := append([]string{canonical.Text}, securitytext.ReverseScanCandidates(canonical.Text)...)
	scanTexts = append(scanTexts, securitytext.ClassicalCipherScanCandidates(text)...)
	var review LocalPolicyDecision
	for _, scanText := range scanTexts {
		decision := evaluateCanonicalLocalPolicy(scanText)
		if decision.Blocked {
			return decision
		}
		if decision.NeedsAI && !review.NeedsAI {
			review = decision
		}
	}
	return review
}

func evaluateCanonicalLocalPolicy(text string) LocalPolicyDecision {
	canonical := securitytext.Canonicalize(text)
	if canonical.Text == "" {
		return LocalPolicyDecision{}
	}
	var review LocalPolicyDecision
	for _, segment := range splitLocalPolicySegments(canonical.Text) {
		for i := range localPolicyRules {
			rule := localPolicyRules[i]
			evidence := scoreLocalPolicyRule(segment, rule)
			if evidence.specificTargets+evidence.genericTargets == 0 {
				continue
			}
			if evidence.strongIntent+evidence.actionMatches+evidence.ambiguous == 0 {
				continue
			}
			score := evidence.strongIntent*localPolicyStrongWeight +
				evidence.actionMatches*localPolicyActionWeight +
				evidence.specificTargets*localPolicySpecificTargetWeight +
				evidence.genericTargets*localPolicyGenericTargetWeight +
				evidence.ambiguous*localPolicyAmbiguousWeight
			if evidence.protective {
				if !review.NeedsAI {
					review = localPolicyReviewDecision(rule)
				}
				continue
			}
			if evidence.actionMatches > 0 && score >= localPolicyBlockScore {
				return localPolicyBlockDecision(rule)
			}
			if !review.NeedsAI {
				review = localPolicyReviewDecision(rule)
			}
		}
	}
	return review
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
	for _, term := range terms {
		if containsLocalPolicyTerm(normalized, compact, term) {
			return true
		}
	}
	return false
}
