package client

import (
	"context"
	"strconv"
	"sync"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// The 会社's 官网 on a work's detail page.
//
// A work detail's labels[] rows carry identity and counts but no links — the
// catalog publishes a label's web presence on the LABEL record, not on every
// attribution edge that points at it. So the detail projection left Link empty
// and every galgame page said 暂无官网, including the ones whose maker has had
// a homepage on its own 会社 page all along. This resolves the missing half.
//
// Labels per work are few (one to four), the fetches run together, and the
// answers are memoized, so a page pays one upstream call per distinct maker
// per TTL window and usually none.

// PrimaryLabelLink picks a maker's primary web presence: the official site if
// the catalog marks one, otherwise the first link it carries (identity anchors
// live in refs[], never in links[], so anything here is a real page).
func PrimaryLabelLink(l *CatalogLabelDetail) string {
	for _, link := range l.Links {
		if link.Source == "official" || link.Source == "homepage" {
			return link.URL
		}
	}
	if len(l.Links) > 0 {
		return l.Links[0].URL
	}
	return ""
}

// HydrateOfficialLinks fills in each 会社's 官网 on a detail projection.
//
// Best-effort by design: a label that 404s, moved, or simply has no site keeps
// an empty Link, and an upstream hiccup costs the page its 官网 links rather
// than the page itself.
func (c *GalgameClient) HydrateOfficialLinks(
	ctx context.Context,
	g *dto.NextMoeGalgameDetailFull,
) {
	if len(g.Official) == 0 {
		return
	}
	ids := make([]int, 0, len(g.Official))
	for _, rel := range g.Official {
		if rel.Official.ID > 0 {
			ids = append(ids, rel.Official.ID)
		}
	}
	links, appErr := cachedBatch(
		&c.labelLinkMu, c.labelLinkCache, ids, false,
		func(missing []int) (map[int]string, *errors.AppError) {
			return c.fetchLabelLinks(ctx, missing), nil
		},
	)
	if appErr != nil {
		return
	}
	for i := range g.Official {
		if link, ok := links[g.Official[i].Official.ID]; ok {
			g.Official[i].Official.Link = link
		}
	}
}

// fetchLabelLinks resolves label ids → primary link, concurrently. Ids that
// resolve to no link are absent from the result, which cachedBatch remembers as
// a negative so a maker without a site isn't re-asked on every page view.
func (c *GalgameClient) fetchLabelLinks(ctx context.Context, ids []int) map[int]string {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out = make(map[int]string, len(ids))
	)
	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rec, found, _, appErr := c.CatalogLabel(ctx, strconv.Itoa(id))
			if appErr != nil || !found {
				return
			}
			link := PrimaryLabelLink(rec)
			if link == "" {
				return
			}
			mu.Lock()
			out[id] = link
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return out
}
