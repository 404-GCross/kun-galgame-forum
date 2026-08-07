package catalogclient

// Contract tests for the READS wave 180 moved onto the user-token plane, plus
// the one asserted-actor write that came with them (the image upload).
//
// The axis is the same as the write lanes' and fails just as silently: a read
// that quietly kept the Basic credential still returns data — the right data,
// even — while answering "who is asking" with "the forum". What the tests pin
// is therefore the request, not the payload: which credential travels, which
// path, and above all which fields DO NOT travel. `site`, `proposer_uid` and
// `actor_uid` are each a sentence the forum is no longer entitled to say, and
// each of them would keep working if it were left in.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func parseQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	q, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("query %q: %v", raw, err)
	}
	return q
}

// TestEditSnapshotUser_TravelsAsTheUser: the snapshot op is NOT viewer-fenced
// upstream — it returns the same entity state a public read renders — so what
// this pins is channel hygiene, not a gate: a read a human triggered must
// arrive as that human, on the Bearer face, rather than as the forum on the
// asserted Basic lane. No site travels; entity_type + entity_id are the whole
// query.
func TestEditSnapshotUser_TravelsAsTheUser(t *testing.T) {
	srv, got := recordingServer(t, 0,
		`{"code":0,"message":"ok","data":{"values":{"catalog.work.name_zh_cn":"现值"}}}`)

	values, err := userClient(srv.URL).EditSnapshotUser(context.Background(), "user-jwt", "catalog.work", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/user/catalog/edit/snapshot" {
		t.Fatalf("snapshot hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if got.query != "entity_id=1000&entity_type=catalog.work" {
		t.Fatalf("snapshot query = %q", got.query)
	}
	if values["catalog.work.name_zh_cn"] != "现值" {
		t.Fatalf("values decoded wrong: %v", values)
	}
}

// The "my proposals" read is the one that used to name a uid. Mine=true must
// become the `mine` flag and nothing else: a proposer_uid on this plane would
// be a caller naming a person again, which is the whole thing that moved.
func TestListEditProposalsUser_MineIsTheToken(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"items":[`+
		`{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"open","proposer_uid":9,"patch":{}}`+
		`],"total":1}}`)

	items, err := userClient(srv.URL).ListEditProposalsUser(context.Background(), "user-jwt",
		UserEditProposalFilter{EntityType: "catalog.work", EntityID: 1000, Status: "open", Limit: 20, Mine: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/user/catalog/edit/proposals" {
		t.Fatalf("list hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	q := parseQuery(t, got.query)
	if q.Get("mine") != "true" {
		t.Fatalf("mine flag missing: %q", got.query)
	}
	if q.Has("proposer_uid") || q.Has("site") {
		t.Fatalf("the user plane names neither a proposer nor a site: %q", got.query)
	}
	if q.Get("entity_id") != "1000" || q.Get("status") != "open" || q.Get("limit") != "20" {
		t.Fatalf("filter lost on the wire: %q", got.query)
	}
	if len(items) != 1 || items[0].ID != 7 {
		t.Fatalf("items decoded wrong: %+v", items)
	}
}

// The review queue is the SAME face without the flag — the catalog decides
// whether this token may see everybody's proposals. Sending mine=false would
// invite an upstream that reads the parameter loosely to treat "present" as
// "true", so the flag is omitted rather than negated.
func TestListEditProposalsUser_QueueOmitsMine(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"items":[],"total":0}}`)

	if _, err := userClient(srv.URL).ListEditProposalsUser(context.Background(), "mod-jwt",
		UserEditProposalFilter{EntityType: "catalog.work", Status: "open"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parseQuery(t, got.query).Has("mine") {
		t.Fatalf("the queue read must not carry mine at all: %q", got.query)
	}
}

// A contributor asking for the queue is refused by the catalog, not by us —
// and the refusal has to stay a refusal (a plain 403), distinct from the stale
// grant that asks for a re-login.
func TestListEditProposalsUser_QueueDenialStaysADenial(t *testing.T) {
	srv, _ := recordingServer(t, http.StatusForbidden,
		`{"code":233,"message":"permission denied: catalog.edit.review"}`)

	_, err := userClient(srv.URL).ListEditProposalsUser(context.Background(), "user-jwt",
		UserEditProposalFilter{EntityType: "catalog.work"})
	if errors.Is(err, ErrInsufficientScope) {
		t.Fatal("a permission denial must not be reported as a scope denial")
	}
	var apiErr *UserAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Fatalf("queue denial = %v, want a 403 UserAPIError", err)
	}
}

