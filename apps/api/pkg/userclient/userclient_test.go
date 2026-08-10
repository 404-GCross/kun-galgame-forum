package userclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestUserBriefFoldsSiteRolesIntoRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","data":{"users":[` +
			`{"id":7,"uuid":"u-7","name":"kun","roles":["creator"],"site_roles":["moderator"]}` +
			`],"not_found":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})

	u, ok, err := c.User(context.Background(), 7)
	if err != nil || !ok {
		t.Fatalf("User() ok=%v err=%v", ok, err)
	}
	if want := []string{"creator", "moderator"}; !slices.Equal(u.Roles, want) {
		t.Fatalf("brief Roles = %v, want the effective union %v", u.Roles, want)
	}
	if want := []string{"moderator"}; !slices.Equal(u.SiteRoles, want) {
		t.Fatalf("brief SiteRoles = %v, want %v", u.SiteRoles, want)
	}
}

func TestUserBriefNoSiteRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","data":{"users":[` +
			`{"id":8,"uuid":"u-8","name":"ren","roles":["admin"]}` +
			`],"not_found":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})
	u, _, _ := c.User(context.Background(), 8)
	if want := []string{"admin"}; !slices.Equal(u.Roles, want) {
		t.Fatalf("brief Roles = %v, want %v (no site grant → unchanged)", u.Roles, want)
	}
}
