package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/pkg/errors"
)

// briefCacheTTL memoizes the public batch lookups (names/covers + detail briefs)
// so the activity feed's repeated GetBatchPublic / GetBatchDetailPublic calls —
// across scroll pages and concurrent viewers — don't re-hit the wiki for the same
// galgames. Names/covers are stable enough to tolerate a couple of minutes stale.
const briefCacheTTL = 2 * time.Minute

// batchCacheMaxEntries crudely bounds each cache's memory: past it the cache is
// cleared (it re-warms within one TTL window). The feed touches recent galgames,
// so this is rarely hit.
const batchCacheMaxEntries = 4096

type batchCacheKey struct {
	id  int
	sfw bool
}

type batchCacheEntry[T any] struct {
	found  bool // false = negative cache (id absent / deleted / NSFW-filtered)
	val    T
	expire time.Time
}

// cachedBatch serves a per-(id,sfw) TTL cache over a batch fetch: returns the
// hits, calls `fetch` for ONLY the misses, then caches the results — including
// negatives, so a deleted/NSFW id isn't re-fetched on every page.
func cachedBatch[T any](
	mu *sync.RWMutex,
	cache map[batchCacheKey]batchCacheEntry[T],
	ids []int,
	sfw bool,
	fetch func([]int) (map[int]T, *errors.AppError),
) (map[int]T, *errors.AppError) {
	result := make(map[int]T, len(ids))
	var missing []int
	now := time.Now()
	mu.RLock()
	for _, id := range ids {
		if e, ok := cache[batchCacheKey{id, sfw}]; ok && now.Before(e.expire) {
			if e.found {
				result[id] = e.val
			}
		} else {
			missing = append(missing, id)
		}
	}
	mu.RUnlock()
	if len(missing) == 0 {
		return result, nil
	}
	fetched, appErr := fetch(missing)
	if appErr != nil {
		return nil, appErr
	}
	expire := now.Add(briefCacheTTL)
	mu.Lock()
	if len(cache) > batchCacheMaxEntries {
		clear(cache)
	}
	for _, id := range missing {
		v, ok := fetched[id]
		cache[batchCacheKey{id, sfw}] = batchCacheEntry[T]{found: ok, val: v, expire: expire}
		if ok {
			result[id] = v
		}
	}
	mu.Unlock()
	return result, nil
}

// GalgameClient calls the Galgame Wiki Service via HTTP.
//
// Holds two authentication contexts:
//   - per-request Bearer token (forwarded from the user's kungal session) —
//     for user-identity endpoints like submit / claim / patch-draft;
//   - a pre-built HTTP Basic header (OAuth client_id:secret, reused from
//     pkg/userclient credentials per decision 3 in 07-submission docs) —
//     for service-to-service endpoints like /galgame/messages/feed.
type GalgameClient struct {
	baseURL    string
	httpClient *http.Client
	basicAuth  string
	// imageCDNBase resolves wiki banner_image_hash → CDN URL inside
	// doRequest (see banner.go). Empty disables resolution (responses
	// pass through untouched).
	imageCDNBase string

	// Public batch-lookup caches (see cachedBatch / briefCacheTTL).
	briefMu     sync.RWMutex
	briefCache  map[batchCacheKey]batchCacheEntry[GalgameBrief]
	detailMu    sync.RWMutex
	detailCache map[batchCacheKey]batchCacheEntry[GalgameDetailBrief]
}

