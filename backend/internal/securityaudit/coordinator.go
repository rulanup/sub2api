package securityaudit

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type LegacyEngine interface {
	Check(ctx context.Context, req Request) (*LegacyDecision, error)
}

type PromptEngine interface {
	EffectiveMode() Mode
	Enqueue(ctx context.Context, req Request) error
	Evaluate(ctx context.Context, req Request) (*PromptDecision, error)
}

type SystemPromptPolicy struct {
	Enabled bool
	Prompt  string
	Mode    string
}

type systemPromptPolicyProvider interface {
	SystemPromptPolicy(groupID *int64) SystemPromptPolicy
}

// DefaultAuditGate 报告内置默认审查策略（本地安全策略初筛、上游错误提示词记录）
// 是否启用。实现方基于 settings 表的 default_audit_policies_enabled 键；
// gate 为 nil 时按启用处理，保持既有行为。
type DefaultAuditGate interface {
	DefaultAuditPoliciesEnabled(ctx context.Context) bool
	LocalAuditPolicyAction(ctx context.Context) string
}

type Coordinator struct {
	legacy       LegacyEngine
	prompt       PromptEngine
	risk         service.RiskScoreRouter
	defaultAudit DefaultAuditGate
}

func NewCoordinator(legacy LegacyEngine, prompt PromptEngine) *Coordinator {
	return &Coordinator{legacy: legacy, prompt: prompt}
}

// SetRiskScoreRouter attaches the optional per-user routing policy while
// keeping the legacy constructor compatible with isolated handlers/tests.
func (c *Coordinator) SetRiskScoreRouter(router service.RiskScoreRouter) {
	if c != nil {
		c.risk = router
	}
}

// SetDefaultAuditGate attaches the switch that can disable the built-in
// always-on audit policies. nil gate keeps stock behavior.
func (c *Coordinator) SetDefaultAuditGate(gate DefaultAuditGate) {
	if c != nil {
		c.defaultAudit = gate
	}
}

func (c *Coordinator) defaultAuditPoliciesEnabled(ctx context.Context) bool {
	return c == nil || c.defaultAudit == nil || c.defaultAudit.DefaultAuditPoliciesEnabled(ctx)
}

func (c *Coordinator) localAuditPolicyAction(ctx context.Context) string {
	if c == nil || c.defaultAudit == nil {
		return service.LocalAuditPolicyActionReview
	}
	return service.NormalizeLocalAuditPolicyAction(c.defaultAudit.LocalAuditPolicyAction(ctx))
}

func (c *Coordinator) SystemPromptPolicy(groupID *int64) SystemPromptPolicy {
	if c == nil || c.prompt == nil {
		return SystemPromptPolicy{}
	}
	provider, ok := c.prompt.(systemPromptPolicyProvider)
	if !ok {
		return SystemPromptPolicy{}
	}
	return provider.SystemPromptPolicy(groupID)
}

func (c *Coordinator) ModelResponseRetentionEnabled(groupID *int64) bool {
	if c == nil || c.prompt == nil {
		return false
	}
	provider, ok := c.prompt.(modelResponseRetentionProvider)
	return ok && provider.ModelResponseRetentionEnabled(groupID)
}

func (c *Coordinator) StoreModelResponse(ctx context.Context, response ModelResponse) error {
	if c == nil || c.prompt == nil {
		return nil
	}
	provider, ok := c.prompt.(modelResponseRetentionProvider)
	if !ok {
		return nil
	}
	return provider.StoreModelResponse(ctx, response)
}

func (c *Coordinator) Check(ctx context.Context, req Request) Decision {
	if c == nil {
		return allowDecision(nil, nil)
	}
	// 内置默认审查策略关闭时，本地筛查与风险评分路由一并跳过，
	// 仅保留显式配置的内容审核与提示词审查链路。
	if !c.defaultAuditPoliciesEnabled(ctx) {
		return c.checkConfigured(ctx, req)
	}
	local := EvaluateLocalPolicy(req)
	if local.Blocked || local.NeedsAI {
		switch c.localAuditPolicyAction(ctx) {
		case service.LocalAuditPolicyActionAllow:
			return allowDecision(nil, nil)
		case service.LocalAuditPolicyActionBlock:
			code := local.ErrorCode
			if code == "" {
				code = ErrorCodeNetworkSecurityPolicyViolation
			}
			decision := Decision{
				Kind:           DecisionBlock,
				HTTPStatus:     http.StatusForbidden,
				ErrorCode:      code,
				ClientMessage:  "请求被本地安全策略拦截",
				AllowNextStage: false,
			}
			c.recordRisk(ctx, req, decision, &local)
			return decision
		default:
			// review：本地策略只做初筛，交由审查模型裁决；未配置模型时放行。
			return c.checkSuspicious(ctx, req)
		}
	}
	if c.risk != nil {
		promptConfigured := c.prompt != nil && c.prompt.EffectiveMode() != ModeOff
		route, err := c.risk.Route(ctx, req.UserID, local.NeedsAI, promptConfigured)
		if err != nil {
			c.logRiskScoreStoreFailure(req, "route", err)
		} else {
			if route.SkipExternal {
				return allowDecision(nil, nil)
			}
			return c.checkRouted(ctx, req, route)
		}
	}
	return c.checkConfigured(ctx, req)
}

