package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newModelsListContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	return c, rec
}

func decodeModelItems(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var body struct {
		Object string           `json:"object"`
		Data   []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "list", body.Object)
	return body.Data
}

func TestWriteOpenAIModelsListWithInfoAnnotatesImageModels(t *testing.T) {
	c, rec := newModelsListContext()
	writeOpenAIModelsListWithInfo(c, []string{"gpt-image-2", "gpt-5.6"}, &modelImageInfo{
		SizeTiers: []string{"1K", "2K"},
		Sizes:     []string{"1024x1024", "1536x1024"},
	})

	data := decodeModelItems(t, rec)
	require.Len(t, data, 2)
	require.Equal(t, "image", data[0]["capability"])
	require.Equal(t, []any{"images"}, data[0]["protocols"])
	require.Equal(t, map[string]any{
		"size_tiers": []any{"1K", "2K"},
		"sizes":      []any{"1024x1024", "1536x1024"},
	}, data[0]["image"])

	// 文本模型声明 text 能力与可用协议，但不得带生图参数块。
	require.Equal(t, "text", data[1]["capability"])
	require.Equal(t, []any{"responses", "chat"}, data[1]["protocols"])
	require.NotContains(t, data[1], "image")
}

// 审查重点 #2：未知模型不得输出 capability。
func TestWriteOpenAIModelsListWithInfoUnknownModelHasNoCapability(t *testing.T) {
	c, rec := newModelsListContext()
	writeOpenAIModelsListWithInfo(c, []string{"my-private-alias"}, nil)

	data := decodeModelItems(t, rec)
	require.Len(t, data, 1)
	require.NotContains(t, data[0], "capability")
	require.NotContains(t, data[0], "protocols")
	require.NotContains(t, data[0], "image")
}

// 无分组档位时仍输出 capability，但不输出 image 块。
func TestWriteOpenAIModelsListWithInfoOmitsImageWithoutTiers(t *testing.T) {
	c, rec := newModelsListContext()
	writeOpenAIModelsListWithInfo(c, []string{"gpt-image-2"}, nil)

	data := decodeModelItems(t, rec)
	require.Equal(t, "image", data[0]["capability"])
	require.NotContains(t, data[0], "image")
}

// 审查重点 #5：Claude 形态响应不得因本次改动出现新字段。
func TestWriteModelsListClaudeShapeUnchanged(t *testing.T) {
	c, rec := newModelsListContext()
	writeModelsList(c, service.PlatformAnthropic, []string{"claude-sonnet-4-5"})

	require.NotContains(t, rec.Body.String(), "capability")
	require.NotContains(t, rec.Body.String(), "protocols")
	require.NotContains(t, rec.Body.String(), "size_tiers")
}
