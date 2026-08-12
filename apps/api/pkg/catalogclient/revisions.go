package catalogclient

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

type EditRevisionFeedItem struct {
	ID            int64     `json:"id"`
	EntityFamily  string    `json:"entity_family"`
	EntityType    string    `json:"entity_type"`
	EntityID      int64     `json:"entity_id"`
	Seq           int       `json:"seq"`
	Action        int16     `json:"action"`
	ChangedFields []string  `json:"changed_fields"`
	ActorUID      int64     `json:"actor_uid"`
	AmenderUID    *int64    `json:"amender_uid"`
	ProposalID    *int64    `json:"proposal_id"`
	Site          string    `json:"site"`
	CreatedAt     time.Time `json:"created_at"`
}

const (
	EditActionCreated  int16 = 0
	EditActionMerged   int16 = 1
	EditActionDirect   int16 = 2
	EditActionReverted int16 = 3
)

type EditRevisionFeedPage struct {
	Items     []EditRevisionFeedItem `json:"items"`
	NextSince int64                  `json:"next_since"`
}

func (c *Client) EditRevisionsSince(
	ctx context.Context,
	since int64,
	limit int,
	entityType string,
) (*EditRevisionFeedPage, error) {
	q := url.Values{
		"since": {strconv.FormatInt(since, 10)},
		"limit": {strconv.Itoa(limit)},
	}
	if entityType != "" {
		q.Set("entity_type", entityType)
	}
	data, err := c.getData(ctx, "/api/v1/catalog/edit-revisions/feed", q)
	if err != nil {
		return nil, err
	}
	var page EditRevisionFeedPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}
	return &page, nil
}
