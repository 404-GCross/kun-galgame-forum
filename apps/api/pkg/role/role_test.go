package role

import (
	"slices"
	"testing"
)

func TestSiteModeratorModeratesButNotAdminister(t *testing.T) {
	eff := Union(nil, []string{"moderator"})
	if !CanModerate(eff) {
		t.Fatalf("site moderator should CanModerate; effective=%v", eff)
	}
	if CanAdminister(eff) {
		t.Fatalf("site moderator MUST NOT CanAdminister; effective=%v", eff)
	}
	if IsCreator(eff) {
		t.Fatalf("site moderator is not a creator; effective=%v", eff)
	}
}

func TestSiteCreatorIsOrthogonal(t *testing.T) {
	eff := Union([]string{}, []string{"creator"})
	if !IsCreator(eff) {
		t.Fatalf("site creator should IsCreator; effective=%v", eff)
	}
	if CanModerate(eff) || CanAdminister(eff) {
		t.Fatalf("site creator must have no management power; effective=%v", eff)
	}
}

func TestSiteCustomNameFailsClosed(t *testing.T) {
	eff := Union(nil, []string{"event_organizer"})
	if CanModerate(eff) || CanAdminister(eff) || IsCreator(eff) {
		t.Fatalf("unknown site role must grant nothing; effective=%v", eff)
	}
}

func TestUnionDedupAndImmutability(t *testing.T) {
	global := []string{"creator"}
	site := []string{"creator", "moderator"}
	eff := Union(global, site)

	if want := []string{"creator", "moderator"}; !slices.Equal(eff, want) {
		t.Fatalf("union = %v, want %v (dedup, order-stable)", eff, want)
	}
	if !IsCreator(eff) || !CanModerate(eff) {
		t.Fatalf("effective should be creator + moderator; effective=%v", eff)
	}
	if CanAdminister(eff) {
		t.Fatalf("no admin power may appear from a site grant; effective=%v", eff)
	}
	if !slices.Equal(global, []string{"creator"}) {
		t.Fatalf("Union mutated its global input: %v", global)
	}
}

func TestGlobalAdminUnaffectedByEmptySite(t *testing.T) {
	eff := Union([]string{"admin"}, nil)
	if !CanAdminister(eff) || !CanModerate(eff) {
		t.Fatalf("global admin should administer + moderate; effective=%v", eff)
	}
	if !slices.Equal(eff, []string{"admin"}) {
		t.Fatalf("union with empty site = %v, want [admin]", eff)
	}
}
