package service

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// TagService serves the canonical-tag browse / detail / multi-select lanes off
// the catalog taxonomy face.
//
// The tag vocabulary changed owner in this wave, and two product facts follow
// from that (both ruled in refs/proj/126 P2):
//
//   - The wiki's content/sexual/technical CATEGORY axis and its 8,700 tag
//     ALIASES did not migrate — the canonical vocabulary has a content|meta
//     `kind` and a tier instead. Category is therefore reconstructed only where
//     it still drives behaviour (the SFW view drops sexual tags); aliases are
//     gone.
//   - The tag DAG did not migrate either, so "contains" mode can no longer
//     expand a tag into its descendants. It degrades to a flat multi-tag AND,
//     which the catalog now expresses natively (ONE tag_id parameter carrying
//     up to ten comma-separated ids).
type TagService struct {
	galgameClient *client.GalgameClient
	// enricher hydrates the multi-tag results with kungal-local data.
	enricher *GalgameEnricher
	// galgameSvc runs the shared local filter/sort/paginate + hydration flow
	// over a tag's member ids for the detail page. See GetDetail.
	galgameSvc *GalgameService
}

func NewTagService(galgameClient *client.GalgameClient, enricher *GalgameEnricher, galgameSvc *GalgameService) *TagService {
	return &TagService{galgameClient: galgameClient, enricher: enricher, galgameSvc: galgameSvc}
}

// TagMultiPage is the enriched response for GET /galgame-tag/multi.
type TagMultiPage struct {
	Galgames []dto.GalgameCard `json:"galgames"`
	Total    int64             `json:"total"`
}

// taxonomyMemberPageCap bounds the member-id walk behind an entity detail page.
// At 100 ids per upstream page this admits 20,000 members, more than the
// largest real term carries; a hypothetical bigger one truncates rather than
// fanning out without limit on a request path.
const taxonomyMemberPageCap = 200

// maxTagFilterIDs mirrors the catalog works-search contract: tag_id carries up
// to TEN comma-separated ids that are ANDed, and an eleventh is a 400. The
// selector upstream of this lane sets no limit of its own, so the list is
// truncated here — an over-eager selection then returns a slightly wider page
// than asked for instead of failing the whole view.
const maxTagFilterIDs = 10

// CatalogCardInclude is the include= vocabulary a kungal CARD renders: the four
// localized names and the two cover slots. Cards show no intro and no maker
// chips, so asking for those blocks would be paid-for-and-discarded work.
// Exported because the search package renders the same card.
const CatalogCardInclude = "names,covers"

// tagCategory renders a canonical tag's category chip. The wiki's three-value
// axis is gone (P2), so the only distinction reconstructed here is the one the
// SFW view acts on; everything else shows the catalog's own kind.
//
// The adult bucket comes from the tag's OWN `sexual` flag. It used to be
// proxied off tier=="hidden", from before the catalog served a real per-tag
// bool: the proxy conflated two unrelated axes, calling every junk term adult
// and — far worse — calling every adult term in the core tier safe. The two
// axes are now read separately: `sexual` decides the category (and the SFW
// gate), `hidden` decides whether the tag is listed at all (see
// client.TagTierHidden), because upstream is moving to that tier as its
// junk/do-not-display marker.
//
// This is the same rule client.catalogTagCategory already applies to a work's
// tag chips; the two deliberately stay separate functions because they read
// different upstream shapes, not because they mean different things.
func tagCategory(kind string, sexual bool) string {
	if sexual {
		return "sexual"
	}
	return kind
}

