package catalogclient

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

// The editing engine's revision log read as a CURSOR FEED over one tenant's
// works — the supply side of kungal's contributor table (wave 180 lane B).
//
// It is deliberately not EditRevisionsSince (revisions.go). That call feeds the
// activity timeline: it is keyed on engine ids and translates entity ids to
// gids with a second round-trip per page. This one asks the registry to do the
// join, because the question is different — "who has edited a work kungal
// claims" — and a per-page id translation would make a full replay of the
// rekeyed wiki-era history (~12.6k revisions) cost one extra call per page for
// an answer the registry already holds.
//
// Every wire name the feed contract owns is a constant below. Lane A ships the
// enrichment in parallel, so a rename on their side is a one-line fix here
// rather than a hunt through the sync.
const (
	// workRevisionsPath is the engine's revision list.
	workRevisionsPath = "/api/v1/catalog/edit/revisions"
	// workRevisionsAfterParam is the exclusive id cursor: strictly-after
	// replay, ascending.
	workRevisionsAfterParam = "after"
	workRevisionsLimitParam = "limit"
	workRevisionsSiteParam  = "site"
	workRevisionsTypeParam  = "entity_type"
)

// WorkRevisionFeedItem is one revision of a catalog work, carrying the claim
// projection kungal needs: `site` says whose work it is and `product_work_id`
// is the gid it is anchored at.
//
// Both editing identities are on the wire and both count as contribution:
// ActorUID filed the change, AmenderUID (null unless a reviewer touched the
// proposal on its way in) shaped it.
//
// ProductWorkID is nullable — a work no product has claimed has no gid, and
// there is nothing local to attribute it to. Never substitute the work id: the
// two key spaces overlap (doc 106 R3).
type WorkRevisionFeedItem struct {
	ID            int64     `json:"id"`
	ActorUID      int64     `json:"actor_uid"`
	AmenderUID    *int64    `json:"amender_uid"`
	Site          string    `json:"site"`
	ProductWorkID *int64    `json:"product_work_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// WorkRevisionFeedPage is one ascending page. Only `items` is read: the caller
// tracks its own cursor as the largest id it applied, which is the value that
// stays correct when a page is only partly ingested.
type WorkRevisionFeedPage struct {
	Items []WorkRevisionFeedItem `json:"items"`
}

// WorkRevisionsAfter reads one page of work revisions strictly after `after`,
// narrowed to one tenant. A page shorter than the limit is the tail.
func (c *Client) WorkRevisionsAfter(
	ctx context.Context,
	after int64,
	limit int,
	site string,
) (*WorkRevisionFeedPage, error) {
	q := url.Values{
		workRevisionsAfterParam: {strconv.FormatInt(after, 10)},
		workRevisionsLimitParam: {strconv.Itoa(limit)},
		workRevisionsTypeParam:  {EntityTypeWork},
	}
	if site != "" {
		q.Set(workRevisionsSiteParam, site)
	}
	data, err := c.getData(ctx, workRevisionsPath, q)
	if err != nil {
		return nil, err
	}
	var page WorkRevisionFeedPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}
	return &page, nil
}
