package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLotteryActivityHandlerRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLotteryActivityHandler(nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/activity/status", nil)

	handler.GetStatus(ctx)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestLotteryActivityHandlerRequiresIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLotteryActivityHandler(nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/activity/draw", nil)
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})

	handler.Draw(ctx)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "INVALID_IDEMPOTENCY_KEY")
}

func TestLotteryActivityHandlerRejectsInvalidHistoryLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLotteryActivityHandler(nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/activity/history?limit=101", nil)
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})

	handler.History(ctx)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
