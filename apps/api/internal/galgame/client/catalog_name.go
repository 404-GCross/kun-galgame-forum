package client

// The credited-name face — GET /v1/catalog/names/{id} — behind the 制作人员
// pages. `{id}` is a CREDIT-NAME id, the thing the staff panel already carries.
//
// A name is not a person, and the distinction is load-bearing rather than
// pedantic. 水城新人 and 獅子王院みづき are two names one human signs work under;
// the registry mints a person only once the evidence supports it, and it
// publishes the link (`person_id` + `siblings`) only when that link is public.
// A name whose link is hidden reads here as a standalone identity — that is the
// registry's visibility policy, not a gap, and the page must not try to defeat
// it by guessing from spellings.
//
// The credits list is OFFSET-paged and publishes no total: `next_offset` is
// present exactly when another page exists. Half the credited names in the
// registry have one work and 97.7% have fifty or fewer, so one page is the
// whole answer for almost everyone — but the page must not print a total it
// was never given.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

// catalogNameCreditsCap is the face's own offset-list ceiling.
const catalogNameCreditsCap = 50

// catNameBuckets is the face's per-script name split. It is the shape on the
// record itself AND on every sibling — a sibling name is not a bare string, and
// decoding it as one fails the whole response for exactly the people who have
// siblings, which is to say the ones the page is most interesting for.
type catNameBuckets struct {
	JA    string `json:"ja"`
	ZH    string `json:"zh"`
	Other string `json:"other"`
}