// Search — GET /galgame-tag/search
//
// Backed by the catalog entity search's tags index. The hit shape is
// identity-only, so this answers in the search shape: a browse row would have
// carried a category and a count the search does not know, and the shared card
// rendered them — every hit said "+ 0" however many games it had. The count is
// one click away on the detail page.
//
// Gated exactly like the browse list, which this face had simply never been:
// it ran the OPEN population with no filter at all, so typing two characters
// offered a SFW visitor the adult vocabulary the browse page next door has
// always withheld — and the picker feeds the multi-tag filter, so the leak was
// one click from a result set too. Hidden-tier tags are dropped for everyone
// (they are junk, not content), adult tags only for SFW callers.
func (s *TagService) Search(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) ([]dto.TaxonomySearchItem, *errors.AppError) {
	hits, appErr := s.galgameClient.CatalogEntitySearch(ctx, "tags",
		rawQuery.Get("q"), atoiOr(rawQuery.Get("limit"), 20))
	if appErr != nil {
		return nil, appErr
	}

	// Tier rides the hit; `sexual` does not, so it takes the memoized detail
	// lookup — and only when there is a gate to apply.
	kept := make([]client.CatalogEntityHit, 0, len(hits))
	ids := make([]int, 0, len(hits))
	for _, h := range hits {
		if h.Tier == client.TagTierHidden {
			continue
		}
		kept = append(kept, h)
		ids = append(ids, int(h.ID))
	}
	sexual := map[int]bool{}
	if isSFW && len(ids) > 0 {
		sexual = s.galgameClient.CatalogSexualTagIDs(ctx, ids)
	}

	items := make([]dto.TaxonomySearchItem, 0, len(kept))
	for _, h := range kept {
		if sexual[int(h.ID)] {
			continue
		}
		items = append(items, dto.TaxonomySearchItem{ID: int(h.ID), Name: h.Name})
	}
	return items, nil
}

// GetByMultiTag — GET /galgame-tag/multi
//
// A galgame must carry EVERY selected tag (flat AND), which the catalog works
// lane expresses with ONE comma-separated tag_id filter of up to ten ids.
//
// `mode=contains` used to additionally match each tag's DESCENDANTS, so 科幻
// also admitted 硬科幻. The canonical vocabulary has no DAG, so that expansion
// has no successor and the mode now behaves exactly like exact. It is still
// accepted rather than 400'd: the FE persists it in saved filters, and quietly
// behaving like exact beats breaking a bookmarked query.
//
// Do NOT reintroduce a forum-side approximation of the expansion: matching tag
// names by substring looks equivalent and is not — it expands 恋爱 into
// 无恋爱剧情, the literal opposite, because name similarity is not a semantic
// relation.
func (s *TagService) GetByMultiTag(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*TagMultiPage, *errors.AppError) {
	ids := rawQuery.Get("tagIds")
	if ids == "" {
		ids = rawQuery.Get("tag_ids")
	}

	// The catalog SEARCH lane carries page/limit pagination and the total this
	// view needs; the browse lane is keyset-only. An empty q makes it a
	// filter-only query, which the search face serves.
	q := url.Values{
		"page":  {strconv.Itoa(atoiOr(rawQuery.Get("page"), 1))},
		"limit": {strconv.Itoa(atoiOr(rawQuery.Get("limit"), 24))},
		// Published members only (doc 106 §37). This is a public 词条 member
		// list like the single-tag page, just reached through the search face,
		// and it had NO claim gate at all — so it listed unclaimed registry
		// rows on top of the unpublished claimed ones.
		"claim_state": {"live"},
		"include":     {CatalogCardInclude},
		"sort":        {"released_desc"},
	}
	// ONE comma-joined tag_id, never a repeated parameter. The upstream reads
	// this filter with Fiber's c.Query, which returns only the FIRST occurrence
	// of a repeated key — so repeating it made every multi-select silently
	// degrade to a filter on the first tag alone.
	if selected := splitCSV(ids); len(selected) > 0 {
		if len(selected) > maxTagFilterIDs {
			selected = selected[:maxTagFilterIDs]
		}
		q.Set("tag_id", strings.Join(selected, ","))
	}
	client.ApplyWorksGate(q, isSFW)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, appErr
	}
	return &TagMultiPage{
		Galgames: s.enricher.ToCards(ctx, catalogItemsToNextMoe(res.Items)),
		Total:    res.Total,
	}, nil
}

