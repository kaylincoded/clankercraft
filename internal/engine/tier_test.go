package engine

import "testing"

func TestTierString(t *testing.T) {
	tests := []struct {
		tier Tier
		want string
	}{
		{TierUnknown, "unknown"},
		{TierVanilla, "vanilla"},
		{TierWorldEdit, "worldedit"},
		{TierFAWE, "fawe"},
	}
	for _, tt := range tests {
		if got := tt.tier.String(); got != tt.want {
			t.Errorf("Tier(%d).String() = %q, want %q", tt.tier, got, tt.want)
		}
	}
}
