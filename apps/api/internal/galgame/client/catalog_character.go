package client

// The character face — GET /v1/catalog/characters/{id} — behind the 角色 pages.
// `{id}` is a CHARACTER id, the one the 登场角色 roster on every game detail
// page already carries, so the link needs no lookup.
//
// This is the character's OWN record, not the per-work roster projection: the
// roster says "she is in this game, voiced by her", this says who she is and
// every game she has ever been in. The two overlap on purpose — the panel is a
// summary and this is the entity.
//
// What the PUBLIC face publishes is narrower than what the catalog stores. The
// physical-attribute block (gender, 生日, BWH, 血型…) lives only on the
// staff-side face, so it is not merely missing here — it is not ours to render.
// What we do get is the multilingual intro, the VNDB trait set, the identity
// anchors and, include-gated, the appearance list with per-work voice names.
//
// The works list is OFFSET-paged with no total, exactly like a credited name's
// filmography: `next_offset` is present precisely when another page exists, and
// the page must not print a count it was never given.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

// catalogCharacterWorksCap is the face's own offset-list ceiling.
const catalogCharacterWorksCap = 50

// catalogCharacterSpoilerCeiling asks for EVERY trait, up to and including the
// major-spoiler tier.
//
// The reveal is a frontend toggle, not a second round trip: each trait carries
// its own level, so fetching the full set once and withholding the flagged ones
// until asked is both fewer requests and a faster reveal. Nothing above 0 is
// rendered before the reader clicks — see dto.GalgameCharacterTrait.
const catalogCharacterSpoilerCeiling = 2

// CatalogCharacter is one character record with a page of its appearances.
type CatalogCharacter struct {
	ID    int64          `json:"id"`
	Name  catNameBuckets `json:"name"`
	Latin string         `json:"latin"`
	// Image is the BUST portrait and Figure the FULL-BODY 立绘, both as
	// complete CDN URLs, either possibly empty. They are DIFFERENT ASSETS and
	// neither is a fallback for the other: the bust was cover-cropped to
	// 256×360 upstream, the figure is a whole person standing on a flat field
	// and must keep its own ratio.
	Image  string `json:"image"`
	Figure string `json:"figure"`
	Intros []struct {
		Lang    string `json:"lang"`
		Intro   string `json:"intro"`
		Source  string `json:"source"`
		Machine bool   `json:"machine"`
	} `json:"intros"`
	// Traits are VNDB's character trait vocabulary, English as published (the
	// catalog has no zh localization for it yet). `Sexual` marks the
	// sexual-family traits, which the catalog serves because it is an R18 face
	// — gating them is the consumer's job, and this one gates on the reader's
	// own SFW switch.
	Traits []struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Group   string `json:"group"`
		Spoiler int    `json:"spoiler"`
		Sexual  bool   `json:"sexual"`
		Lie     bool   `json:"lie"`
	} `json:"traits"`
	Refs  []catRef `json:"refs"`
	Works []struct {
		Work struct {
			ID            int64         `json:"id"`
			DisplayName   string        `json:"display_name"`
			Medium        string        `json:"medium"`
			ContentRating string        `json:"content_rating"`
			ClaimedBy     *catClaimedBy `json:"claimed_by"`
		} `json:"work"`
		// Voices is who voiced her IN THAT WORK — a character recast between a
		// game and its remake has both names, one per row, and flattening them
		// into a single "CV" for the character would erase the recast.
		Voices []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Lang  string `json:"lang"`
			Latin string `json:"latin"`
		} `json:"voices"`
	} `json:"works"`
	// NextOffset is absent on the last page — the only end-of-list signal the
	// face gives.
	NextOffset *int `json:"next_offset"`
}

// CatalogCharacterDetail fetches one character with a page of its appearances.
// found is false for an id the registry does not know. movedTo is non-zero —
// and the record nil — when the id was merged away: characters are a
// merge-capable entity, so the catalog answers 301 + current_id and the caller
// redirects rather than painting the survivor under the dead id.
//
// withWorks is off for the game page's character modal, which is a summary of
// the identity and nothing else: the appearance list is include-gated upstream,
// and asking for it would cost a works-lane hydration on every popup for rows
// that popup never renders.
func (c *GalgameClient) CatalogCharacterDetail(
	ctx context.Context, id int64, limit, offset int, withWorks bool,
) (*CatalogCharacter, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > catalogCharacterWorksCap {
		limit = catalogCharacterWorksCap
	}
	q := url.Values{
		"spoilers": {strconv.Itoa(catalogCharacterSpoilerCeiling)},
		"limit":    {strconv.Itoa(limit)},
		"offset":   {strconv.Itoa(max(offset, 0))},
	}
	if withWorks {
		q.Set("include", "works")
	}
	// The age gate stays open for the same reason the credited-name face opens
	// it: 94.5% of the registry is r18, and closing it here would not filter a
	// character page's adult material, it would delete most of the games she is
	// in. The EDITORIAL gate still applies to the work rows, which are hydrated
	// through the gated works lane, and the sexual-family traits it also
	// unlocks are dropped again in the service for a SFW reader.
	openPopulation(q)

	status, env, appErr := c.getV1Envelope(ctx, "/catalog/characters/"+strconv.FormatInt(id, 10), q)
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
			// A 301 we cannot read is a miss, not a 500: the character really
			// is gone from this id either way.
			return nil, false, 0, nil
		}
		return nil, false, moved.CurrentID, nil
	case env.Code != 0:
		return nil, false, 0, errors.New(env.Code, env.Message, status)
	}
	var ch CatalogCharacter
	if err := json.Unmarshal(env.Data, &ch); err != nil {
		return nil, false, 0, errors.ErrInternal("解析 Catalog 角色详情响应失败")
	}
	return &ch, true, 0, nil
}
