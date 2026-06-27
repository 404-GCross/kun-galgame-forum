package service

import (
	"net/url"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
)

// buildEntityFilter builds the local list filter for a wiki-entity detail page
// (tag/official/engine) from the request query + the entity's member ids. Only
// the basic filters those pages expose (type/language/platform/sort/page) are
// read; RestrictIDs scopes the whole list to the entity's members so the SAME
// local filter/sort/paginate as /galgame runs over them. A non-nil (even empty)
// RestrictIDs means "restrict to this set" — an entity with no local members
// renders empty, never the whole catalogue (see list_repo).
func buildEntityFilter(q url.Values, restrictIDs []int) model.GalgameListFilter {
	sortOrder := q.Get("sortOrder")
	if sortOrder == "" {
		sortOrder = "desc"
	}
	return model.GalgameListFilter{
		Type:        q.Get("type"),
		Language:    q.Get("language"),
		Platform:    q.Get("platform"),
		SortField:   q.Get("sortField"),
		SortOrder:   sortOrder,
		Page:        atoiOr(q.Get("page"), 1),
		Limit:       atoiOr(q.Get("limit"), 24),
		RestrictIDs: restrictIDs,
	}
}

// listCardsToEntityCards converts the shared /galgame list cards to the entity
// detail card shape (drops the rating fields the entity card doesn't carry),
// preserving the existing entity-page response shape so the FE is unchanged.
func listCardsToEntityCards(cards []dto.GalgameListCard) []dto.GalgameCard {
	out := make([]dto.GalgameCard, len(cards))
	for i, c := range cards {
		out[i] = dto.GalgameCard{
			ID:                       c.ID,
			Name:                     c.Name,
			Banner:                   c.Banner,
			User:                     c.User,
			ContentLimit:             c.ContentLimit,
			View:                     c.View,
			LikeCount:                c.LikeCount,
			ResourceUpdateTime:       c.ResourceUpdateTime,
			Platform:                 c.Platform,
			Language:                 c.Language,
			ReleaseDate:              c.ReleaseDate,
			ReleaseDateTBA:           c.ReleaseDateTBA,
			EffectiveBannerHash:      c.EffectiveBannerHash,
			EffectiveBannerURL:       c.EffectiveBannerURL,
			EffectiveBannerWidth:     c.EffectiveBannerWidth,
			EffectiveBannerHeight:    c.EffectiveBannerHeight,
			EffectiveBannerThumbhash: c.EffectiveBannerThumbhash,
		}
	}
	return out
}
