package catalogclient

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

const (
	workRevisionsPath       = "/api/v1/catalog/edit-revisions/feed"
	workRevisionsAfterParam = "since"
	workRevisionsLimitParam = "limit"
	workRevisionsSiteParam  = "site"
	workRevisionsTypeParam  = "entity_type"
)

type WorkRevisionFeedItem struct {
	ID            int64     `json:"id"`
	ActorUID      int64     `json:"actor_uid"`
	AmenderUID    *int64    `json:"amender_uid"`
	Site          string    `json:"site"`
	ProductWorkID *int64    `json:"product_work_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type WorkRevisionFeedPage struct {
	Items []WorkRevisionFeedItem `json:"items"`
}

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
