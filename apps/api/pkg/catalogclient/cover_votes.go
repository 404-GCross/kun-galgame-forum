package catalogclient

// The READ half of the best-cover vote facet (wave 175 upstream, consumed in
// 176). It reads over S2S — Basic, not the user's token — because the tallies
// are public information the page renders for anonymous visitors too; only the
// `voted` flag is per-viewer, and the viewer is named by the ?uid= parameter
// rather than by a credential.
//
// Why this exists at all, given the detail page already fetches the work: the
// PUBLIC face (/v1/catalog/works/{id}, what internal/galgame/client reads)
// publishes covers WITHOUT their row ids and without the tallies. The vote ops
// address a cover BY ROW ID, so a page fed only by the public face cannot name
// the thing it is voting on. Until the public face carries them, the ids and
// counts come from the S2S bundle, joined onto the rendered covers by image
// hash — the one key both faces publish.

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// CoverTally is one cover's vote projection, keyed by the image hash the
// public face also publishes.
type CoverTally struct {
	// ID is the catalog_work_cover row id — the address the vote ops take.
	ID        int64  `json:"id"`
	ImageHash string `json:"image_hash"`
	VoteCount int    `json:"vote_count"`
	// Voted answers only when a viewer uid was given; it is absent otherwise
	// (upstream omits it), which decodes to false — the same thing "this viewer
	// has not voted" means for a rendering client.
	Voted bool `json:"voted"`
}

// WorkCoverVotes reads one work's covers with their vote tallies. viewerUID 0
// means "nobody asking": the counts still come back, the voted flags do not.
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
