package graph

import (
	"sort"
	"testing"
)

func TestTier_TrustRank(t *testing.T) {
	cases := []struct {
		tier Tier
		want int
	}{
		{TierSpecs, 0},
		{TierDocs, 1},
		{TierContext, 2},
		{TierNotes, 3},
		{"", len(TrustTierOrder)},
		{Tier("unknown"), len(TrustTierOrder)},
	}
	for _, tc := range cases {
		if g := tc.tier.TrustRank(); g != tc.want {
			t.Errorf("TrustRank(%q) = %d, want %d", tc.tier, g, tc.want)
		}
	}
}

func TestLessTrustTier_sortsSpecsBeforeNotes(t *testing.T) {
	tiers := []Tier{TierNotes, TierSpecs, TierContext, TierDocs}
	sort.Slice(tiers, func(i, j int) bool {
		return LessTrustTier(tiers[i], tiers[j])
	})
	want := []Tier{TierSpecs, TierDocs, TierContext, TierNotes}
	for i := range want {
		if tiers[i] != want[i] {
			t.Fatalf("sorted order = %#v, want %#v", tiers, want)
		}
	}
}

func TestLessTrustTier_unknownStable(t *testing.T) {
	a, b := Tier("unknown-a"), Tier("unknown-b")
	if !LessTrustTier(a, b) {
		t.Fatal("expected unknown-a < unknown-b lexicographically")
	}
	if LessTrustTier(b, a) {
		t.Fatal("expected asymmetric order for unknown tiers")
	}
}
