package handler

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelResponseCaptureWriterCapsWithoutChangingClientBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	base, _ := gin.CreateTestContext(recorder)
	capture := &modelResponseCaptureWriter{ResponseWriter: base.Writer}
	body := bytes.Repeat([]byte("r"), securityaudit.MaxModelResponseBytes+23)

	written, err := capture.Write(body)
	require.NoError(t, err)
	require.Equal(t, len(body), written)
	require.Equal(t, body, recorder.Body.Bytes())
	require.Len(t, capture.body, securityaudit.MaxModelResponseBytes)
	require.True(t, capture.truncated)
}
