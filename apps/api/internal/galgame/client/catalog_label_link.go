package client

import (
	"context"
	"strconv"
	"sync"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

const LabelLinkOfficialSite = "official_site"

func PrimaryLabelLink(l *CatalogLabelDetail) string {
	for _, link := range l.Links {
		if link.Source == LabelLinkOfficialSite {
			return link.URL
		}
	}
	return ""
}

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
