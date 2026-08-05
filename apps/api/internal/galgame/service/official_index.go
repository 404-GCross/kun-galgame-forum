package service

// The 会社 browse ORDER.
//
// "Most games first" is the only order this list has ever wanted, and it is the
// one the catalog cannot serve: the label lane is keyset-paged by id ASC and
// takes no sort parameter (kind / has_works are the whole filter vocabulary),
// so id ASC is really "whichever maker was imported first" — an order that
// tells a reader nothing.
//
// Sorting by work_count therefore has to happen over the WHOLE vocabulary,
// which is why this file exists. The walk is ~2.8k rows on the dev registry and
// ~22k live, at the face's 100-row ceiling, so it is one burst of upstream
// calls rather than one per reader: built at most once per TTL, and every page
// after that is a slice of an ordered array. That also RETIRES the deep-page
// walk the pager used to pay for — reaching kungal page 10 cost ten upstream
// calls before, and costs none now.
//
// Staleness is deliberate. A refresh serves the previous order and rebuilds
// behind the reader, because a browse order half an hour out of date is a far
// better answer than a reader waiting on the whole walk. Only the first request
// after a cold start pays for it, and work counts move slowly enough that no
// one can see the lag.

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

const (
	// officialIndexTTL — how long an order is served before a rebuild is kicked
	// off behind the next reader.
	officialIndexTTL = 30 * time.Minute
	// officialIndexPageSize is the face's own ceiling for the browse lane.
	officialIndexPageSize = 100
	// officialIndexCap stops a walk that would never end (a cursor lane that
	// stopped advancing upstream). Well above the live vocabulary; if it ever
	// binds, the page is short rather than the process wedged.
	officialIndexCap = 200000
	// officialRefreshTimeout bounds the background rebuild. It runs on its own
	// context — a reader who happened to trigger it must never be able to
	// cancel it by navigating away.
	officialRefreshTimeout = 3 * time.Minute
)

// officialIndex is one fully ordered 会社 vocabulary.
type officialIndex struct {
	items   []dto.OfficialListItem
	builtAt time.Time
}

// officialIndexCache holds one index per `kind` filter (the empty key being the
// unfiltered browse list, which is the only one the frontend asks for today).
type officialIndexCache struct {
	mu       sync.RWMutex
	byKind   map[string]*officialIndex
	building map[string]struct{}
}

func newOfficialIndexCache() *officialIndexCache {
	return &officialIndexCache{
		byKind:   map[string]*officialIndex{},
		building: map[string]struct{}{},
	}
}

func (c *officialIndexCache) get(kind string) *officialIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.byKind[kind]
}

func (c *officialIndexCache) put(kind string, idx *officialIndex) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byKind[kind] = idx
}

// claim reserves the right to rebuild `kind`, so a stale index read by fifty
// readers at once triggers one walk rather than fifty.
func (c *officialIndexCache) claim(kind string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, busy := c.building[kind]; busy {
		return false
	}
	c.building[kind] = struct{}{}
	return true
}

func (c *officialIndexCache) release(kind string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.building, kind)
}

// page serves one 1-based page of the ordered vocabulary, building or
// refreshing the index as needed. The total is the index's own length, which is
// exact rather than upstream's per-page estimate — the whole set is in hand.
func (s *OfficialService) page(
	ctx context.Context, kind string, page, limit int,
) ([]dto.OfficialListItem, int64, *errors.AppError) {
	idx := s.index.get(kind)
	if idx == nil {
		// Cold: nothing to serve but the walk itself.
		built, appErr := s.buildIndex(ctx, kind)
		if appErr != nil {
			return nil, 0, appErr
		}
		s.index.put(kind, built)
		idx = built
	} else if time.Since(idx.builtAt) > officialIndexTTL {
		s.refreshIndexAsync(kind)
	}
	return sliceOfficialPage(idx.items, page, limit), int64(len(idx.items)), nil
}

func (s *OfficialService) refreshIndexAsync(kind string) {
	if !s.index.claim(kind) {
		return
	}
	go func() {
		defer s.index.release(kind)
		ctx, cancel := context.WithTimeout(context.Background(), officialRefreshTimeout)
		defer cancel()
		// A failed rebuild leaves the previous order in place: the reader keeps
		// getting the list they had, and the next request tries again.
		if built, appErr := s.buildIndex(ctx, kind); appErr == nil {
			s.index.put(kind, built)
		}
	}()
}

// buildIndex walks the whole label lane and orders it.
func (s *OfficialService) buildIndex(ctx context.Context, kind string) (*officialIndex, *errors.AppError) {
	// has_works=1 drops the empty vocabulary. Some 40% of the label词表 is
	// organisations nothing here credits, so browsing it unfiltered was mostly
	// "+ 0" cards; the predicate is the same one upstream counts with, so a
	// listed 会社 always has something to show.
	base := client.OpenPopulation(url.Values{"has_works": {"1"}})
	if kind != "" {
		base.Set("kind", kind)
	}

	items := make([]dto.OfficialListItem, 0, 4096)
	seenCursor := map[string]struct{}{}
	cursor := ""
	for {
		q := url.Values{}
		for k, v := range base {
			q[k] = v
		}
		q.Set("limit", strconv.Itoa(officialIndexPageSize))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.galgameClient.CatalogTaxonomyList(ctx, "labels", q)
		if appErr != nil {
			return nil, appErr
		}
		for _, o := range res.Items {
			items = append(items, s.officialRow(o))
		}
		if res.NextCursor == nil || *res.NextCursor == "" || len(items) >= officialIndexCap {
			break
		}
		cursor = *res.NextCursor
		// A cursor that repeats is a lane that stopped advancing; stop rather
		// than walk it forever.
		if _, seen := seenCursor[cursor]; seen {
			break
		}
		seenCursor[cursor] = struct{}{}
	}

	sortOfficialsByCount(items)
	return &officialIndex{items: items, builtAt: time.Now()}, nil
}

// sortOfficialsByCount orders the browse list: count first, then name — so the
// long tail of one-game makers is at least alphabetical instead of arbitrary,
// and the order is stable across rebuilds (two makers with the same count can
// never swap places between one walk and the next).
func sortOfficialsByCount(items []dto.OfficialListItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].GalgameCount != items[j].GalgameCount {
			return items[i].GalgameCount > items[j].GalgameCount
		}
		return items[i].Name < items[j].Name
	})
}

// officialRow projects one catalog label onto a browse card.
func (s *OfficialService) officialRow(o client.CatalogTaxonomyItem) dto.OfficialListItem {
	return dto.OfficialListItem{
		ID:   int(o.ID),
		Name: o.Label(),
		// The browse row is identity + count; the maker's website lives on the
		// record (links[]), which the detail page fetches.
		Category: o.Kind,
		// The one image the browse row does carry. Resolved here rather than
		// shipped as a hash so the card can render it straight.
		Logo:         s.galgameClient.ImageURLFromHash(o.LogoHash),
		Alias:        emptyStrSliceIfNil(o.Aliases),
		GalgameCount: o.WorkCount,
	}
}

func sliceOfficialPage(items []dto.OfficialListItem, page, limit int) []dto.OfficialListItem {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	start := (page - 1) * limit
	if start >= len(items) {
		return []dto.OfficialListItem{}
	}
	end := min(start+limit, len(items))
	return items[start:end]
}