// NewGalgameClient builds a client that can only do anonymous + Bearer calls.
// Suitable when service-to-service endpoints aren't needed.
//
// imageCDNBase must match the wiki's KUN_IMAGE_PUBLIC_BASE_URL so
// hash-backed banners resolve to the same CDN URLs the wiki would build.
func NewGalgameClient(baseURL, imageCDNBase string) *GalgameClient {
	// Clone the default transport and lift the per-host idle-connection
	// cap. net/http defaults MaxIdleConnsPerHost to 2, which throttles a
	// single-host service-to-service client: concurrent callers (runtime
	// list hydration, the release-date backfill's worker pool) can't reuse
	// keep-alive connections beyond 2 and pay a fresh dial per request.
	// Lifting it lets concurrent requests to the one wiki host reuse the
	// pool instead of churning connections.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 64

	return &GalgameClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		imageCDNBase: imageCDNBase,
		briefCache:   map[batchCacheKey]batchCacheEntry[GalgameBrief]{},
		detailCache:  map[batchCacheKey]batchCacheEntry[GalgameDetailBrief]{},
	}
}

// NewGalgameClientWithBasicAuth additionally enables service-to-service
// endpoints (currently: /galgame/messages/feed) by pre-computing the Basic
// auth header. Pass the same OAuth Client ID/secret used by pkg/userclient.
func NewGalgameClientWithBasicAuth(baseURL, imageCDNBase, clientID, clientSecret string) *GalgameClient {
	c := NewGalgameClient(baseURL, imageCDNBase)
	c.basicAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret))
	return c
}

// apiResponse is the standard {code, message, data} wrapper.
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Get performs a GET request to the wiki service.
func (c *GalgameClient) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, *errors.AppError) {
	return c.GetWithToken(ctx, path, "", query)
}

// GetWithToken is like Get but attaches a Bearer token. Used by endpoints
// whose response shape depends on the caller's identity:
//   - /galgame/batch with Bearer returns the caller's own pending drafts
//   - /galgame/search?include_pending=true returns the caller's pending hits
//   - /galgame/mine and /galgame/messages/mine are inherently user-scoped
//
// token "" reduces to an anonymous GET (same as Get).
func (c *GalgameClient) GetWithToken(ctx context.Context, path, token string, query url.Values) (json.RawMessage, *errors.AppError) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.doRequest(req)
}

