package securityaudit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoundModelResponse(t *testing.T) {
	body := bytes.Repeat([]byte("x"), MaxModelResponseBytes+17)
	bounded, truncated := BoundModelResponse(body, false)
	require.Len(t, bounded, MaxModelResponseBytes)
	require.True(t, truncated)
	require.Equal(t, body[:MaxModelResponseBytes], bounded)

	small, truncated := BoundModelResponse([]byte("complete"), false)
	require.Equal(t, []byte("complete"), small)
	require.False(t, truncated)
}

func TestEventListColumnsNeverSelectModelResponse(t *testing.T) {
	columns := eventColumns("e")
	require.NotContains(t, columns, "model_response")
	require.NotContains(t, columns, "response_body")
	require.NotContains(t, strings.ToLower(columns), "prompt_audit_model_responses")
	event := &Event{modelResponse: "sensitive raw response", responseTruncated: true}
	raw, err := json.Marshal(event)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "sensitive raw response")
	require.NotContains(t, string(raw), "model_response")
}
