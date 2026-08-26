package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	promptErrorBodyKey     = "prompt_error.body"
	promptErrorAPIKeyKey   = "prompt_error.api_key"
	promptErrorSubjectKey  = "prompt_error.subject"
	promptErrorProtocolKey = "prompt_error.protocol"
	promptErrorModelKey    = "prompt_error.model"
)

func storePromptErrorContext(c *gin.Context, body []byte, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string) {
	if c == nil {
		return
	}
	if len(body) > 0 {
		bCopy := make([]byte, len(body))
		copy(bCopy, body)
		c.Set(promptErrorBodyKey, bCopy)
	}
	if apiKey != nil {
		c.Set(promptErrorAPIKeyKey, apiKey)
	}
	c.Set(promptErrorSubjectKey, subject)
	if protocol != "" {
		c.Set(promptErrorProtocolKey, protocol)
	}
	if model != "" {
		c.Set(promptErrorModelKey, model)
	}
}

func tryRecordPromptErrorFromContext(c *gin.Context, svc *securityaudit.PromptErrorService, errorStatus int, errorBody string) {
	if c == nil || svc == nil {
		return
	}
	bodyVal, ok := c.Get(promptErrorBodyKey)
	if !ok {
		return
	}
	body, _ := bodyVal.([]byte)
	if len(body) == 0 {
		return
	}
	var apiKey *service.APIKey
	if v, ok := c.Get(promptErrorAPIKeyKey); ok {
		apiKey, _ = v.(*service.APIKey)
	}
	var subject middleware2.AuthSubject
	if v, ok := c.Get(promptErrorSubjectKey); ok {
		subject, _ = v.(middleware2.AuthSubject)
	}
	protocol := c.GetString(promptErrorProtocolKey)
	model := c.GetString(promptErrorModelKey)
	recordPromptErrorAsync(c, svc, apiKey, subject, protocol, model, body, errorStatus, errorBody)
}

// recordPromptErrorAsync persists the full prompt that triggered an upstream error.
// It runs asynchronously to avoid blocking the gateway hot path.
func recordPromptErrorAsync(
	c *gin.Context,
	svc *securityaudit.PromptErrorService,
	apiKey *service.APIKey,
	subject middleware2.AuthSubject,
	protocol, model string,
	body []byte,
	errorStatus int,
	errorBody string,
) {
	if svc == nil || c == nil || len(body) == 0 {
		return
	}
	requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
	// Build securityaudit.Request similar to buildSecurityAuditRequest but without extra context lookups.
	legacy := buildContentModerationInput(c, apiKey, subject, protocol, model, body)
	req := securityaudit.Request{
		RequestID:  requestID,
		UserID:     legacy.UserID,
		UserEmail:  legacy.UserEmail,
		APIKeyID:   legacy.APIKeyID,
		APIKeyName: legacy.APIKeyName,
		GroupID:    cloneSecurityAuditGroupID(legacy.GroupID),
		GroupName:  legacy.GroupName,
		Provider:   legacy.Provider,
		Endpoint:   legacy.Endpoint,
		Protocol:   legacy.Protocol,
		Model:      legacy.Model,
		Body:       body,
		Stage:      "http",
	}
	if apiKey != nil && apiKey.User != nil {
		req.Username = apiKey.User.Username
		if req.UserEmail == "" {
			req.UserEmail = apiKey.User.Email
		}
	}
	// Clone body to avoid data race after caller returns.
	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)
	req.Body = bodyCopy
	// Fire-and-forget with background context (gin context may be cancelled after response).
	go func() {
		ctx := context.Background()
		_ = svc.RecordUpstreamError(ctx, req, errorStatus, errorBody)
	}()
}
