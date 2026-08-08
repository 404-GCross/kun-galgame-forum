package service

import (
	"context"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// OfficialService serves the 会社 browse / search / detail lanes.
//
// The catalog calls this entity a LABEL; the kungal product wording (会社) is
// unchanged, so the rename shows up only in the upstream path and the new URL
// space. Two field-level notes:
//
//   - `category` is now the label KIND (game_brand / publisher / doujin_circle
//     …), a richer and better-defined vocabulary than the wiki's free-text one.
//   - `original` (the wiki's original-language name field) has no catalog
//     counterpart on the public face. It survives only where it is still
//     WRITTEN — the staff edit form, which reads it back off the /api face.
type OfficialService struct {
	galgameClient *client.GalgameClient
	// galgameSvc runs the shared local filter/sort/paginate + hydration flow
	// over the label's member ids (the catalog can't filter by kungal-local
	// resource data). See GetDetail.
	galgameSvc *GalgameService
	// index holds the browse list in "most games first" order — an order the
	// catalog cannot serve, so kungal computes it over the whole vocabulary.
	// See official_index.go.
	index *officialIndexCache
}

func NewOfficialService(galgameClient *client.GalgameClient, galgameSvc *GalgameService) *OfficialService {
	return &OfficialService{
		galgameClient: galgameClient,
		galgameSvc:    galgameSvc,
		index:         newOfficialIndexCache(),
	}
}

// GetList — GET /galgame-official
//
// Ordered by how many games the maker has, which is the only ranking a browse
// list of 22,000 companies can be read with. The catalog's lane is id ASC and
// has no sort parameter, so the order is computed here over the whole
// vocabulary and cached — see official_index.go for the walk and its staleness
// bargain.
func (s *OfficialService) GetList(
	ctx context.Context,
	rawQuery url.Values,
) (*dto.OfficialListPage, *errors.AppError) {
	items, total, appErr := s.page(ctx, rawQuery.Get("kind"),
		atoiOr(rawQuery.Get("page"), 1), atoiOr(rawQuery.Get("limit"), 50))
	if appErr != nil {
		return nil, appErr
	}
	return &dto.OfficialListPage{Officials: items, Total: total}, nil
}

// Search — GET /galgame-official/search
//
// Identity only: the upstream search face returns id + name (+ the label's logo
// hash, which for a maker IS identity) and nothing else, so this answers in the
// search shape rather than dressing a hit up as a browse row whose count,
// category and language would all be zero values.
//
// The frontend does `searchResult.value = res` expecting a bare array, so the
// envelope is unwrapped here and the gateway response shape stays put.
func (s *OfficialService) Search(
	ctx context.Context,
	rawQuery url.Values,
) ([]dto.TaxonomySearchItem, *errors.AppError) {
	hits, appErr := s.galgameClient.CatalogEntitySearch(ctx, "labels",
		rawQuery.Get("q"), atoiOr(rawQuery.Get("limit"), 20))
	if appErr != nil {
		return nil, appErr
	}
	items := make([]dto.TaxonomySearchItem, 0, len(hits))
	for _, o := range hits {
		items = append(items, dto.TaxonomySearchItem{
			ID:   int(o.ID),
			Name: o.Name,
			Logo: s.galgameClient.ImageURLFromHash(o.LogoHash),
		})
	}
	return items, nil
}

// GetDetail — GET /galgame-official/:id (id = a catalog LABEL id)
//
// Entity detail lists the forum-LOCAL subset of the label's catalogue, so the
// kungal filters (类型/语言/平台/作品类型) + every sort work. Only the label's
// metadata comes from upstream; the galgame list is recomputed locally from the
// member ids below.
func (s *OfficialService) GetDetail(
	ctx context.Context,
	id string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.OfficialDetail, *errors.AppError) {
	o, found, movedTo, appErr := s.galgameClient.CatalogLabel(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	// Merged away upstream: hand the page the survivor's id and nothing else,
	// so it can 301 in one hop instead of rendering a ghost — a dead label used
	// to answer with its old name and an empty catalogue, which reads as a real
	// but empty company page.
	if movedTo != 0 {
		return &dto.OfficialDetail{MovedTo: int(movedTo)}, nil
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该会社")
	}

	memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,
		url.Values{"label_id": {id}}, isSFW, taxonomyMemberPageCap)
	if appErr != nil {
		return nil, appErr
	}
	page, appErr := s.galgameSvc.hydrateListCards(ctx, buildEntityFilter(rawQuery, memberIDs), isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.OfficialDetail{
		ID:   int(o.ID),
		Name: o.DisplayName,
		// The wiki's separate `original` field has no public catalog
		// counterpart; the label's own name IS the original in the common case
		// (a Japanese brand is stored under its Japanese name), and the
		// alternate spellings are in alias[].
		Links:       officialLinks(o.Links),
		Link:        client.PrimaryLabelLink(o),
		Logo:        s.galgameClient.ImageURLFromHash(o.LogoHash),
		Category:    o.Kind,
		Lang:        o.Lang,
		Description: preferredIntro(o.Intros),
		Alias:       emptyStrSliceIfNil(o.Aliases),
		Galgame:     listCardsToEntityCards(page.Galgames),
		// The header count and the pager MUST come from the same gated page the
		// rows do. o.WorkCount (upstream, nsfw-aware) is right there and is the
		// wrong answer: it counts the label's whole catalogue, published or not
		// and forum-local or not, so using it would put a pager on works this
		// page cannot list.
		GalgameCount: page.Total,
	}, nil
}

// GetRelationGraph — GET /galgame-official/:id/relation-graph
//
// Its own endpoint rather than a block on GetDetail: the detail lane is
// refetched on every pagination / filter change of the games grid, and the
// corporate family does not change when the reader turns a page. Paying for a
// graph walk on each of those would be pure waste.
//
// Nothing is recomputed locally here — unlike the games grid, whose rows are
// forum-local, the family tree is entirely upstream data. The only projection
// is the logo hash → CDN URL, resolved exactly as the detail's own logo is.
func (s *OfficialService) GetRelationGraph(ctx context.Context, id string) (*dto.OfficialRelationGraph, *errors.AppError) {
	graph, found, appErr := s.galgameClient.CatalogLabelRelationGraph(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该会社")
	}

	// Always slices, never null: the FE iterates both unguarded.
	out := &dto.OfficialRelationGraph{
		Nodes: make([]dto.OfficialRelationNode, 0, len(graph.Nodes)),
		Edges: make([]dto.OfficialRelationEdge, 0, len(graph.Edges)),
	}
	for _, n := range graph.Nodes {
		out.Nodes = append(out.Nodes, dto.OfficialRelationNode{
			ID:        int(n.ID),
			Name:      n.Name,
			Logo:      s.galgameClient.ImageURLFromHash(n.LogoHash),
			WorkCount: n.WorkCount,
		})
	}
	for _, e := range graph.Edges {
		out.Edges = append(out.Edges, dto.OfficialRelationEdge{
			From:     int(e.From),
			To:       int(e.To),
			Relation: e.Relation,
		})
	}
	return out, nil
}

// officialLinks passes the label's web presences through, each named here
// rather than in the browser — see dto.OfficialLink. Always a slice, never
// null: the FE iterates it unguarded.
func officialLinks(links []client.CatalogLabelLink) []dto.OfficialLink {
	out := make([]dto.OfficialLink, 0, len(links))
	for _, l := range links {
		out = append(out, dto.OfficialLink{
			Source: l.Source,
			Name:   client.LinkDisplayName(l.Source, l.URL),
			URL:    l.URL,
		})
	}
	return out
}

// ResolveLegacyID maps a legacy wiki 会社 id onto its catalog label id.
func (s *OfficialService) ResolveLegacyID(ctx context.Context, wikiID int) (int64, bool, *errors.AppError) {
	return s.galgameClient.LookupWikiLabel(ctx, wikiID)
}