// checkSuspicious 处理本地策略初筛命中的请求：拦截与否完全由审查模型裁决，
// 本地判定不再直接产生 403。审查模型按其运行模式参与：
//   - ModeBlocking：同步送审，模型拦截才拦截；模型不可用时 prioritize 按放行处理；
//   - ModeAsync：异步送审记录，当前请求放行；
//   - ModeOff（未配置审查模型）：直接跳过，仅保留显式配置的内容审核链路。
func (c *Coordinator) checkSuspicious(ctx context.Context, req Request) Decision {
	mode := ModeOff
	if c.prompt != nil {
		mode = c.prompt.EffectiveMode()
	}
	switch mode {
	case ModeBlocking:
		decision := c.checkBlocking(ctx, req)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	case ModeAsync:
		_ = c.prompt.Enqueue(ctx, req.Clone())
		legacy, _ := c.checkLegacy(ctx, req)
		decision := prioritize(legacy, nil)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	default:
		legacy, _ := c.checkLegacy(ctx, req)
		decision := prioritize(legacy, nil)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	}
}

func (c *Coordinator) checkConfigured(ctx context.Context, req Request) Decision {
	mode := ModeOff
	if c.prompt != nil {
		mode = c.prompt.EffectiveMode()
	}
	switch mode {
	case ModeAsync:
		// Enqueue is deliberately best-effort. The implementation owns a bounded
		// context and copies request memory before it can outlive the Handler.
		_ = c.prompt.Enqueue(ctx, req.Clone())
		legacy, _ := c.checkLegacy(ctx, req)
		decision := prioritize(legacy, nil)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	case ModeBlocking:
		decision := c.checkBlocking(ctx, req)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	default:
		legacy, _ := c.checkLegacy(ctx, req)
		decision := prioritize(legacy, nil)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	}
}

func (c *Coordinator) checkRouted(ctx context.Context, req Request, route service.RiskRoute) Decision {
	mode := ModeOff
	if c.prompt != nil {
		mode = c.prompt.EffectiveMode()
	}
	if route.RunPromptAudit && mode == ModeAsync {
		_ = c.prompt.Enqueue(ctx, req.Clone())
		var legacy *LegacyDecision
		if route.RunModeration {
			legacy, _ = c.checkLegacy(ctx, req)
		}
		decision := prioritize(legacy, nil)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	}
	if route.RunPromptAudit && mode == ModeBlocking {
		decision := c.checkBlockingStages(ctx, req, route.RunModeration, true)
		c.recordRisk(ctx, req, decision, nil)
		return decision
	}
	var legacy *LegacyDecision
	if route.RunModeration {
		legacy, _ = c.checkLegacy(ctx, req)
	}
	decision := prioritize(legacy, nil)
	c.recordRisk(ctx, req, decision, nil)
	return decision
}

func (c *Coordinator) checkBlocking(ctx context.Context, req Request) Decision {
	return c.checkBlockingStages(ctx, req, true, true)
}

func (c *Coordinator) checkBlockingStages(ctx context.Context, req Request, runLegacy, runPrompt bool) Decision {
	var wg sync.WaitGroup
	workers := 0
	if runLegacy {
		workers++
	}
	if runPrompt {
		workers++
	}
	if workers == 0 {
		return allowDecision(nil, nil)
	}
	wg.Add(workers)
	var legacy *LegacyDecision
	var prompt *PromptDecision
	if runLegacy {
		go func() {
			defer wg.Done()
			legacy, _ = c.checkLegacy(ctx, req)
		}()
	}
	if runPrompt {
		go func() {
			defer wg.Done()
			prompt = c.evaluatePrompt(ctx, req)
		}()
	}
	wg.Wait()
	return prioritize(legacy, prompt)
}

func (c *Coordinator) evaluatePrompt(ctx context.Context, req Request) *PromptDecision {
	if c.prompt == nil {
		return unavailablePromptDecision(ErrorCodeUnavailable)
	}
	result, err := c.prompt.Evaluate(ctx, req.Clone())
	if err != nil {
		var guardErr *GuardError
		if errors.As(err, &guardErr) && guardErr.Code == ErrorCodeInvalidResponse {
			return unavailablePromptDecision(ErrorCodeInvalidResponse)
		}
		return unavailablePromptDecision(ErrorCodeUnavailable)
	}
	if result == nil {
		return unavailablePromptDecision(ErrorCodeUnavailable)
	}
	return result
}

