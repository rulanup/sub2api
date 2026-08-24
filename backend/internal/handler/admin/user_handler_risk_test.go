package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeRiskProfileReader struct {
	profiles map[int64]service.UserRiskProfile
	err      error
	userIDs  []int64
}

func (f *fakeRiskProfileReader) GetForUsers(_ context.Context, userIDs []int64) (map[int64]service.UserRiskProfile, error) {
	f.userIDs = append([]int64(nil), userIDs...)
	return f.profiles, f.err
}

func TestAdminUserListIncludesRiskProfileFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminService := newStubAdminService()
	adminService.users = []service.User{{ID: 7, Email: "risk@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	riskReader := &fakeRiskProfileReader{profiles: map[int64]service.UserRiskProfile{
		7: {UserID: 7, Score: 72, Level: service.RiskLevelHigh, LastReasonCode: "prompt_guard_blocked"},
	}}
	handler := NewUserHandler(adminService, nil, nil, nil, nil, nil, nil)
	handler.SetUserRiskProfileReader(riskReader)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	handler.List(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, []int64{7}, riskReader.userIDs)
	var response struct {
		Data struct {
			Items []struct {
				RiskScore          int     `json:"risk_score"`
				RiskLevel          string  `json:"risk_level"`
				LastRiskReasonCode string  `json:"last_risk_reason_code"`
				LastRiskEventAt    *string `json:"last_risk_event_at"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, 72, response.Data.Items[0].RiskScore)
	require.Equal(t, "high", response.Data.Items[0].RiskLevel)
	require.Equal(t, "prompt_guard_blocked", response.Data.Items[0].LastRiskReasonCode)
}

func TestAdminUserListMissingOrUnavailableRiskProfileDoesNotBreakList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminService := newStubAdminService()
	adminService.users = []service.User{{ID: 8, Email: "risk-missing@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	riskReader := &fakeRiskProfileReader{err: context.DeadlineExceeded}
	handler := NewUserHandler(adminService, nil, nil, nil, nil, nil, nil)
	handler.SetUserRiskProfileReader(riskReader)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	handler.List(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data struct {
			Items []struct {
				RiskScore int    `json:"risk_score"`
				RiskLevel string `json:"risk_level"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Data.Items[0].RiskScore)
	require.Equal(t, "low", response.Data.Items[0].RiskLevel)
}

func TestAdminUserDetailIncludesRiskProfileFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminService := newStubAdminService()
	adminService.users = []service.User{{ID: 9, Email: "risk-detail@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	riskReader := &fakeRiskProfileReader{profiles: map[int64]service.UserRiskProfile{
		9: {UserID: 9, Score: 91, Level: service.RiskLevelCritical, LastReasonCode: "local_cookie_theft"},
	}}
	handler := NewUserHandler(adminService, nil, nil, nil, nil, nil, nil)
	handler.SetUserRiskProfileReader(riskReader)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "9"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/9", nil)
	handler.GetByID(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data struct {
			RiskScore int    `json:"risk_score"`
			RiskLevel string `json:"risk_level"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 91, response.Data.RiskScore)
	require.Equal(t, "critical", response.Data.RiskLevel)
}
