package securityaudit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakePromptErrorRepo struct {
	PromptErrorRepository
	inserted int
}

func (f *fakePromptErrorRepo) InsertPromptError(context.Context, *PromptErrorRecord) error {
	f.inserted++
	return nil
}

func TestPromptErrorServiceDefaultAuditGateControlsRecording(t *testing.T) {
	repo := &fakePromptErrorRepo{}
	svc := NewPromptErrorService(repo)
	gate := &fakeDefaultAuditGate{enabled: true}
	svc.SetDefaultAuditGate(gate)

	require.NoError(t, svc.RecordUpstreamError(context.Background(), localPolicyRequest("记录这条提示词"), 502, "bad gateway"))
	require.Equal(t, 1, repo.inserted)

	gate.enabled = false
	require.NoError(t, svc.RecordUpstreamError(context.Background(), localPolicyRequest("这条不应落库"), 502, "bad gateway"))
	require.Equal(t, 1, repo.inserted)
}

func TestPromptErrorServiceNilGateKeepsRecording(t *testing.T) {
	repo := &fakePromptErrorRepo{}
	svc := NewPromptErrorService(repo)

	require.NoError(t, svc.RecordUpstreamError(context.Background(), localPolicyRequest("未挂开关仍记录"), 502, "bad gateway"))
	require.Equal(t, 1, repo.inserted)
}
