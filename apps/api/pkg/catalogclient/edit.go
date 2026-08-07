// The editing-engine S2S face (infra E3a; contract /api/v1/catalog/edit/*),
// reduced by wave 178 to the claim-free reads and by wave 180 to the
// VIEWER-INDEPENDENT ones: the revision log, the diff, the public per-game
// proposal list, and the third-person contribution counts. Those answer the
// same thing for everybody, so there is nothing here for a token to carry.
//
// Everything that is read BY somebody — the value snapshot, one proposal's
// detail, "my proposals" — went to user_edit.go's Bearer plane along with every
// act, where the catalog reads the subject, the tenant and the entity's
// ownership off the token itself.
//
// The shared wire types below serve both planes: the Bearer face answers with
// the same proposal / amendment / revision / schema shapes.
//
// Unlike the GET-only reads in client.go, the edit face's 4xx replies carry
// actionable reasons (validation details, policy denials, rebase conflicts),
// so these calls keep the envelope's business code + message as a typed
// EditAPIError instead of collapsing everything into the read sentinels.
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
	"time"
)

// EditActor — the asserted end-user identity (uid + roles + trust tier) that
// the S2S posture used to carry — is GONE as of wave 179, and this note is here
// so nobody reintroduces it as a convenience.
//
// It shrank once already: wave 178 dropped its IsEntityOwner flag when the
// catalog started holding per-user work ownership itself. Wave 179 took the
// rest, because the claims face was its last consumer and that face now derives
// the subject, the tenant and the review permission from the user's own token.
// Nothing S2S is left that acts on a person's behalf — the surviving Basic-authed
// calls are reads with no actor at all — so a struct whose whole job was for the
// forum to assert who somebody is has no honest use.