func (c *Coordinator) checkLegacy(ctx context.Context, req Request) (*LegacyDecision, error) {
	if c.legacy == nil {
		return nil, nil
	}
	return c.legacy.Check(ctx, req)
}

func prioritize(legacy *LegacyDecision, prompt *PromptDecision) Decision {
	if legacy != nil && legacy.Blocked {
		status := legacy.StatusCode
		if status < 400 || status > 599 {
			status = http.StatusForbidden
		}
		code := legacy.ErrorCode
		if code == "" {
			code = "content_policy_violation"
		}
		return Decision{
			Kind: DecisionBlock, HTTPStatus: status, ErrorCode: code, ClientMessage: legacy.Message,
			Legacy: legacy, Prompt: prompt, AllowNextStage: false,
		}
	}
	if prompt == nil {
		return allowDecision(legacy, nil)
	}
	switch prompt.Kind {
	case DecisionBlock:
		return Decision{Kind: DecisionBlock, HTTPStatus: http.StatusForbidden, ErrorCode: ErrorCodeBlocked,
			ClientMessage: "提示词安全审计拒绝了该请求，请调整输入后重试", Legacy: legacy, Prompt: prompt}
	case DecisionInvalid:
		return Decision{Kind: DecisionInvalid, HTTPStatus: http.StatusServiceUnavailable, ErrorCode: ErrorCodeInvalidResponse,
			ClientMessage: "提示词安全审计暂时不可用，已放行当前请求", Legacy: legacy, Prompt: prompt, AllowNextStage: true}
	case DecisionUnavailable:
		return Decision{Kind: DecisionUnavailable, HTTPStatus: http.StatusServiceUnavailable, ErrorCode: ErrorCodeUnavailable,
			ClientMessage: "提示词安全审计暂时不可用，已放行当前请求", Legacy: legacy, Prompt: prompt, AllowNextStage: true}
	case DecisionFlag:
		return Decision{Kind: DecisionFlag, HTTPStatus: http.StatusOK, Legacy: legacy, Prompt: prompt, AllowNextStage: true}
	default:
		return allowDecision(legacy, prompt)
	}
}

func allowDecision(legacy *LegacyDecision, prompt *PromptDecision) Decision {
	return Decision{Kind: DecisionAllow, HTTPStatus: http.StatusOK, Legacy: legacy, Prompt: prompt, AllowNextStage: true}
}

func unavailablePromptDecision(code string) *PromptDecision {
	kind := DecisionUnavailable
	if code == ErrorCodeInvalidResponse {
		kind = DecisionInvalid
	}
	return &PromptDecision{Kind: kind, ErrorCode: code, AllowNextStage: true}
}

func (c *Coordinator) recordRisk(ctx context.Context, req Request, decision Decision, local *LocalPolicyDecision) {
	if c == nil || c.risk == nil {
		return
	}
	if local != nil && local.Blocked {
		c.recordRiskEvent(ctx, req, service.RiskEvent{
			ReasonCode: local.ReasonCode,
			Delta:      50,
			DedupeKey:  riskDedupeKey(req, local.ReasonCode),
		})
		return
	}
	event := service.RiskEvent{}
	switch {
	case decision.Kind == DecisionBlock && decision.Legacy != nil:
		event = service.RiskEvent{ReasonCode: "content_moderation_blocked", Delta: 30}
	case decision.Kind == DecisionBlock && decision.Prompt != nil:
		event = service.RiskEvent{ReasonCode: "prompt_guard_blocked", Delta: 35}
	case decision.Legacy != nil && decision.Legacy.Flagged:
		event = service.RiskEvent{ReasonCode: "content_moderation_flagged", Delta: 20}
	case decision.Kind == DecisionFlag && decision.Prompt != nil:
		event = service.RiskEvent{ReasonCode: "prompt_guard_flagged", Delta: 20}
	default:
		return
	}
	event.DedupeKey = riskDedupeKey(req, event.ReasonCode)
	c.recordRiskEvent(ctx, req, event)
}

func (c *Coordinator) recordRiskEvent(ctx context.Context, req Request, event service.RiskEvent) {
	if c == nil || c.risk == nil {
		return
	}
	if err := c.risk.Record(ctx, req.UserID, event); err != nil {
		c.logRiskScoreStoreFailure(req, "record", err)
	}
}

func (c *Coordinator) logRiskScoreStoreFailure(req Request, action string, err error) {
	if err == nil {
		return
	}
	fields := requestLogFields(req)
	fields["action"] = "risk_score_" + action
	fields["error_code"] = err.Error()
	LogWarn(EventRiskScoreStoreFailed, fields)
}

func riskDedupeKey(req Request, reasonCode string) string {
	identity := req.RequestID
	if identity == "" {
		digest := sha256.Sum256(req.Body)
		identity = fmt.Sprintf("body-%x", digest[:8])
	}
	return fmt.Sprintf("%s:%s:%s:%s", identity, req.Stage, req.Protocol, reasonCode)
}
