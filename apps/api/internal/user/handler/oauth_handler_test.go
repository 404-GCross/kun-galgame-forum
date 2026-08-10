package handler

import (
	"slices"
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
)

func TestEnrichProfileFromSessionSurfacesEffectiveRoles(t *testing.T) {
	p := &dto.UserProfile{ID: 7, Name: "kun", Roles: []string{"creator"}}
	u := &middleware.UserInfo{Sub: "uuid-7", Roles: []string{"creator", "moderator"}}

	enrichProfileFromSession(p, u)

	if want := []string{"creator", "moderator"}; !slices.Equal(p.Roles, want) {
		t.Fatalf("/auth/me roles = %v, want the session's effective set %v", p.Roles, want)
	}
	if p.Sub != "uuid-7" {
		t.Fatalf("Sub not filled from session: %q", p.Sub)
	}
}
