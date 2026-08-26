package securityaudit

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type PromptErrorAdminHandler struct {
	service *PromptErrorService
}

func NewPromptErrorAdminHandler(service *PromptErrorService) *PromptErrorAdminHandler {
	return &PromptErrorAdminHandler{service: service}
}

func (h *PromptErrorAdminHandler) List(c *gin.Context) {
	page, err := positiveIntQuery(c, "page", 1, 0)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	pageSize, err := positiveIntQuery(c, "page_size", 20, 100)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	filter, err := promptErrorFilterFromQuery(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.service.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *PromptErrorAdminHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_invalid_id", "记录 ID 无效"))
		return
	}
	rec, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if err == ErrPromptErrorNotFound {
			response.ErrorFrom(c, infraerrors.NotFound("prompt_error_not_found", "错误提示词记录不存在"))
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rec)
}

func (h *PromptErrorAdminHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_invalid_id", "记录 ID 无效"))
		return
	}
	count, err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"result": "success", "deleted": count, "id": id})
	response.Success(c, gin.H{"deleted": count})
}

func (h *PromptErrorAdminHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 || len(req.IDs) > 500 {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_invalid_batch", "批量删除必须包含 1-500 个 ID"))
		return
	}
	for _, id := range req.IDs {
		if id <= 0 {
			response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_invalid_id", "记录 ID 无效"))
			return
		}
	}
	count, err := h.service.DeleteByIDs(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"result": "success", "deleted": count, "requested": len(req.IDs)})
	response.Success(c, gin.H{"deleted": count})
}

func (h *PromptErrorAdminHandler) DeletePreview(c *gin.Context) {
	var filter PromptErrorFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_delete_preview_invalid", "删除预览筛选无效"))
		return
	}
	preview, err := h.service.PreviewDelete(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_delete_preview_invalid", err.Error()))
		return
	}
	response.Success(c, preview)
}

func (h *PromptErrorAdminHandler) DeleteByFilter(c *gin.Context) {
	var req PromptErrorDeleteByFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_delete_confirmation_invalid", "删除确认无效或已过期"))
		return
	}
	count, err := h.service.DeleteByFilter(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_error_delete_confirmation_invalid", err.Error()))
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"result": "success", "deleted": count})
	response.Success(c, gin.H{"deleted": count})
}

func (h *PromptErrorAdminHandler) ExportCSV(c *gin.Context) {
	filter, err := promptErrorFilterFromQuery(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	records, err := h.service.ListForExport(c.Request.Context(), filter, 10000)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var buf bytes.Buffer
	// BOM for Excel compatibility
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"id", "request_id", "username", "user_email", "api_key_name", "group_name", "provider", "endpoint", "protocol", "model", "prompt_hash", "prompt_length", "message_count", "error_status", "error_type", "error_body", "full_prompt", "created_at"})
	for _, r := range records {
		uid := ""
		if r.UserID != nil {
			uid = fmt.Sprintf("%d", *r.UserID)
		}
		gid := ""
		if r.GroupID != nil {
			gid = fmt.Sprintf("%d", *r.GroupID)
		}
		// Keep CSV fields single-line: replace newlines
		prompt := strings.ReplaceAll(r.FullPrompt, "\r", " ")
		prompt = strings.ReplaceAll(prompt, "\n", " ")
		errBody := strings.ReplaceAll(r.ErrorBody, "\r", " ")
		errBody = strings.ReplaceAll(errBody, "\n", " ")
		_ = writer.Write([]string{
			fmt.Sprintf("%d", r.ID),
			r.RequestID,
			uid + ":" + r.UsernameSnapshot,
			r.UserEmailSnapshot,
			r.APIKeyNameSnapshot,
			gid + ":" + r.GroupName,
			r.Provider,
			r.Endpoint,
			r.Protocol,
			r.Model,
			r.PromptHash,
			fmt.Sprintf("%d", r.PromptLength),
			fmt.Sprintf("%d", r.MessageCount),
			fmt.Sprintf("%d", r.ErrorStatus),
			r.ErrorType,
			errBody,
			prompt,
			r.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		response.InternalError(c, "导出失败: "+err.Error())
		return
	}
	filename := fmt.Sprintf("prompt-error-records-%s.csv", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))
	_, _ = c.Writer.Write(buf.Bytes())
}

func promptErrorFilterFromQuery(c *gin.Context) (PromptErrorFilter, error) {
	groupID, err := optionalPositiveInt64Query(c, "group_id")
	if err != nil {
		return PromptErrorFilter{}, err
	}
	userID, err := optionalPositiveInt64Query(c, "user_id")
	if err != nil {
		return PromptErrorFilter{}, err
	}
	apiKeyID, err := optionalPositiveInt64Query(c, "api_key_id")
	if err != nil {
		return PromptErrorFilter{}, err
	}
	filter := PromptErrorFilter{
		Keyword:    c.Query("keyword"),
		Model:      c.Query("model"),
		GroupID:    groupID,
		UserID:     userID,
		APIKeyID:   apiKeyID,
		RequestID:  c.Query("request_id"),
		PromptHash: c.Query("prompt_hash"),
	}
	if v := strings.TrimSpace(c.Query("error_status")); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return PromptErrorFilter{}, infraerrors.BadRequest("prompt_error_invalid_filter", "error_status 无效")
		}
		filter.ErrorStatus = &parsed
	}
	if value := strings.TrimSpace(c.Query("start_at")); value != "" {
		filter.StartAt = parseTimeQuery(value)
		if filter.StartAt == nil {
			return PromptErrorFilter{}, infraerrors.BadRequest("prompt_error_invalid_time", "开始时间无效")
		}
	}
	if value := strings.TrimSpace(c.Query("end_at")); value != "" {
		filter.EndAt = parseTimeQuery(value)
		if filter.EndAt == nil {
			return PromptErrorFilter{}, infraerrors.BadRequest("prompt_error_invalid_time", "结束时间无效")
		}
	}
	return filter, nil
}
