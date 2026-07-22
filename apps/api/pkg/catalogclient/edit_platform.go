// The editing-engine PLATFORM propose face (infra 09-open-api-phase2 06b — the
// devapi dogfood bridge at /internal/edit/*). Where edit.go ASSERTS the end-user
// identity over the Basic-authed S2S channel, this transport speaks the
// platform's dual-credential devapi channel: the service X-API-Key (the SAME
// internal-tier key the 06a galgame write chain carries) plus the end user's
// Bearer JWT. The platform derives the actor SERVER-SIDE from that JWT, so the
// platform DTOs carry NO actor / site / proposer_uid — the platform pins
// proposer = JWT subject and tenant = the key's oauth_clients.catalog_site
// binding. Only the policy-irrelevant proposal subset (mine / withdraw /
// snapshot) for everyone, and submit / schema for PLAIN actors, routes here
// (06b P6); owner/staff writes stay on the S2S actor-assertion channel so owner
// direct-edit automerge never silently degrades.
//
// Response envelopes are byte-identical to the S2S edit face (06b P10 — the
// platform reuses the same view constructors), so the SAME typed shapes
// (EditProposal / EditSchema / EditCreateResult / …) parse both faces; nothing
// is re-declared here. The one visible wire difference — Huma prepends a
// "$schema" decoration key to the S2S body that the pure-Fiber platform omits —
// is invisible to the {code,message,data} envelope decode below.
package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PlatformEditClient is the dual-credential transport onto catalog's platform
// propose face. It is a distinct type from the S2S Client on purpose: its method
// set physically cannot carry an asserted actor/site (06b G7) — every operation
// takes only the user's Bearer, and the platform decides identity + tenant.
type PlatformEditClient struct {
	baseURL    string // {NextMoe host base}/internal/edit ("" = inert)
	apiKey     string
	httpClient *http.Client
}

// PlatformConfig bundles connection settings (created in app.go from config).
// BaseURL is the NextMoe host base (no /internal suffix) — the SAME base the 06a
// galgame write chain uses (no new env). APIKey is the SAME internal-tier devapi
// key. An empty BaseURL / APIKey = an inert client whose calls return
// ErrNotConfigured, so the BFF degrades to 503 (never a local fallback), exactly
// like the S2S Client.
type PlatformConfig struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client // optional; defaults to a 10s-timeout client
}

// platformEditBase is the propose face's route prefix under the NextMoe internal
// face (mirrors the infra mount: /internal/edit/*).
const platformEditBase = "/internal/edit"

// NewPlatform constructs a PlatformEditClient. Empty BaseURL or APIKey = a no-op
// client (Configured reports false), so kungal degrades gracefully when the
// NextMoe base/key isn't wired.
func NewPlatform(cfg PlatformConfig) *PlatformEditClient {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	base := ""
	if b := strings.TrimRight(cfg.BaseURL, "/"); b != "" {
		base = b + platformEditBase
	}
	return &PlatformEditClient{baseURL: base, apiKey: cfg.APIKey, httpClient: hc}
}

// Configured reports whether the client can reach the platform propose face.
func (c *PlatformEditClient) Configured() bool { return c.baseURL != "" && c.apiKey != "" }

// PlatformCreateRequest files a proposal as the plain JWT actor. It carries NO
// actor and NO site (06b P2/P5) — both are server-derived from the credentials.
type PlatformCreateRequest struct {
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	Patch      map[string]any `json:"patch"`
	Note       string         `json:"note,omitempty"`
}

// PlatformMineFilter narrows the caller's own proposal list. proposer_uid and
// site are intentionally absent: the platform forces both server-side (06b
// P2/P5) — a client can never widen the scope to enumerate others' proposals or
// cross tenants.
type PlatformMineFilter struct {
	EntityType string
	EntityID   int64
	Status     string
	Limit      int
}

// CreateProposal files an edit through the platform propose face (plain actor).
func (c *PlatformEditClient) CreateProposal(ctx context.Context, bearer string, req PlatformCreateRequest) (*EditCreateResult, error) {
	return platformPost[EditCreateResult](ctx, c, bearer, "/proposals", req)
}

