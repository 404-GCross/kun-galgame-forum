package service

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"
)

type PurgeService struct {
	repo       *repository.PurgeRepository
	userClient *userclient.Client
	community  *communityclient.Client
}

func NewPurgeService(repo *repository.PurgeRepository, userClient *userclient.Client, community *communityclient.Client) *PurgeService {
	return &PurgeService{repo: repo, userClient: userClient, community: community}
}

func (s *PurgeService) GetUserContentStats(ctx context.Context, userID int) dto.UserContentStats {
	stats := s.repo.CountUserContent(userID)
	if resp, err := s.community.AuthorStats(ctx, []int64{int64(userID)}); err == nil {
		for _, st := range resp.Stats {
			if st.AuthorID == int64(userID) {
				stats.CommunityPosts = st.VisiblePosts
				break
			}
		}
	}
	return stats
}

// Privileged accounts (anyone with the moderation capability) are NOT purgeable:
// their content includes site documentation other users read, and this feature
// exists for spam accounts. Roles come from OAuth, and an OAuth lookup ERROR
// refuses the purge — never purge during an outage when the target's privilege
// cannot be confirmed. A not-found user is a gone normal account and is purgeable.
//
// Order: local transaction first, then the community compliance purge. Both
// sides are idempotent, so a community failure surfaces as an error and the
// admin retries.
func (s *PurgeService) PurgeUserContent(ctx context.Context, operatorID, userID int) (dto.PurgeResult, *errors.AppError) {
	u, found, err := s.userClient.User(ctx, userID)
	if err != nil {
		return dto.PurgeResult{}, errors.ErrInternal("无法核验用户身份, 已中止清除")
	}
	if found && role.CanModerate(u.Roles) {
		return dto.PurgeResult{}, errors.ErrForbidden("不可清除管理员 / 版主用户的内容")
	}

	stats, dbErr := s.repo.PurgeUserContent(userID)
	if dbErr != nil {
		return dto.PurgeResult{}, errors.ErrInternal("清除用户内容失败")
	}

	purged, cErr := s.community.AuthorPurge(ctx, int64(userID))
	if cErr != nil {
		slog.Error("purge: community AuthorPurge failed — local delete done, community pending; admin should retry",
			"operator_id", operatorID, "target_id", userID, "local_total", stats.Total, "error", cErr)
		return dto.PurgeResult{}, errors.ErrInternal("已清除本地内容, 但社区内容清除失败, 请重试")
	}

	slog.Info("purge: user content purged",
		"operator_id", operatorID, "target_id", userID,
		"local_total", stats.Total,
		"community_posts_purged", purged.PostsPurged,
		"community_reactions_deleted", purged.ReactionsDeleted)

	return dto.PurgeResult{
		UserContentStats:          stats,
		CommunityPostsPurged:      purged.PostsPurged,
		CommunityReactionsDeleted: purged.ReactionsDeleted,
	}, nil
}