// The covers read: the ballot comes from the token, so there is no uid to send
// and no query at all.
func TestWorkCoversUser_BallotFromTheToken(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"covers":[`+
		`{"id":88,"image_hash":"abc","vote_count":3,"voted":true}]}}`)

	covers, err := userClient(srv.URL).WorkCoversUser(context.Background(), "user-jwt", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/user/catalog/works/1000/covers" {
		t.Fatalf("covers hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if got.query != "" {
		t.Fatalf("the viewer is the token, not a query parameter: %q", got.query)
	}
	if len(covers) != 1 || covers[0].ID != 88 || !covers[0].Voted || covers[0].VoteCount != 3 {
		t.Fatalf("covers decoded wrong: %+v", covers)
	}
}

// The upload was the last asserted-actor WRITE. The multipart body must carry
// the file and the preset and NOTHING that names a person.
func TestUploadEditImageUser_SendsNoActorUID(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotFields = map[string]string{}
		gotFile   []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("content type: %v", err)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			raw, _ := io.ReadAll(part)
			if part.FileName() != "" {
				gotFile = raw
				continue
			}
			gotFields[part.FormName()] = string(raw)
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"hash":"h1","url":"https://cdn/image/h1","width":1920,"height":1080,"size_bytes":4}}`))
	}))
	t.Cleanup(srv.Close)

	res, err := userClient(srv.URL).UploadEditImageUser(context.Background(), "user-jwt",
		bytes.NewReader([]byte("PNG!")), "cover.png", "galgame_banner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/v1/user/catalog/edit/images" {
		t.Fatalf("upload hit %s", gotPath)
	}
	if gotAuth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", gotAuth)
	}
	if strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatal("the user plane must never attach the client credential")
	}
	if _, ok := gotFields["actor_uid"]; ok {
		t.Fatalf("the upload asserts no actor: %v", gotFields)
	}
	if gotFields["preset"] != "galgame_banner" || string(gotFile) != "PNG!" {
		t.Fatalf("upload body wrong: fields %v file %q", gotFields, gotFile)
	}
	if res.Hash != "h1" || res.Width != 1920 {
		t.Fatalf("upload result decoded wrong: %+v", res)
	}
}

// The upload shares the user plane's taxonomy: an old session is told to log
// back in, not that it may not upload.
func TestUploadEditImageUser_ScopeDenial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":233,"message":"missing required scope: catalog:edit"}`))
	}))
	t.Cleanup(srv.Close)

	_, err := userClient(srv.URL).UploadEditImageUser(context.Background(), "old-jwt",
		bytes.NewReader([]byte("x")), "cover.png", "galgame_banner")
	if !errors.Is(err, ErrInsufficientScope) {
		t.Fatalf("upload scope denial = %v, want ErrInsufficientScope", err)
	}
}

// An empty token never leaves the process: a dead session is not a question
// for the catalog.
func TestUploadEditImageUser_RefusesAnEmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("an empty token must not reach the catalog")
	}))
	t.Cleanup(srv.Close)

	_, err := userClient(srv.URL).UploadEditImageUser(context.Background(), "",
		bytes.NewReader([]byte("x")), "cover.png", "galgame_banner")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("empty token = %v, want ErrUnauthorized", err)
	}
}