// MineProposals lists the caller's own proposals. The platform hard-scopes the
// list to (this JWT uid, the key-bound site, galgame family); this client only
// forwards the optional entity/status/limit narrowing.
func (c *PlatformEditClient) MineProposals(ctx context.Context, bearer string, f PlatformMineFilter) ([]EditProposal, error) {
	q := url.Values{}
	if f.EntityType != "" {
		q.Set("entity_type", f.EntityType)
	}
	if f.EntityID > 0 {
		q.Set("entity_id", strconv.FormatInt(f.EntityID, 10))
	}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	data, err := platformGet[struct {
		Items []EditProposal `json:"items"`
	}](ctx, c, bearer, "/proposals?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return data.Items, nil
}

// WithdrawProposal closes the caller's own open proposal. The platform enforces
// family + tenant (404 anti-enumeration) and the proposer-only rule (403) itself
// — no pre-flight read is needed on the BFF side.
func (c *PlatformEditClient) WithdrawProposal(ctx context.Context, bearer string, id int64) (*EditProposal, error) {
	return platformPost[EditProposal](ctx, c, bearer, "/proposals/"+strconv.FormatInt(id, 10)+"/withdraw", nil)
}

// GetSchema reads the field schema + the PLAIN caller's capability projection
// (entityID 0 = type-level). No asserted overlay is sent (06b P2): the platform
// projects the plain actor, byte-identical to the S2S plain-actor schema.
func (c *PlatformEditClient) GetSchema(ctx context.Context, bearer, entityType string, entityID int64) (*EditSchema, error) {
	path := "/schema/" + entityType
	if entityID > 0 {
		path += "?entity_id=" + strconv.FormatInt(entityID, 10)
	}
	return platformGet[EditSchema](ctx, c, bearer, path)
}

// Snapshot reads the entity's CURRENT registered-field values (the editor
// bootstrap's value source). Policy-irrelevant and family-fenced on the
// platform.
func (c *PlatformEditClient) Snapshot(ctx context.Context, bearer, entityType string, entityID int64) (map[string]any, error) {
	q := url.Values{}
	q.Set("entity_type", entityType)
	q.Set("entity_id", strconv.FormatInt(entityID, 10))
	data, err := platformGet[struct {
		Values map[string]any `json:"values"`
	}](ctx, c, bearer, "/snapshot?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return data.Values, nil
}

// platformGet / platformPost / platformDo mirror edit.go's editGet/editPost/
// editDo but swap the Basic header for the dual-credential pair (X-API-Key +
// Bearer). Kept as a separate transport so the reviewed S2S path stays
// byte-untouched; the {code,message,data} envelope contract is frozen and shared.
func platformGet[T any](ctx context.Context, c *PlatformEditClient, bearer, path string) (*T, error) {
	return platformDo[T](ctx, c, http.MethodGet, bearer, path, nil)
}

func platformPost[T any](ctx context.Context, c *PlatformEditClient, bearer, path string, body any) (*T, error) {
	var raw []byte
	if body != nil {
		var err error
		if raw, err = json.Marshal(body); err != nil {
			return nil, err
		}
	}
	return platformDo[T](ctx, c, http.MethodPost, bearer, path, raw)
}

func platformDo[T any](ctx context.Context, c *PlatformEditClient, method, bearer, path string, body []byte) (*T, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Dual-credential transport (06b P2): the service key identifies kungal's
	// devapi client (its catalog_site binding is the filing tenant); the user's
	// Bearer is the identity the platform derives the plain actor from. No
	// actor/site/proposer is ever sent — the platform forces all three.
	req.Header.Set("X-API-Key", c.apiKey)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("%w: malformed envelope", ErrUpstream)
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return nil, &EditAPIError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	var out T
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return nil, fmt.Errorf("%w: malformed data payload", ErrUpstream)
		}
	}
	return &out, nil
}
