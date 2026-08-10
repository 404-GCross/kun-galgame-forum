package service

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/communityclient"
)

type legacyGalgameCommentMap interface {
	FindMapByLegacyID(legacyID int) (*model.GalgameCommentCommunityMap, error)
}

type GalgameCommentEnforcer struct {
	community *communityclient.Client
	posts     legacyGalgameCommentMap
}

func NewGalgameCommentEnforcer(community *communityclient.Client, posts legacyGalgameCommentMap) *GalgameCommentEnforcer {
	return &GalgameCommentEnforcer{community: community, posts: posts}
}

func (e *GalgameCommentEnforcer) Tombstone(ctx context.Context, legacyID int) error {
	m, err := e.posts.FindMapByLegacyID(legacyID)
	if err != nil {
		return err
	}
	if m == nil {
		slog.Warn("trust galgame_comment enforcement: no community map row; no-op", "legacy_id", legacyID)
		return nil
	}
	var authorID int64
	if res, rerr := e.community.ResolvePosts(ctx, []int64{m.PostID}); rerr == nil && len(res.Posts) > 0 {
		authorID = res.Posts[0].Post.AuthorID
	}
	return e.community.DeletePost(ctx, m.PostID, authorID, true)
}

func (e *GalgameCommentEnforcer) AuthorID(ctx context.Context, legacyID int) (int, error) {
	m, err := e.posts.FindMapByLegacyID(legacyID)
	if err != nil || m == nil {
		return 0, nil
	}
	res, err := e.community.ResolvePosts(ctx, []int64{m.PostID})
	if err != nil || len(res.Posts) == 0 {
		return 0, nil
	}
	return int(res.Posts[0].Post.AuthorID), nil
}
