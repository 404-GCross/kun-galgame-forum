package catalogclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEditRevisionsSince_RequestAndParse(t *testing.T) {
	var gotPath, gotSince, gotLimit, gotType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSince = r.URL.Query().Get("since")
		gotLimit = r.URL.Query().Get("limit")
		gotType = r.URL.Query().Get("entity_type")
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"items":[
			{"id":41,"entity_family":"galgame","entity_type":"galgame.game","entity_id":1207,
			 "seq":8,"action":1,"changed_fields":["galgame.game.name_ja_jp"],
			 "actor_uid":9,"amender_uid":null,"proposal_id":77,"site":"kungal",
			 "created_at":"2026-07-30T10:00:00Z"}],"next_since":41}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	page, err := c.EditRevisionsSince(context.Background(), 40, 1000, "galgame.game")
	if err != nil {
		t.Fatalf("EditRevisionsSince: %v", err)
	}

	if gotPath != "/api/v1/catalog/edit-revisions/feed" {
		t.Errorf("path = %q", gotPath)
	}
	if gotSince != "40" || gotLimit != "1000" || gotType != "galgame.game" {
		t.Errorf("query = since:%q limit:%q entity_type:%q", gotSince, gotLimit, gotType)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}
	got := page.Items[0]
	if got.ID != 41 || got.EntityID != 1207 || got.Seq != 8 || got.ActorUID != 9 {
		t.Errorf("item = %+v", got)
	}
	if got.Action != EditActionMerged {
		t.Errorf("action = %d, want merged", got.Action)
	}
	if page.NextSince != 41 {
		t.Errorf("next_since = %d, want 41", page.NextSince)
	}
}

func TestEditRevisionsSince_EmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"items":[],"next_since":12681}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	page, err := c.EditRevisionsSince(context.Background(), 12681, 1000, "galgame.game")
	if err != nil {
		t.Fatalf("EditRevisionsSince: %v", err)
	}
	if len(page.Items) != 0 || page.NextSince != 12681 {
		t.Errorf("page = %+v", page)
	}
}

func TestEditRevisionsSince_NotConfigured(t *testing.T) {
	if _, err := New(Config{}).EditRevisionsSince(context.Background(), 0, 10, ""); err == nil {
		t.Fatal("want an error from an unconfigured client")
	}
}