// EditProposal is the wire shape of a proposal (status: open / merged /
// declined / withdrawn). EffectivePatch and Amendments only arrive on the
// detail read.
type EditProposal struct {
	ID              int64           `json:"id"`
	EntityType      string          `json:"entity_type"`
	EntityID        int64           `json:"entity_id"`
	BaseRevisionSeq int             `json:"base_revision_seq"`
	Patch           map[string]any  `json:"patch"`
	EffectivePatch  map[string]any  `json:"effective_patch,omitempty"`
	ProposerUID     int64           `json:"proposer_uid"`
	Note            string          `json:"note"`
	Site            string          `json:"site"`
	Status          string          `json:"status"`
	DecidedByUID    *int64          `json:"decided_by_uid,omitempty"`
	DecidedAt       *time.Time      `json:"decided_at,omitempty"`
	DecisionNote    string          `json:"decision_note,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Amendments      []EditAmendment `json:"amendments,omitempty"`
}

// EditAmendment is one maintainer patch delta (set/unset), seq-ordered.
type EditAmendment struct {
	ID         int64          `json:"id"`
	Seq        int            `json:"seq"`
	Set        map[string]any `json:"set,omitempty"`
	Unset      []string       `json:"unset,omitempty"`
	AmenderUID int64          `json:"amender_uid"`
	Note       string         `json:"note"`
	CreatedAt  time.Time      `json:"created_at"`
}

// EditRevision is one row of the append-only revision log. Migrated rows
// (the E2 transform's 11,860) additionally carry honest provenance: the
// original action word plus the old-wire note / minor-edit flag.
type EditRevision struct {
	ID            int64          `json:"id"`
	Seq           int            `json:"seq"`
	Action        string         `json:"action"`
	ChangedFields []string       `json:"changed_fields"`
	Snapshot      map[string]any `json:"snapshot"`
	ActorUID      int64          `json:"actor_uid"`
	AmenderUID    *int64         `json:"amender_uid,omitempty"`
	ProposalID    *int64         `json:"proposal_id,omitempty"`
	Site          string         `json:"site"`
	CreatedAt     time.Time      `json:"created_at"`
	LegacyAction  string         `json:"legacy_action,omitempty"`
	LegacyNote    string         `json:"legacy_note,omitempty"`
	LegacyMinor   bool           `json:"legacy_minor,omitempty"`
	// LegacyID is the migrated row's source galgame_revision id (the old
	// wire's row id) — resolves pre-engine revision-row ids to seqs.
	LegacyID *int64 `json:"legacy_id,omitempty"`
}

// EditRevertResult is the revert reply: the sugar proposal + the produced
// revision (action=reverted; history is never deleted).
type EditRevertResult struct {
	Proposal EditProposal `json:"proposal"`
	Revision EditRevision `json:"revision"`
}

// EditSchemaField is one field of the edit-schema projection: shape + the
// asserted caller's evaluated capabilities (the UI holds zero policy logic).
type EditSchemaField struct {
	Key            string `json:"key"`
	Kind           string `json:"kind"`
	DiffHint       string `json:"diff_hint"`
	Deprecated     bool   `json:"deprecated,omitempty"`
	Locked         bool   `json:"locked"`
	CanPropose     bool   `json:"can_propose"`
	CanReview      bool   `json:"can_review"`
	WouldAutomerge bool   `json:"would_automerge"`
}

type EditSchema struct {
	EntityType string            `json:"entity_type"`
	Fields     []EditSchemaField `json:"fields"`
}

// EditFieldDiff is one field's difference between two revisions.
type EditFieldDiff struct {
	Key      string `json:"key"`
	Kind     string `json:"kind,omitempty"`
	DiffHint string `json:"diff_hint,omitempty"`
	From     any    `json:"from"`
	To       any    `json:"to"`
}

type EditDiff struct {
	FromSeq int             `json:"from_seq"`
	ToSeq   int             `json:"to_seq"`
	Fields  []EditFieldDiff `json:"fields"`
}

// EditCreateResult reports whether the direct-edit sugar landed the patch
// (never on kungal's overlaid fields — automerge=never).
type EditCreateResult struct {
	Proposal EditProposal  `json:"proposal"`
	Merged   bool          `json:"merged"`
	Revision *EditRevision `json:"revision,omitempty"`
}

// EditProposalFilter narrows ListEditProposals (zero values = no filter).
type EditProposalFilter struct {
	EntityType  string
	EntityID    int64
	Site        string
	ProposerUID int64
	Status      string
	Limit       int
}

// EditAPIError is a non-2xx edit-face reply whose message is worth showing
// to the user (a validation 422, a policy 403, a rebase-conflict 409).
type EditAPIError struct {
	Status  int
	Code    int
	Message string
}

func (e *EditAPIError) Error() string {
	return fmt.Sprintf("catalog edit: status=%d code=%d %s", e.Status, e.Code, e.Message)
}

const editBase = "/api/v1/catalog/edit"

// EntityTypeWork is the registry's editable work entity — the target of every
// galgame field edit since the wiki's `galgame.game` entity retired.
//
// The two are the SAME rows: the rekey moved the engine's proposals and
// revisions onto this type and onto registry ids, so an entry's history did not
// restart. What did change is the id space of `entity_id`, which is now a
// registry work id and no longer a gid — every consumer that builds a URL from
// it has to come back through the bridge first, or it links to a different
// game without erroring.
const EntityTypeWork = "catalog.work"

// FieldKeyPrefix guards a pass-through patch: only this family's keys may ride
// the BFF (the engine re-validates each one against its registry anyway).
const FieldKeyPrefix = EntityTypeWork + "."

// ListEditProposals lists proposals newest-first (the review queue and the
// "my proposals" reads).
func (c *Client) ListEditProposals(ctx context.Context, f EditProposalFilter) ([]EditProposal, error) {
	page, err := c.listProposalsPage(ctx, proposalQuery(f))
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func proposalQuery(f EditProposalFilter) url.Values {
	q := url.Values{}
	if f.EntityType != "" {
		q.Set("entity_type", f.EntityType)
	}
	if f.EntityID > 0 {
		q.Set("entity_id", strconv.FormatInt(f.EntityID, 10))
	}
	if f.Site != "" {
		q.Set("site", f.Site)
	}
	if f.ProposerUID > 0 {
		q.Set("proposer_uid", strconv.FormatInt(f.ProposerUID, 10))
	}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	return q
}

// CountEditProposals answers "how many proposals match", without paging.
//
// The list face's `total` is counted under the same filter and is independent
// of `limit`, so this asks for one row and reads the count off it. Paging a
// capped list to arrive at the same number is how a contribution threshold
// silently pins itself to the page size.
func (c *Client) CountEditProposals(ctx context.Context, f EditProposalFilter) (int64, error) {
	f.Limit = 1
	page, err := c.listProposalsPage(ctx, proposalQuery(f))
	if err != nil {
		return 0, err
	}
	return page.Total, nil
}

type proposalListPage struct {
	Items []EditProposal `json:"items"`
	Total int64          `json:"total"`
}

func (c *Client) listProposalsPage(ctx context.Context, q url.Values) (*proposalListPage, error) {
	return editGet[proposalListPage](ctx, c, editBase+"/proposals?"+q.Encode())
}

// ListEditRevisions reads an entity's revision log, newest-first.
func (c *Client) ListEditRevisions(ctx context.Context, entityType string, entityID int64, limit int) ([]EditRevision, error) {
	q := url.Values{}
	q.Set("entity_type", entityType)
	q.Set("entity_id", strconv.FormatInt(entityID, 10))
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	data, err := editGet[struct {
		Items []EditRevision `json:"items"`
	}](ctx, c, editBase+"/revisions?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return data.Items, nil
}

// DiffEditRevisions reads the field-level diff between two revisions.
func (c *Client) DiffEditRevisions(ctx context.Context, entityType string, entityID int64, fromSeq, toSeq int) (*EditDiff, error) {
	q := url.Values{}
	q.Set("entity_type", entityType)
	q.Set("entity_id", strconv.FormatInt(entityID, 10))
	q.Set("from_seq", strconv.Itoa(fromSeq))
	q.Set("to_seq", strconv.Itoa(toSeq))
	return editGet[EditDiff](ctx, c, editBase+"/diff?"+q.Encode())
}

// editGet / editPost mirror getData but keep the envelope's business code +
// message: the edit face's 4xx replies carry actionable reasons the BFF
// surfaces to the user.
func editGet[T any](ctx context.Context, c *Client, path string) (*T, error) {
	return editDo[T](ctx, c, http.MethodGet, path, nil)
}

func editPost[T any](ctx context.Context, c *Client, path string, body any) (*T, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return editDo[T](ctx, c, http.MethodPost, path, raw)
}

func editDo[T any](ctx context.Context, c *Client, method, path string, body []byte) (*T, error) {
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
	req.Header.Set("Authorization", c.basicAuth)

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
