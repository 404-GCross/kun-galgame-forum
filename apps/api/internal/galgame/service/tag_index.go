package service

import (
	"context"
	"net/url"
	"sort"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

const (
	tagIndexTTL     = 10 * time.Minute
	tagIndexPageCap = 40
)

type indexedTag struct {
	item   dto.TagListItem
	sexual bool
}

func (s *TagService) indexRows(ctx context.Context) ([]indexedTag, *errors.AppError) {
	return s.index.get(ctx, tagIndexTTL, s.buildIndex)
}

func (s *TagService) buildIndex(ctx context.Context) ([]indexedTag, *errors.AppError) {
	base := client.OpenPopulation(url.Values{"has_works": {"1"}, "limit": {"100"}})
	rows := make([]indexedTag, 0, 2000)
	cursor := ""
	for page := 0; page < tagIndexPageCap; page++ {
		q := url.Values{}
		for k, v := range base {
			q[k] = v
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.galgameClient.CatalogTaxonomyList(ctx, "tags", q)
		if appErr != nil {
			return nil, appErr
		}
		for _, t := range res.Items {
			if t.Tier == client.TagTierHidden {
				continue
			}
			rows = append(rows, indexedTag{
				item: dto.TagListItem{
					ID: int(t.ID), Name: t.VocabularyLabel(),
					Category: tagCategory(t.Kind, t.Sexual), GalgameCount: t.WorkCount,
				},
				sexual: t.Sexual,
			})
		}
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = *res.NextCursor
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].item.GalgameCount != rows[j].item.GalgameCount {
			return rows[i].item.GalgameCount > rows[j].item.GalgameCount
		}
		return rows[i].item.Name < rows[j].item.Name
	})
	return rows, nil
}

func (s *TagService) sexualByID(ctx context.Context, ids []int) (sexual map[int]bool, missing []int) {
	sexual = make(map[int]bool, len(ids))
	rows, appErr := s.indexRows(ctx)
	if appErr != nil {
		return sexual, ids
	}
	want := make(map[int]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	seen := make(map[int]bool, len(ids))
	for _, r := range rows {
		if want[r.item.ID] {
			sexual[r.item.ID] = r.sexual
			seen[r.item.ID] = true
		}
	}
	for _, id := range ids {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	return sexual, missing
}
