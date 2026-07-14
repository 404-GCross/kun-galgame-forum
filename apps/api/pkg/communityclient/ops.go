package communityclient

import (
	"context"
	"net/http"
)

// ResolveComments gets-or-creates the single comments thread for an anchor and
// returns its first page of posts (the read-first-screen; idempotent). For
// galgame comments: anchor_kind=site_game, anchor_id=galgame id text,
// content_rating=all (kungal's comment read face is gated by the site page).
func (c *Client) ResolveComments(ctx context.Context, req ResolveCommentsRequest) (*ThreadWithPosts, error) {
	var out ThreadWithPosts
	err := c.do(ctx, http.MethodPost, "/comments/resolve", req, &out)
	return &out, err
}

// ListPosts returns a thread's posts after a post_number, ascending (keyset).
// after/limit empty → community defaults (limit 50, from the top). The primitive
// clamps limit to 100 server-side.
func (c *Client) ListPosts(ctx context.Context, threadID int64, after, limit string) (*PostListResponse, error) {
	var out PostListResponse
	q := query(map[string]string{"after": after, "limit": limit})
	err := c.do(ctx, http.MethodGet, "/threads/"+itoa(threadID)+"/posts"+q, nil, &out)
	return &out, err
}

// Reply appends a post to a thread and returns the created post view.
func (c *Client) Reply(ctx context.Context, threadID int64, req ReplyRequest) (*PostView, error) {
	var out struct {
		Post PostView `json:"post"`
	}
	err := c.do(ctx, http.MethodPost, "/threads/"+itoa(threadID)+"/posts", req, &out)
	return &out.Post, err
}

// EditPost edits a post's body (author, or a moderator via as_moderator;
// re-cooked + edited_at stamped) and returns the updated post view.
func (c *Client) EditPost(ctx context.Context, postID int64, req EditPostRequest) (*PostView, error) {
	var out struct {
		Post PostView `json:"post"`
	}
	err := c.do(ctx, http.MethodPatch, "/posts/"+itoa(postID), req, &out)
	return &out.Post, err
}

// DeletePost tombstones a post (post_number preserved, replies kept). The acting
// user rides as a query param (?author_id=) — the community DELETE takes no body.
// asModerator declares the mod-actor variant (docs/proj/17 decision 3): the
// author match is skipped and the primitive audit-logs the action.
func (c *Client) DeletePost(ctx context.Context, postID, authorID int64, asModerator bool) error {
	q := map[string]string{"author_id": itoa(authorID)}
	if asModerator {
		q["as_moderator"] = "true"
	}
	return c.do(ctx, http.MethodDelete, "/posts/"+itoa(postID)+query(q), nil, nil)
}

// ToggleReaction toggles a reaction on a post; result.Added reports the new
// state, and the result carries the post's author + thread/anchor for the like
// notification fan-out.
func (c *Client) ToggleReaction(ctx context.Context, postID int64, req ReactionToggleRequest) (*ReactionToggleResult, error) {
	var out ReactionToggleResult
	err := c.do(ctx, http.MethodPost, "/posts/"+itoa(postID)+"/reaction", req, &out)
	return &out, err
}

// SubmitFlag reports a post (reputation-weighted; may auto-hide + enqueue into
// the moderation queue).
func (c *Client) SubmitFlag(ctx context.Context, postID int64, req FlagRequest) error {
	return c.do(ctx, http.MethodPost, "/posts/"+itoa(postID)+"/flag", req, nil)
}

// Boost declares a starter boost (veteran/creator/staff) for a user. A boost
// only ever raises the floor; the server is idempotent.
func (c *Client) Boost(ctx context.Context, req SetBoostRequest) (*TrustView, error) {
	var out TrustView
	err := c.do(ctx, http.MethodPost, "/trust/boost", req, &out)
	return &out, err
}