// pick returns the one form to render, Japanese first: this is a Japanese
// medium and the ja bucket is what the person actually signs with.
func (b catNameBuckets) pick() string {
	for _, candidate := range []string{b.JA, b.ZH, b.Other} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// CatalogName is one credited identity with its work list.
type CatalogName struct {
	ID       int64          `json:"id"`
	Name     catNameBuckets `json:"name"`
	Latin    string         `json:"latin"`
	PersonID int64          `json:"person_id"`
	// The five fields below describe the PERSON, not the name, and therefore
	// ride the same public-link gate `person_id` / `siblings` do: the registry
	// zeroes every one of them for a name whose link is private. So an absent
	// photo here means "nothing published about this person" — never "the
	// person has no photo" — and the page must not fill the gap by guessing.
	//
	// All five are additive: a catalog that predates them decodes as a name
	// with no person data, which renders exactly like a hidden link.

	// PhotoHash is a bare content hash in image_service's catalog scope, "" for
	// none. Resolve it with GalgameClient.ImageURLFromHash (the banner walker
	// does not fire on it — it carries no `sort_order` sibling), the same way
	// a 会社 logo is resolved.
	PhotoHash string `json:"photo_hash"`
	// Gender: 1 = male, 2 = female, nil = unknown. nil rather than 0 because
	// "the registry does not know" is a different answer from any code.
	Gender *int `json:"gender"`
	// Birth* is a FUZZY date whose precision is self-expressed by which parts
	// are present: a year alone, a year and a month, or a month and a day with
	// no year at all are each a complete answer the registry can stand behind.
	// Any part is independently nil, so consumers must format from what is
	// there rather than assembling a calendar date.
	BirthY *int `json:"birth_y"`
	BirthM *int `json:"birth_m"`
	BirthD *int `json:"birth_d"`
	// Siblings are the other names the SAME person signs under — public links
	// only, so an empty list means "none published", never "none exist".
	Siblings []struct {
		ID    int64          `json:"id"`
		Name  catNameBuckets `json:"name"`
		Latin string         `json:"latin"`
	} `json:"siblings"`
	Intros []struct {
		Lang   string `json:"lang"`
		Intro  string `json:"intro"`
		Source string `json:"source"`
	} `json:"intros"`
	Refs    []catRef `json:"refs"`
	Credits []struct {
		Work struct {
			ID            int64         `json:"id"`
			DisplayName   string        `json:"display_name"`
			Medium        string        `json:"medium"`
			ContentRating string        `json:"content_rating"`
			ClaimedBy     *catClaimedBy `json:"claimed_by"`
		} `json:"work"`
		// Roles carries one entry per credit, so a voice actor appears once per
		// character voiced (plus, often, a bare entry with no character at all).
		// Folding them is the caller's job — see the staff service.
		Roles []struct {
			RoleKey     string `json:"role_key"`
			RoleName    string `json:"role_name"`
			CharacterID int64  `json:"character_id"`
			Character   string `json:"character"`
		} `json:"roles"`
	} `json:"credits"`
	// NextOffset is absent on the last page — the only end-of-list signal the
	// face gives.
	NextOffset *int `json:"next_offset"`
}

// CatalogNameDetail fetches one credited name with a page of its credits.
// found is false for an id the registry does not know. movedTo is non-zero —
// and the record nil — when the id was merged away (wave 171's name fold made
// that a live event): the catalog answers 301 + current_id, and the caller
// redirects rather than painting the survivor under the dead id.
func (c *GalgameClient) CatalogNameDetail(
	ctx context.Context, id int64, limit, offset int,
) (*CatalogName, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > catalogNameCreditsCap {
		limit = catalogNameCreditsCap
	}
	q := url.Values{
		"include": {"credits"},
		"limit":   {strconv.Itoa(limit)},
		"offset":  {strconv.Itoa(max(offset, 0))},
	}
	// The age gate stays open for the same reason every other kungal read lane
	// opens it: 94.5% of the registry is r18, and closing it here would not
	// filter a staff page's adult material, it would delete most of the
	// person's career. The EDITORIAL gate still applies to the work rows, which
	// are hydrated through the gated works lane.
	openPopulation(q)

	status, env, appErr := c.getV1Envelope(ctx, "/catalog/names/"+strconv.FormatInt(id, 10), q)
	if appErr != nil {
		return nil, false, 0, appErr
	}
	switch {
	case status == http.StatusNotFound:
		return nil, false, 0, nil
	case status == http.StatusMovedPermanently && env.Code == catalogMovedCode:
		var moved struct {
			CurrentID int64 `json:"current_id"`
		}
		if err := json.Unmarshal(env.Data, &moved); err != nil || moved.CurrentID == 0 {
			// A 301 we cannot read is a miss, not a 500: the name really is
			// gone from this id either way.
			return nil, false, 0, nil
		}
		return nil, false, moved.CurrentID, nil
	case env.Code != 0:
		return nil, false, 0, errors.New(env.Code, env.Message, status)
	}
	var n CatalogName
	if err := json.Unmarshal(env.Data, &n); err != nil {
		return nil, false, 0, errors.ErrInternal("解析 Catalog 名义详情响应失败")
	}
	return &n, true, 0, nil
}

// CatalogRowsByCatalogIDs hydrates a filmography straight from catalog ids —
// the staff page's work list arrives already identified, with no kungal gid to
// resolve first (most of a career is works this forum has never ingested).
//
// It routes through the works LANE rather than fetching each record, which is
// also where the editorial gate lives: a row the gate drops is simply absent
// from the returned map, and the caller renders a shorter filmography instead
// of a game the site refuses to show anywhere else.
func (c *GalgameClient) CatalogRowsByCatalogIDs(
	ctx context.Context, ids []int64, isSFW bool,
) (map[int64]CatalogWorkListItem, *errors.AppError) {
	rows, appErr := c.worksByCatalogIDs(ctx, ids, catalogBriefInclude, contentLimitFor(isSFW))
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int64]CatalogWorkListItem, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

// PickCatalogName renders one credited name, Japanese first — see
// catNameBuckets.pick. latin is the last resort, for a western name the
// registry files under no script bucket at all.
func PickCatalogName(buckets catNameBuckets, latin string) string {
	if name := buckets.pick(); name != "" {
		return name
	}
	return latin
}
