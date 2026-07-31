package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
)

func ptr[T any](v T) *T { return &v }

func event(id int64, from *string, to string, productWorkID *int64) *catalogclient.ClaimEventFeedItem {
	return &catalogclient.ClaimEventFeedItem{
		ID: id, WorkID: 900 + id, FromState: from, ToState: to,
		ProductWorkID: productWorkID, Site: "kungal",
	}
}

// The local effect is decided by the DESTINATION state, and two states that
// look like they should tidy up deliberately do not.
func TestEffectOfTransition(t *testing.T) {
	gid := ptr(int64(4321))
	cases := []struct {
		name string
		ev   *catalogclient.ClaimEventFeedItem
		want claimEffect
	}{
		{"birth into live seeds the stub", event(1, nil, catalogclient.ClaimStateLive, gid), claimEffectSeedStub},
		{"approval seeds the stub", event(2, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateLive, gid), claimEffectSeedStub},
		{"ban drops the stub", event(3, ptr(catalogclient.ClaimStateLive), catalogclient.ClaimStateHidden, gid), claimEffectDropStub},
		{"submit remembers the submitter", event(4, ptr(catalogclient.ClaimStateDraft), catalogclient.ClaimStatePending, gid), claimEffectRememberSubmitter},
		// A withdrawal is reversible and the stub carries user content.
		{"withdrawal keeps the stub", event(5, ptr(catalogclient.ClaimStateLive), catalogclient.ClaimStateDraft, gid), claimEffectNone},
		// A declined entry was never publicly visible.
		{"decline does nothing", event(6, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateDeclined, gid), claimEffectNone},
		// Without an anchor there is no row in kungal's key space to touch, and
		// the work id must never stand in for a gid.
		{"no product anchor is inert", event(7, nil, catalogclient.ClaimStateLive, nil), claimEffectNone},
		{"zero product anchor is inert", event(8, nil, catalogclient.ClaimStateLive, ptr(int64(0))), claimEffectNone},
		{"an unknown state is reported, not guessed", event(9, nil, "archived", gid), claimEffectUnknownState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectOf(tc.ev); got != tc.want {
				t.Errorf("effectOf = %v, want %v", got, tc.want)
			}
		})
	}
}

// Only ONE of the two routes into `live` pays the +3 here; the other is paid in
// the request path, and matching on the destination alone would pay twice.
func TestOnlyApprovalAwardsFromTheFeed(t *testing.T) {
	gid := ptr(int64(11))
	if !isApproval(event(1, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateLive, gid)) {
		t.Error("pending → live is the approval route and must award")
	}
	if isApproval(event(2, ptr(catalogclient.ClaimStateDraft), catalogclient.ClaimStateLive, gid)) {
		t.Error("draft → live is the owner publishing; the request path already awarded it")
	}
	if isApproval(event(3, nil, catalogclient.ClaimStateLive, gid)) {
		t.Error("a claim born live has no submission to reward")
	}
	if isApproval(event(4, ptr(catalogclient.ClaimStatePending), catalogclient.ClaimStateDeclined, gid)) {
		t.Error("a decline is not an approval")
	}
}

// claimFeedStub serves the claim-event feed as a paginated, ascending,
// exclusive-cursor stream, recording every cursor and site it was asked for.
type claimFeedStub struct {
	mu    sync.Mutex
	total int64
	page  int
	sites []string
}

func (f *claimFeedStub) client(t *testing.T) *catalogclient.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		f.mu.Lock()
		f.sites = append(f.sites, r.URL.Query().Get("site"))
		f.mu.Unlock()

		items := []string{}
		for id := since + 1; id <= f.total && len(items) < f.page; id++ {
			items = append(items, fmt.Sprintf(
				`{"id":%d,"work_id":%d,"from_state":null,"to_state":"live","actor_uid":7,`+
					`"reason":null,"site":"kungal","product_work_id":%d,`+
					`"created_at":"2026-07-30T10:00:00Z"}`, id, 5000+id, 100+id))
		}
		next := since
		if n := len(items); n > 0 {
			next = since + int64(n)
		}
		_, _ = fmt.Fprintf(w, `{"code":0,"message":"成功","data":{"items":[%s],"next_since":%d}}`,
			strings.Join(items, ","), next)
	}))
	t.Cleanup(srv.Close)
	return catalogclient.New(catalogclient.Config{
		BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec",
	})
}

// A fresh cursor must land on the feed's CURRENT last id. Starting from 0 would
// replay the re-site backfill — one event per existing claim — and seed a local
// stub for the entire registry, which is the browse list's population.
func TestFeedHeadWalksToTheLastEvent(t *testing.T) {
	stub := &claimFeedStub{total: 250, page: 100}
	s := NewGalgameClaimEventSync(stub.client(t), nil, nil)
	s.batch = stub.page

	head, err := s.feedHead(t.Context())
	if err != nil {
		t.Fatalf("feedHead: %v", err)
	}
	if head != 250 {
		t.Errorf("head = %d, want 250 (the last id, not the first page's)", head)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	for _, site := range stub.sites {
		if site != claimSite {
			t.Fatalf("feed asked with site=%q, want %q — an unfiltered feed would "+
				"hand kungal another product's claims to seed stubs for", site, claimSite)
		}
	}
}

// An empty feed leaves the cursor at 0 rather than erroring, so the very first
// deploy (no transitions yet) seeds cleanly.
func TestClaimFeedHeadOnEmptyFeed(t *testing.T) {
	stub := &claimFeedStub{total: 0, page: 100}
	s := NewGalgameClaimEventSync(stub.client(t), nil, nil)
	s.batch = stub.page

	head, err := s.feedHead(t.Context())
	if err != nil {
		t.Fatalf("feedHead: %v", err)
	}
	if head != 0 {
		t.Errorf("head = %d, want 0", head)
	}
}

// The two cursor id spaces are disjoint small integers, so the keys must not be
// either. This is pinned because the failure mode is silent: pointing the new
// feed at the wiki cursor starts it at an arbitrary offset in a foreign
// sequence, and reusing the award key prefix lets wiki message 42 suppress the
// reward for claim event 42.
func TestClaimNamespacesAreDistinctFromTheWikiFeed(t *testing.T) {
	if strings.Contains(claimCursorKey, "wiki:") {
		t.Errorf("cursor key %q reuses the wiki namespace", claimCursorKey)
	}
	if claimCursorKey == "wiki:msg:cron:since" {
		t.Errorf("cursor key %q is the retired wiki message cursor", claimCursorKey)
	}
}
