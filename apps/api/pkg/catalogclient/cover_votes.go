package catalogclient

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

type CoverTally struct {
	ID        int64  `json:"id"`
	ImageHash string `json:"image_hash"`
	VoteCount int    `json:"vote_count"`
	Voted     bool   `json:"voted"`
}

func (c *Client) WorkCoverVotes(ctx context.Context, workID, viewerUID int64) ([]CoverTally, error) {
	q := url.Values{}
	if viewerUID > 0 {
		q.Set("uid", strconv.FormatInt(viewerUID, 10))
	}
	data, err := c.getData(ctx, "/api/v1/catalog/works/"+strconv.FormatInt(workID, 10), q)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Covers []CoverTally `json:"covers"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, ErrUpstream
	}
	return parsed.Covers, nil
}