// EntityGalgameIDs fetches the member galgame ids of a wiki entity
// (kind = "tag" | "official" | "engine") so the forum can run its OWN local
// resource-based filtering over them — the wiki has no resource concept. The
// result is always non-nil so the caller's RestrictIDs guard distinguishes
// "restrict to this (possibly empty) set" from "no restriction".
func (c *GalgameClient) EntityGalgameIDs(ctx context.Context, kind string, id int) ([]int, *errors.AppError) {
	data, appErr := c.Get(ctx, fmt.Sprintf("/%s/%d/galgame-ids", kind, id), nil)
	if appErr != nil {
		return nil, appErr
	}
	var parsed struct {
		IDs []int `json:"ids"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}
	if parsed.IDs == nil {
		parsed.IDs = []int{}
	}
	return parsed.IDs, nil
}

// PostWithToken performs a POST with Bearer token.
//
// contentType controls how body is forwarded:
//   - "" (empty)        → defaults to "application/json"; struct/map bodies
//     are JSON-marshaled
//   - "application/json" → same as empty
//   - any multipart/* / form-encoded / etc. → body MUST be passed as
//     []byte / json.RawMessage,
//     forwarded byte-for-byte
//     with the boundary preserved
func (c *GalgameClient) PostWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "POST", path, token, body, contentType)
}

// PutWithToken performs a PUT with Bearer token. See PostWithToken for
// contentType semantics.
func (c *GalgameClient) PutWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "PUT", path, token, body, contentType)
}

// DeleteWithToken performs a DELETE with Bearer token. See PostWithToken
// for contentType semantics.
func (c *GalgameClient) DeleteWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "DELETE", path, token, body, contentType)
}

// PatchWithToken performs a PATCH with Bearer token. Used by user draft
// edits (PATCH /galgame/:gid for status IN (3,4)). See PostWithToken for
// contentType semantics.
func (c *GalgameClient) PatchWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "PATCH", path, token, body, contentType)
}

func (c *GalgameClient) mutateWithToken(ctx context.Context, method, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	if contentType == "" {
		contentType = "application/json"
	}

	var bodyReader io.Reader
	if body != nil {
		// Pass-through for already-encoded bodies (multipart, form-urlencoded,
		// etc.). Without this, json.Marshal would wrap raw bytes in quotes
		// and lose the multipart boundary.
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case json.RawMessage:
			bodyReader = bytes.NewReader([]byte(v))
		default:
			b, err := json.Marshal(body)
			if err != nil {
				return nil, errors.ErrInternal("序列化请求失败")
			}
			bodyReader = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	req.Header.Set("Content-Type", contentType)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.doRequest(req)
}

// WikiUserStats is the user galgame stats from wiki service.
type WikiUserStats struct {
	GalgameCreated      int64 `json:"galgame_created"`
	GalgameCreatedToday int64 `json:"galgame_created_today"`
	GalgameContributed  int64 `json:"galgame_contributed"`
	PRMerged            int64 `json:"pr_merged"`
}

// GetUserStats fetches galgame-related stats for a user from wiki.
func (c *GalgameClient) GetUserStats(ctx context.Context, userID int) (*WikiUserStats, error) {
	path := fmt.Sprintf("/galgame/user/%d/stats", userID)
	data, appErr := c.Get(ctx, path, nil)
	if appErr != nil {
		return nil, appErr
	}

	var stats WikiUserStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// WikiUserGalgames is the paginated published-galgame list for a user.
type WikiUserGalgames struct {
	Galgames []GalgameBrief `json:"galgames"`
	Total    int64          `json:"total"`
}

// GetUserGalgames fetches a user's PUBLISHED galgames (briefs, newest first,
// paginated) from the wiki — the source for the profile "已发布 Galgame" tab.
// kungal's local galgame mirror has no user_id (ownership is wiki-managed after
// the OAuth migration), so this wiki endpoint is the only way to answer "what
// has this user published". NSFW gating is done server-side by the wiki.
//
//	isSFW=true  → content_limit=sfw  (drop NSFW server-side)
//	isSFW=false → content_limit=all
func (c *GalgameClient) GetUserGalgames(ctx context.Context, userID, page, limit int, isSFW bool) ([]GalgameBrief, int64, *errors.AppError) {
	contentLimit := "all"
	if isSFW {
		contentLimit = "sfw"
	}
	query := url.Values{
		"page":          {strconv.Itoa(page)},
		"limit":         {strconv.Itoa(limit)},
		"content_limit": {contentLimit},
	}
	data, appErr := c.Get(ctx, fmt.Sprintf("/galgame/user/%d/galgames", userID), query)
	if appErr != nil {
		return nil, 0, appErr
	}
	var resp WikiUserGalgames
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, 0, errors.ErrInternal("解析 Wiki 用户 Galgame 列表失败")
	}
	return resp.Galgames, resp.Total, nil
}

// GetUserContributedGalgames fetches a user's CONTRIBUTED galgames — the
// games they created OR edited/claimed — from the wiki (briefs, paginated,
// same shape + NSFW gating as GetUserGalgames). Source for the profile
// "贡献的 Galgame" tab. This is the superset of the "已发布" (created) list:
// "已发布" answers "what did this user create", "贡献的" adds games they only
// edited. Backed by wiki's GET /galgame/user/:id/contributed.
func (c *GalgameClient) GetUserContributedGalgames(ctx context.Context, userID, page, limit int, isSFW bool) ([]GalgameBrief, int64, *errors.AppError) {
	contentLimit := "all"
	if isSFW {
		contentLimit = "sfw"
	}
	query := url.Values{
		"page":          {strconv.Itoa(page)},
		"limit":         {strconv.Itoa(limit)},
		"content_limit": {contentLimit},
	}
	data, appErr := c.Get(ctx, fmt.Sprintf("/galgame/user/%d/contributed", userID), query)
	if appErr != nil {
		return nil, 0, appErr
	}
	var resp WikiUserGalgames
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, 0, errors.ErrInternal("解析 Wiki 用户贡献 Galgame 列表失败")
	}
	return resp.Galgames, resp.Total, nil
}

// WikiAdminStats is the admin stats response from wiki service.
type WikiAdminStats struct {
	Totals map[string]int64 `json:"totals"`
	Daily  []map[string]any `json:"daily"`
}

// GetAdminStats fetches wiki-side admin stats for the last N days. The wiki's
// /admin/stats is an authenticated admin endpoint (401s anonymously), so the
// caller MUST forward the requesting admin's OAuth Bearer — sourced from the
// kungal session via middleware.GetAccessToken, exactly like the submission-
// review (/admin/galgame*) calls. An empty token degrades to an anonymous call
// the wiki rejects, which the overview merges as zeroes (non-blocking).
func (c *GalgameClient) GetAdminStats(ctx context.Context, days int, token string) (*WikiAdminStats, error) {
	query := url.Values{"days": {fmt.Sprintf("%d", days)}}
	data, appErr := c.GetWithToken(ctx, "/admin/stats", token, query)
	if appErr != nil {
		return nil, appErr
	}

	var stats WikiAdminStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// GalgameBrief is the lightweight metadata returned by /galgame/batch.
//
// Status is meaningful for Bearer-authenticated calls (which can see the
// caller's own status=3 pending / 4 declined drafts in addition to status=0).
// Anonymous calls always get status=0 entries — see 01-galgame.md.
//
// EffectiveBannerHash is the derived hash from covers[sort_order=0];
// frontend reads it (or the rewriteBanners-injected
// effective_banner_url) to render head images. banner_image_hash was
// retired in wiki PR5 (K-PR6) — no top-level field any more.
type GalgameBrief struct {
	ID                 int    `json:"id"`
	VndbID             string `json:"vndb_id"`
	NameEnUs           string `json:"name_en_us"`
	NameJaJp           string `json:"name_ja_jp"`
	NameZhCn           string `json:"name_zh_cn"`
	NameZhTw           string `json:"name_zh_tw"`
	Banner             string `json:"banner"`
	Status             int    `json:"status"`
	ContentLimit       string `json:"content_limit"`
	UserID             int    `json:"user_id"`
	ResourceUpdateTime string `json:"resource_update_time"`
	OriginalLanguage   string `json:"original_language"`
	AgeLimit           string `json:"age_limit"`
	// U1: see WikiGalgameDetailFull. nil = unknown; TBA can coexist with a
	// concrete date ("predicted 2024 sometime") so don't enforce mutex.
	ReleaseDate    *string `json:"release_date"`
	ReleaseDateTBA bool    `json:"release_date_tba"`
	// U2: derived effective banner hash on briefs (wiki computes from the
	// row's covers[sort_order=0]). EffectiveBannerURL is injected by
	// rewriteBanners over the wiki response BEFORE this struct is
	// unmarshalled — declare the field so we capture it; without it
	// Go's unmarshal silently drops the walker's work and downstream
	// DTOs are stuck with only the hash. banner_image_hash retired in
	// wiki PR5 (K-PR6).
	EffectiveBannerHash      string `json:"effective_banner_hash"`
	EffectiveBannerURL       string `json:"effective_banner_url"`
	EffectiveBannerWidth     int    `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int    `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
}

// GalgameDetailBrief is GalgameBrief plus the introduction + officials a richer
// list view needs (the "new galgame" feed card). Served by
// GET /galgame/batch?view=detail (see docs/galgame_wiki/01-galgame.md). Officials
// are bare maker names — the caller joins them (e.g. 、) for "由 X 制作". The
// embedded GalgameBrief's release_date IS populated in the detail view (the plain
// brief omits it).
type GalgameDetailBrief struct {
	GalgameBrief
	IntroEnUS string   `json:"intro_en_us"`
	IntroJaJP string   `json:"intro_ja_jp"`
	IntroZhCN string   `json:"intro_zh_cn"`
	IntroZhTW string   `json:"intro_zh_tw"`
	Officials []string `json:"officials"`
}

// GetBatchDetailPublic is GetBatchPublic's richer sibling: GET
// /galgame/batch?view=detail returns GalgameDetailBrief (intro + officials +
// release_date) for list cards that need it. Same NSFW contract as
// GetBatchPublic (isSFW → content_limit=sfw). Anonymous (status=0 only).
func (c *GalgameClient) GetBatchDetailPublic(ctx context.Context, ids []int, isSFW bool) (map[int]GalgameDetailBrief, *errors.AppError) {
	return cachedBatch(&c.detailMu, c.detailCache, ids, isSFW, func(miss []int) (map[int]GalgameDetailBrief, *errors.AppError) {
		limit := "all"
		if isSFW {
			limit = "sfw"
		}
		idStrs := make([]string, len(miss))
		for i, id := range miss {
			idStrs[i] = strconv.Itoa(id)
		}
		query := url.Values{
			"ids":           {joinStrings(idStrs, ",")},
			"content_limit": {limit},
			"view":          {"detail"},
		}
		data, appErr := c.Get(ctx, "/galgame/batch", query)
		if appErr != nil {
			return nil, appErr
		}
		var briefs []GalgameDetailBrief
		if err := json.Unmarshal(data, &briefs); err != nil {
			return nil, errors.ErrInternal("解析 Wiki 批量详情响应失败")
		}
		result := make(map[int]GalgameDetailBrief, len(briefs))
		for _, b := range briefs {
			result[b.ID] = b
		}
		return result, nil
	})
}

// GetBatch fetches lightweight galgame info for multiple IDs anonymously
// (status=0 only). Returns a map[galgameID] -> GalgameBrief for easy lookup.
//
// For "show me my own pending drafts too" use GetBatchWithViewer.
func (c *GalgameClient) GetBatch(ctx context.Context, ids []int) (map[int]GalgameBrief, *errors.AppError) {
	return c.GetBatchWithOptions(ctx, ids, "", "")
}

// GetBatchPublic is the cookie-aware batch fetch for any public list /
// feed enrichment path: enriches kungal-local IDs with wiki briefs while
// honouring the caller's NSFW preference.
//
//	isSFW=true  → content_limit=sfw  (drop NSFW server-side)
//	isSFW=false → content_limit=all  (caller opted in to NSFW)
//
// Per docs/galgame_wiki/00-handbook §16, /galgame/batch defaults to
// NO filter (callers presumed to know the IDs they want). Any path
// reachable by anonymous traffic / search crawlers MUST go through
// this helper rather than the bare GetBatch — see §16 "不要在下游做
// 客户端 filtering" for why service-layer post-filtering isn't
// equivalent (data has already left the wiki boundary).
func (c *GalgameClient) GetBatchPublic(ctx context.Context, ids []int, isSFW bool) (map[int]GalgameBrief, *errors.AppError) {
	return cachedBatch(&c.briefMu, c.briefCache, ids, isSFW, func(miss []int) (map[int]GalgameBrief, *errors.AppError) {
		limit := "all"
		if isSFW {
			limit = "sfw"
		}
		return c.GetBatchWithOptions(ctx, miss, "", limit)
	})
}

// GetBatchWithViewer is the Bearer-aware batch fetch. With a non-empty token
// the wiki additionally returns any status=3/4 row whose user_id matches
// the JWT's userID claim — used by the "我的提交"/"发布向导" UX.
//
// token="" reduces to the anonymous form.
func (c *GalgameClient) GetBatchWithViewer(ctx context.Context, ids []int, token string) (map[int]GalgameBrief, *errors.AppError) {
	return c.GetBatchWithOptions(ctx, ids, token, "")
}

// GetBatchWithOptions is the fully-parameterized batch fetch:
//
//   - token: caller's Bearer access_token; "" = anonymous
//   - contentLimit: "sfw" / "nsfw" / "all" / "" (omit, wiki default = no filter)
//
// Per docs/galgame_wiki/00-handbook §16, /galgame/batch's default is
// **no filter** (the caller already knows the IDs they want). Public
// list/feed paths that re-use the batch endpoint to enrich kungal-local
// IDs MUST pass "sfw" to keep NSFW out — there is no implicit safety
// net at this layer.
func (c *GalgameClient) GetBatchWithOptions(ctx context.Context, ids []int, token, contentLimit string) (map[int]GalgameBrief, *errors.AppError) {
	if len(ids) == 0 {
		return map[int]GalgameBrief{}, nil
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.Itoa(id)
	}
	query := url.Values{"ids": {joinStrings(idStrs, ",")}}
	if contentLimit != "" {
		query.Set("content_limit", contentLimit)
	}

	data, appErr := c.GetWithToken(ctx, "/galgame/batch", token, query)
	if appErr != nil {
		return nil, appErr
	}

	var briefs []GalgameBrief
	if err := json.Unmarshal(data, &briefs); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 批量响应失败")
	}

	result := make(map[int]GalgameBrief, len(briefs))
	for _, b := range briefs {
		result[b.ID] = b
	}
	return result, nil
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for _, v := range s[1:] {
		result += sep + v
	}
	return result
}

// ──────────────────────────────────────────
// Submission (user-identity, Bearer-forwarded)
// ──────────────────────────────────────────

// SubmitDraft posts a new pending submission (status=3). Returns the wiki
// response data raw so the handler can forward verbatim. See
// docs/galgame_wiki/07-submission.md §POST /galgame/submit.
func (c *GalgameClient) SubmitDraft(ctx context.Context, token string, body []byte, contentType string) (json.RawMessage, *errors.AppError) {
	return c.PostWithToken(ctx, "/galgame/submit", token, json.RawMessage(body), contentType)
}

// ClaimDraft flips a VNDB-source draft (status=2) to published (status=0)
// and assigns the caller as creator + contributor. Server enforces the
// status precondition.
func (c *GalgameClient) ClaimDraft(ctx context.Context, token string, gid int) (json.RawMessage, *errors.AppError) {
	path := "/galgame/" + strconv.Itoa(gid) + "/claim"
	return c.PostWithToken(ctx, path, token, json.RawMessage(`{}`), "application/json")
}

// PatchDraft updates the caller's own pending/declined draft (status IN 3,4).
// If the row was status=4, the wiki flips it back to status=3 (re-queues).
func (c *GalgameClient) PatchDraft(ctx context.Context, token string, gid int, body []byte, contentType string) (json.RawMessage, *errors.AppError) {
	path := "/galgame/" + strconv.Itoa(gid)
	return c.PatchWithToken(ctx, path, token, json.RawMessage(body), contentType)
}

// DeleteDraft hard-deletes the caller's own pending/declined draft. Wiki
// CASCADEs the associated wiki tables; kungal still needs to clean its
// local stub if interaction lazy-created one.
func (c *GalgameClient) DeleteDraft(ctx context.Context, token string, gid int) *errors.AppError {
	path := "/galgame/" + strconv.Itoa(gid)
	_, err := c.DeleteWithToken(ctx, path, token, nil, "")
	return err
}

// ──────────────────────────────────────────
// Message feed (service identity, Basic auth)
// ──────────────────────────────────────────

// WikiMessageGalgameBrief is the brief embed inside each WikiMessage.
// Null on hard-deleted galgames — consumers must null-check.
type WikiMessageGalgameBrief struct {
	ID     int `json:"id"`
	Status int `json:"status"`
}

// WikiMessage matches the per-message shape in /galgame/messages/feed.
// See docs/galgame_wiki/08-messages.md for the wire format.
type WikiMessage struct {
	ID           int64                    `json:"id"`
	Type         string                   `json:"type"`
	GalgameID    int                      `json:"galgame_id"`
	Galgame      *WikiMessageGalgameBrief `json:"galgame"`
	ActorUserID  int                      `json:"actor_user_id"`
	TargetUserID *int                     `json:"target_user_id"`
	Payload      json.RawMessage          `json:"payload"`
	CreatedAt    string                   `json:"created_at"`
}

// WikiMessageFeed is the envelope returned by /galgame/messages/feed.
type WikiMessageFeed struct {
	Items   []WikiMessage `json:"items"`
	HasMore bool          `json:"has_more"`
}

// MessagesFeed pulls a batch of admin-triggered events (approved /
// declined / banned / unbanned) using OAuth Client Basic Auth. Used by
// the wiki-message sync cron. Returns ErrInternal if the client wasn't
// constructed with NewGalgameClientWithBasicAuth.
func (c *GalgameClient) MessagesFeed(ctx context.Context, sinceID int64, limit int) (*WikiMessageFeed, *errors.AppError) {
	if c.basicAuth == "" {
		return nil, errors.ErrInternal("wiki client 未配置 Basic Auth 凭证")
	}
	if limit <= 0 {
		limit = 1000
	}

	reqURL := c.baseURL + "/galgame/messages/feed?since_id=" +
		strconv.FormatInt(sinceID, 10) + "&limit=" + strconv.Itoa(limit)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	req.Header.Set("Authorization", c.basicAuth)

	data, appErr := c.doRequest(req)
	if appErr != nil {
		return nil, appErr
	}
	var feed WikiMessageFeed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, errors.ErrInternal("解析 wiki 消息 feed 失败")
	}
	return &feed, nil
}

// WikiRevision matches one item in /galgame/revisions/recent — a merged
// (edit-landed) galgame revision. UserID is the editor (correct actor).
type WikiRevision struct {
	ID        int64  `json:"id"`
	GalgameID int    `json:"galgame_id"`
	Revision  int    `json:"revision"` // per-galgame number — the diff endpoint's :rev
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Created   string `json:"created"`
}

// WikiRevisionFeed is the envelope returned by /galgame/revisions/recent.
type WikiRevisionFeed struct {
	Items   []WikiRevision `json:"items"`
	HasMore bool           `json:"has_more"`
}

// RecentRevisions pulls a batch of merged-revision (edit) events using OAuth
// Client Basic Auth — the wiki-revision sync cron's source for mirroring galgame
// edits into the local activity timeline. Returns ErrInternal if the client
// wasn't constructed with NewGalgameClientWithBasicAuth.
func (c *GalgameClient) RecentRevisions(ctx context.Context, sinceID int64, limit int) (*WikiRevisionFeed, *errors.AppError) {
	if c.basicAuth == "" {
		return nil, errors.ErrInternal("wiki client 未配置 Basic Auth 凭证")
	}
	if limit <= 0 {
		limit = 1000
	}

	reqURL := c.baseURL + "/galgame/revisions/recent?since_id=" +
		strconv.FormatInt(sinceID, 10) + "&limit=" + strconv.Itoa(limit)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	req.Header.Set("Authorization", c.basicAuth)

	data, appErr := c.doRequest(req)
	if appErr != nil {
		return nil, appErr
	}
	var feed WikiRevisionFeed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, errors.ErrInternal("解析 wiki 修订 feed 失败")
	}
	return &feed, nil
}

// bodySnippet trims an upstream body for safe logging / error context.
// Wiki errors are tiny JSON; a misconfigured upstream may return a large
// HTML page, so cap it.
func bodySnippet(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max]) + "…(truncated)"
	}
	return string(b)
}

// ──────────────────────────────────────────
// Cross-source scores / stats (public wiki read face, step 34)
// ──────────────────────────────────────────

// Scores fetches the three-source (VNDB / Bangumi / EG) rating snapshot for one
// galgame from the wiki's public read face and returns the response `data`
// verbatim (naming = contract; the 37 frontend reads the infra shape directly).
// Missing sources are null server-side; a galgame with no rows returns all-null.
func (c *GalgameClient) Scores(ctx context.Context, gid int) (json.RawMessage, *errors.AppError) {
	return c.Get(ctx, "/galgame/"+strconv.Itoa(gid)+"/scores", nil)
}

// StatsResult carries a stats fetch: the verbatim `data` payload plus the wiki's
// ETag, or NotModified when the wiki answered 304 to the forwarded If-None-Match
// (Data is then nil). The BFF re-emits the ETag and the 304 so the browser's
// conditional cache survives the extra hop.
type StatsResult struct {
	NotModified bool
	ETag        string
	Data        json.RawMessage
}

// Stats fetches the site-wide cross-source statistics overview from the wiki's
// public read face, preserving the ETag / If-None-Match conditional cache. A
// non-empty ifNoneMatch is forwarded; a wiki 304 comes back as
// StatsResult.NotModified so the caller can relay an empty 304. The six frozen
// keys' payloads pass through untouched. This does not go through the shared
// doRequest because it needs the raw status code + ETag header, which the
// envelope-only doRequest hides.
func (c *GalgameClient) Stats(ctx context.Context, ifNoneMatch string) (*StatsResult, *errors.AppError) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/galgame/stats", nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Wiki 服务请求失败 (传输层)",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return nil, errors.ErrInternal(fmt.Sprintf("Wiki 服务不可达: %v", err))
	}
	defer resp.Body.Close()

	etag := resp.Header.Get("ETag")
	if resp.StatusCode == http.StatusNotModified {
		return &StatsResult{NotModified: true, ETag: etag}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.ErrInternal("读取 Wiki 响应失败")
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("解析 Wiki 响应失败 (非 JSON 响应)",
			"method", req.Method, "url", req.URL.String(),
			"status", resp.StatusCode, "body", bodySnippet(body))
		return nil, errors.New(errors.CodeBiz,
			fmt.Sprintf("Wiki 服务返回了非预期响应 (HTTP %d)", resp.StatusCode), resp.StatusCode)
	}
	if result.Code != 0 {
		return nil, errors.New(result.Code, result.Message, resp.StatusCode)
	}
	return &StatsResult{ETag: etag, Data: result.Data}, nil
}

func (c *GalgameClient) doRequest(req *http.Request) (json.RawMessage, *errors.AppError) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Transport-level failure: wiki unreachable / DNS / timeout /
		// connection refused. The most common operational cause is the
		// wiki service simply not running on the configured base URL.
		slog.Error("Wiki 服务请求失败 (传输层)",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return nil, errors.ErrInternal(fmt.Sprintf("Wiki 服务不可达: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("读取 Wiki 响应失败",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return nil, errors.ErrInternal("读取 Wiki 响应失败")
	}

	var result apiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Wiki returned something that isn't the {code,message,data}
		// envelope — almost always a Fiber default error page (e.g. a
		// plain-text "Cannot POST /api/galgame/30831/claim" when the
		// running wiki binary predates the submission endpoints, or an
		// upstream proxy 5xx HTML). Surface the real HTTP status + body
		// so this is diagnosable instead of a blanket 500.
		slog.Error("解析 Wiki 响应失败 (非 JSON 响应)",
			"method", req.Method, "url", req.URL.String(),
			"status", resp.StatusCode, "body", bodySnippet(respBody))
		return nil, errors.New(
			errors.CodeBiz,
			fmt.Sprintf(
				"Wiki 服务返回了非预期响应 (HTTP %d), 请确认 wiki 服务已部署对应接口",
				resp.StatusCode,
			),
			resp.StatusCode,
		)
	}

	if result.Code != 0 {
		// Transparently forward wiki service error code + message.
		return nil, errors.New(result.Code, result.Message, resp.StatusCode)
	}

	// Resolve image_service hash-backed banners → CDN URLs once, here,
	// for EVERY galgame payload (typed mappers + verbatim passthroughs
	// like /galgame/mine). Cosmetic + fail-safe: see banner.go.
	return rewriteBanners(result.Data, c.imageCDNBase), nil
}
