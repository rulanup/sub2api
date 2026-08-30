package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeLocalAuditPolicyAction(t *testing.T) {
	tests := map[string]string{
		"allow":   LocalAuditPolicyActionAllow,
		" ALLOW ": LocalAuditPolicyActionAllow,
		"review":  LocalAuditPolicyActionReview,
		"block":   LocalAuditPolicyActionBlock,
		"BLOCK":   LocalAuditPolicyActionBlock,
		"":        LocalAuditPolicyActionReview,
		"invalid": LocalAuditPolicyActionReview,
	}
	for input, expected := range tests {
		require.Equal(t, expected, NormalizeLocalAuditPolicyAction(input), input)
	}
}
