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
	"strings"

	"kun-galgame-api/pkg/errors"
)

// catalogNameCreditsCap is the face's own offset-list ceiling.
const catalogNameCreditsCap = 50

// catLocalizedName is one entry of the face's `localized` map: the name this
// entity prefers in ONE locale, keyed upstream by a canonically-cased BCP-47
// tag (`zh-Hans`, `ja`, `pt-BR`).
//
// This replaces the retired `name{ja,zh,other}` buckets. Those never held three
// names — the registry files the ONE name of record into whichever bucket its
// own language falls in, so at most one was ever populated and this client's
// old "prefer ja, else zh, else other" scan never actually chose anything. What
// the buckets could not express is the case that matters: a Japanese name WITH
// a Chinese rendering on file still published `{"ja": "…"}` and no `zh` key, so
// the Chinese rendering was unreachable and its absence read as "there is none".
type catLocalizedName struct {
	Value string `json:"value"`
	Kind  string `json:"kind"`
}

// catalogZhLocales / catalogJaLocales are the tags that answer "the Chinese
// form" and "the Japanese form" for this site's own name_zh / name_ja fields,
// most specific first.
//
// The API has no per-request locale — nothing reads Accept-Language anywhere in
// it — so these are fixed site preferences, in the spirit of the chain
// staffIntro and characterIntros already apply to descriptions. They stay
// separate from that chain on purpose: an intro and a name are chosen from
// different tables, and either can move without dragging the other.
var (
	catalogZhLocales = []string{"zh-Hans", "zh", "zh-Hant"}
	catalogJaLocales = []string{"ja"}
)

// CatalogName is one credited identity with its work list.
type CatalogName struct {
	ID int64 `json:"id"`
	// DisplayName is the name of record and is never empty; Lang is its BCP-47
	// tag, "" when the source never declared one. Localized is SPARSE — most
	// names have no entry at all, which is correct rather than missing data
	// (real people's names are largely not translated), so render
	// PickCatalogName and never surface the absence as a blank.
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	PersonID    int64                       `json:"person_id"`
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
	//
	// A sibling carries no `localized` and is not meant to: it is a LINK to a
	// record that publishes its own full map, so a page that needs one in a
	// particular locale follows the id rather than asking for it inline.
	Siblings []struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
		Lang        string `json:"lang"`
		Latin       string `json:"latin"`
	} `json:"siblings"`
	Intros []struct {
		Lang   string `json:"lang"`
		Intro  string `json:"intro"`
		Source string `json:"source"`
	} `json:"intros"`
	Refs []catRef `json:"refs"`
	// Links is the PERSON's web presence (wave 186): homepage / X / pixiv /
	// Ci-en, already rendered to absolute URLs upstream. A DIFFERENT lane from
	// Refs, which are identity anchors carrying a bare external id and no
	// address — the two never overlap, so the page shows both.
	//
	// It rides the same person_id gate the photo and the birthday do: an
	// orphan name, or one whose person link is withheld, yields [].
	Links   []catRelatedLink `json:"links"`
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

// PickCatalogName renders the HEADLINE: the name of record, which is the form
// the person signs with or the character is credited under.
//
// This is byte-identical to what the retired buckets produced. Their scan only
// ever had one non-empty bucket to find, and that bucket held display_name — so
// the headline does not move as part of this migration. A translated rendering
// is a SUBTITLE here, surfaced through CatalogNameByScript, not a replacement
// for the credited name.
//
// latin is the last resort, for the western name the registry holds in no other
// script; display_name is non-empty on every real record.
func PickCatalogName(displayName, latin string) string {
	if displayName != "" {
		return displayName
	}
	return latin
}

// CatalogNameByScript answers the two questions this site's own StaffDetail and
// GalgameCharacterDetail ask beside the headline: the Japanese form and the
// Chinese form. Either is "" when the registry holds no such rendering, which
// the DTOs' omitempty already expects.
//
// The buckets could not answer either one. They filed the SINGLE name of record
// under its own language, so name_zh was non-empty only when the record was
// itself Chinese — which is exactly when it equals the headline, and the staff
// and character pages drop a subtitle part that equals the headline. The slot
// was therefore structurally dead: a Japanese name with a Chinese rendering on
// file published an empty zh, and the rendering that existed was unreachable.
// Reading localized{} is what makes the slot work, and it is why the buckets
// went.
func CatalogNameByScript(localized map[string]catLocalizedName, displayName, lang string) (ja, zh string) {
	switch {
	case strings.HasPrefix(lang, "ja"):
		ja = displayName
	case strings.HasPrefix(lang, "zh"):
		zh = displayName
	}
	if localizedForm := pickLocalized(localized, catalogJaLocales); localizedForm != "" {
		ja = localizedForm
	}
	if localizedForm := pickLocalized(localized, catalogZhLocales); localizedForm != "" {
		zh = localizedForm
	}
	return ja, zh
}

// pickLocalized returns the first rendering the registry has among locales, ""
// when it has none. A missing entry is the norm rather than a gap — most
// credited names are real people's, where a kanji name is not translated at all
// — so the caller must render its absence as nothing, never as an empty slot.
func pickLocalized(localized map[string]catLocalizedName, locales []string) string {
	for _, locale := range locales {
		if entry, ok := localized[locale]; ok && entry.Value != "" {
			return entry.Value
		}
	}
	return ""
}
