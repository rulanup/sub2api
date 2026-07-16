package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type adminWhitelistRuleRepo struct{}

func (adminWhitelistRuleRepo) List(context.Context) ([]*model.ErrorPassthroughRule, error) {
	return nil, nil
}
func (adminWhitelistRuleRepo) GetByID(context.Context, int64) (*model.ErrorPassthroughRule, error) {
	return nil, nil
}
func (adminWhitelistRuleRepo) Create(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (adminWhitelistRuleRepo) Update(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (adminWhitelistRuleRepo) Delete(context.Context, int64) error { return nil }

type adminWhitelistSettingRepo struct {
	value  string
	getErr error
}

func (r *adminWhitelistSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.value == "" {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Value: r.value}, nil
}

func (r *adminWhitelistSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}
func (r *adminWhitelistSettingRepo) Set(_ context.Context, _, value string) error {
	r.value = value
	return nil
}
func (*adminWhitelistSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (*adminWhitelistSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (*adminWhitelistSettingRepo) GetAll(context.Context) (map[string]string, error)    { return nil, nil }
func (*adminWhitelistSettingRepo) Delete(context.Context, string) error                 { return nil }

func newAdminWhitelistHandler(value string) (*ErrorPassthroughHandler, *adminWhitelistSettingRepo) {
	repo := &adminWhitelistSettingRepo{value: value}
	svc := service.NewErrorPassthroughService(adminWhitelistRuleRepo{}, nil)
	svc.SetSettingRepository(repo)
	return NewErrorPassthroughHandler(svc), repo
}

func TestErrorPassthroughWhitelistGetConvergesAfterMissedNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repo := newAdminWhitelistHandler(`[7]`)
	repo.value = `[11,4,11]`

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/error-passthrough-rules/whitelist", nil)
	h.GetWhitelist(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, `[4,11]`, gjson.GetBytes(rec.Body.Bytes(), "data.user_ids").Raw)
}

func TestErrorPassthroughWhitelistGetReturnsReadError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repo := newAdminWhitelistHandler(`[7]`)
	repo.getErr = errors.New("database unavailable")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/error-passthrough-rules/whitelist", nil)
	h.GetWhitelist(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Empty(t, gjson.GetBytes(rec.Body.Bytes(), "data.user_ids").Array())
}

func TestErrorPassthroughWhitelistGetAndPut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repo := newAdminWhitelistHandler(`[8,3,8]`)

	getRec := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRec)
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/error-passthrough-rules/whitelist", nil)
	h.GetWhitelist(getCtx)
	require.Equal(t, http.StatusOK, getRec.Code)
	require.Equal(t, `[3,8]`, gjson.GetBytes(getRec.Body.Bytes(), "data.user_ids").Raw)

	putRec := httptest.NewRecorder()
	putCtx, _ := gin.CreateTestContext(putRec)
	putCtx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/error-passthrough-rules/whitelist", strings.NewReader(`{"user_ids":[9,2,9]}`))
	putCtx.Request.Header.Set("Content-Type", "application/json")
	h.UpdateWhitelist(putCtx)
	require.Equal(t, http.StatusOK, putRec.Code)
	require.Equal(t, `[2,9]`, gjson.GetBytes(putRec.Body.Bytes(), "data.user_ids").Raw)
	require.Equal(t, `[2,9]`, repo.value)
}

func TestErrorPassthroughWhitelistPutRejectsNonPositiveIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repo := newAdminWhitelistHandler(`[7]`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/error-passthrough-rules/whitelist", strings.NewReader(`{"user_ids":[7,0]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateWhitelist(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, gjson.GetBytes(rec.Body.Bytes(), "message").String(), "user IDs must be positive")
	require.Equal(t, `[7]`, repo.value)
}