// GetList — GET /galgame-tag
//
// Server-side paged over the catalog tag browse lane. `total` is the whole
// filtered vocabulary (an identity count, so it does not move with the NSFW
// cookie); each row's work_count IS nsfw-aware, so a count never disagrees with
// the member list the same caller can page.
//
// has_works=1 drops the empty vocabulary — 114 of the 1,744 canonical tags have
// no works at all, and a browse page of "+ 0" chips is a list of dead ends. The
// filter is the same predicate upstream counts with, so "count > 0" and "row
// present" cannot drift, and `total` converges with the rows rather than
// promising pages that filter away to nothing.
func (s *TagService) GetList(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagListPage, *errors.AppError) {
	base := client.OpenPopulation(url.Values{"has_works": {"1"}})
	rows, total, appErr := s.galgameClient.CatalogTaxonomyPageAt(ctx, "tags", base,
		atoiOr(rawQuery.Get("page"), 1), atoiOr(rawQuery.Get("limit"), 100))
	if appErr != nil {
		return nil, appErr
	}
	tags := make([]dto.TagListItem, 0, len(rows))
	for _, t := range rows {
		// Hidden-tier tags leave the browse vocabulary entirely — upstream is
		// making that tier its junk/do-not-display marker, and a page of junk
		// terms is worse than a short page. Like the SFW drop below this runs
		// AFTER pagination (the lane filters on tier equality, not exclusion),
		// so a page can come back under-filled while `total` stays the whole
		// vocabulary. That asymmetry is the list's long-standing contract —
		// consumers page by `total`, never by "was this page full".
		if t.Tier == client.TagTierHidden {
			continue
		}
		category := tagCategory(t.Kind, t.Sexual)
		if isSFW && category == "sexual" {
			continue
		}
		tags = append(tags, dto.TagListItem{
			ID: int(t.ID), Name: t.Label(), Category: category,
			GalgameCount: t.WorkCount,
		})
	}
	return &dto.TagListPage{Tags: tags, Total: total}, nil
}

// GetDetail — GET /galgame-tag/:id (id = a CANONICAL catalog tag id)
//
// Entity detail lists the forum-LOCAL subset of the tag's catalogue, so the
// kungal filters (类型/语言/平台/作品类型) + every sort work. Only the tag's
// metadata comes from upstream; the galgame list is recomputed locally from the
// member ids below.
func (s *TagService) GetDetail(
	ctx context.Context,
	id string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagDetail, *errors.AppError) {
	t, found, appErr := s.galgameClient.CatalogTag(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该标签")
	}

	memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,
		url.Values{"tag_id": {id}}, isSFW, taxonomyMemberPageCap)
	if appErr != nil {
		return nil, appErr
	}
	page, appErr := s.galgameSvc.hydrateListCards(ctx, buildEntityFilter(rawQuery, memberIDs), isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.TagDetail{
		ID:   int(t.ID),
		Name: t.Name,
		// The canonical vocabulary's own axis; the wiki's category axis did not
		// migrate (P2).
		Category: tagCategory(t.Kind, t.Sexual),
		// The tag itself is never hidden by a direct link (the stance this
		// whole method takes: an id someone holds resolves), but a hidden tag
		// is not a page search engines should carry, so the FE noindexes on
		// this flag exactly as it already does on category=="sexual".
		Hidden:      t.Tier == client.TagTierHidden,
		Description: preferredIntro(t.Intros),
		// Tag aliases did not migrate (P2) — the canonical vocabulary has no
		// alias table. Always empty rather than absent so the FE contract holds.
		Alias:   []string{},
		Galgame: listCardsToEntityCards(page.Galgames),
		// Same gated page as the rows — never t.WorkCount (upstream counts the
		// tag's whole catalogue, published or not, forum-local or not).
		GalgameCount: page.Total,
	}, nil
}

// preferredIntro picks the description a Chinese-first UI should show:
// zh → ja → en → whatever exists. The catalog merges each language to its
// winning source upstream, so any row for a language is authoritative.
func preferredIntro(intros []client.CatalogIntro) string {
	for _, want := range []string{"zh", "ja", "en"} {
		for _, in := range intros {
			if strings.HasPrefix(strings.ToLower(in.Lang), want) {
				return in.Intro
			}
		}
	}
	if len(intros) > 0 {
		return intros[0].Intro
	}
	return ""
}

// catalogItemsToNextMoe projects a page of catalog rows onto the enricher's
// item shape, dropping rows the owning product withdrew.
func catalogItemsToNextMoe(items []client.CatalogWorkListItem) []dto.NextMoeGalgameItem {
	out := make([]dto.NextMoeGalgameItem, 0, len(items))
	for i := range items {
		if !client.CatalogItemRenderable(&items[i]) {
			continue
		}
		out = append(out, client.CatalogItemToNextMoeItem(&items[i]))
	}
	return out
}
