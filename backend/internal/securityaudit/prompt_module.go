package securityaudit

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

func ProvideCoordinator(legacy LegacyEngine, prompt PromptEngine, risk service.RiskScoreRouter, defaultAudit DefaultAuditGate) *Coordinator {
	coordinator := NewCoordinator(legacy, prompt)
	coordinator.SetRiskScoreRouter(risk)
	coordinator.SetDefaultAuditGate(defaultAudit)
	return coordinator
}

// NewGatedPromptErrorService 在标准构造之上挂载内置默认审查策略开关。
func NewGatedPromptErrorService(repo PromptErrorRepository, defaultAudit DefaultAuditGate) *PromptErrorService {
	svc := NewPromptErrorService(repo)
	svc.SetDefaultAuditGate(defaultAudit)
	return svc
}

var ProviderSet = wire.NewSet(
	NewPostgreSQLRepository,
	wire.Bind(new(JobRepository), new(*PostgreSQLRepository)),
	wire.Bind(new(EventRepository), new(*PostgreSQLRepository)),
	wire.Bind(new(PromptErrorRepository), new(*PostgreSQLRepository)),
	NewRedisPayloadStore,
	wire.Bind(new(PayloadStore), new(*RedisPayloadStore)),
	NewOpenAICompatibleScanner,
	wire.Bind(new(PromptScanner), new(*OpenAICompatibleScanner)),
	NewAtomicMetrics,
	wire.Bind(new(Metrics), new(*AtomicMetrics)),
	NewConfigManager,
	wire.Bind(new(ConfigStore), new(*ConfigManager)),
	NewPromptService,
	wire.Bind(new(PromptEngine), new(*PromptService)),
	wire.Bind(new(PromptAdminService), new(*PromptService)),
	wire.Bind(new(DefaultAuditGate), new(*service.ContentModerationService)),
	NewGatedPromptErrorService,
	NewPromptErrorAdminHandler,
	NewLegacyModerationAdapter,
	ProvideCoordinator,
	NewPromptAdminHandler,
)
