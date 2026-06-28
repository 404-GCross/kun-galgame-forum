package service

import (
	"context"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"
)

type PurgeService struct {
	repo       *repository.PurgeRepository
	userClient *userclient.Client
}

func NewPurgeService(repo *repository.PurgeRepository, userClient *userclient.Client) *PurgeService {
	return &PurgeService{repo: repo, userClient: userClient}
}

// GetUserContentStats previews what a purge would remove.
func (s *PurgeService) GetUserContentStats(userID int) dto.UserContentStats {
	return s.repo.CountUserContent(userID)
}

// PurgeUserContent hard-deletes all of the user's kungal content +
// interactions and returns the breakdown of what was removed.
//
// Guard: privileged accounts (anyone with the moderation capability —
// moderators / admins) are NOT purgeable. Their content includes site
// documentation / changelogs that other users read, and the feature exists for
// spam/ad accounts only. We resolve the TARGET user's roles from OAuth (the
// role source of truth). On an OAuth lookup error we refuse (fail-safe — never
// purge during an outage if we can't confirm the target isn't privileged); a
// not-found user is treated as a gone normal account whose leftover content may
// still be purged.
func (s *PurgeService) PurgeUserContent(ctx context.Context, userID int) (dto.UserContentStats, *errors.AppError) {
	u, found, err := s.userClient.User(ctx, userID)
	if err != nil {
		return dto.UserContentStats{}, errors.ErrInternal("无法核验用户身份, 已中止清除")
	}
	if found && role.CanModerate(u.Roles) {
		return dto.UserContentStats{}, errors.ErrForbidden("不可清除管理员 / 版主用户的内容")
	}

	stats, dbErr := s.repo.PurgeUserContent(userID)
	if dbErr != nil {
		return dto.UserContentStats{}, errors.ErrInternal("清除用户内容失败")
	}
	return stats, nil
}
