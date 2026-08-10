package communityclient

import (
	"context"
	"net/http"
)

func (c *Client) ResolveComments(ctx context.Context, req ResolveCommentsRequest) (*ThreadWithPosts, error) {
	var out ThreadWithPosts
	err := c.do(ctx, http.MethodPost, "/comments/resolve", req, &out)
	return &out, err
}

func (c *Client) ListPosts(ctx context.Context, threadID int64, after, limit string) (*PostListResponse, error) {
	var out PostListResponse
	q := query(map[string]string{"after": after, "limit": limit})
	err := c.do(ctx, http.MethodGet, "/threads/"+itoa(threadID)+"/posts"+q, nil, &out)
	return &out, err
}

func (c *Client) Reply(ctx context.Context, threadID int64, req ReplyRequest) (*PostView, error) {
	var out struct {
		Post PostView `json:"post"`
	}
	err := c.do(ctx, http.MethodPost, "/threads/"+itoa(threadID)+"/posts", req, &out)
	return &out.Post, err
}

func (c *Client) EditPost(ctx context.Context, postID int64, req EditPostRequest) (*PostView, error) {
	var out struct {
		Post PostView `json:"post"`
	}
	err := c.do(ctx, http.MethodPatch, "/posts/"+itoa(postID), req, &out)
	return &out.Post, err
}

func (c *Client) DeletePost(ctx context.Context, postID, authorID int64, asModerator bool) error {
	q := map[string]string{"author_id": itoa(authorID)}
	if asModerator {
		q["as_moderator"] = "true"
	}
	return c.do(ctx, http.MethodDelete, "/posts/"+itoa(postID)+query(q), nil, nil)
}

func (c *Client) ToggleReaction(ctx context.Context, postID int64, req ReactionToggleRequest) (*ReactionToggleResult, error) {
	var out ReactionToggleResult
	err := c.do(ctx, http.MethodPost, "/posts/"+itoa(postID)+"/reaction", req, &out)
	return &out, err
}

func (c *Client) SubmitFlag(ctx context.Context, postID int64, req FlagRequest) error {
	return c.do(ctx, http.MethodPost, "/posts/"+itoa(postID)+"/flag", req, nil)
}

func (c *Client) Boost(ctx context.Context, req SetBoostRequest) (*TrustView, error) {
	var out TrustView
	err := c.do(ctx, http.MethodPost, "/trust/boost", req, &out)
	return &out, err
}

func (c *Client) AuthorPosts(ctx context.Context, authorID int64, after string, limit, anchorKind int) (*AuthorPostsResponse, error) {
	var out AuthorPostsResponse
	q := map[string]string{"after": after}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	if anchorKind >= 0 {
		q["anchor_kind"] = itoa(int64(anchorKind))
	}
	err := c.do(ctx, http.MethodGet, "/authors/"+itoa(authorID)+"/posts"+query(q), nil, &out)
	return &out, err
}

func (c *Client) AuthorStats(ctx context.Context, ids []int64) (*AuthorStatsResponse, error) {
	if len(ids) == 0 {
		return &AuthorStatsResponse{Stats: []AuthorStat{}}, nil
	}
	var out AuthorStatsResponse
	err := c.do(ctx, http.MethodGet, "/authors/stats"+query(map[string]string{"ids": joinInt64(ids)}), nil, &out)
	return &out, err
}

func (c *Client) AuthorPurge(ctx context.Context, authorID int64) (*PurgeResult, error) {
	var out PurgeResult
	err := c.do(ctx, http.MethodPost, "/authors/"+itoa(authorID)+"/purge", nil, &out)
	return &out, err
}

func (c *Client) ResolvePosts(ctx context.Context, ids []int64) (*PostsResolveResponse, error) {
	if len(ids) == 0 {
		return &PostsResolveResponse{Posts: []AuthorPostView{}}, nil
	}
	var out PostsResolveResponse
	err := c.do(ctx, http.MethodPost, "/posts/resolve", PostsResolveRequest{IDs: ids}, &out)
	return &out, err
}
