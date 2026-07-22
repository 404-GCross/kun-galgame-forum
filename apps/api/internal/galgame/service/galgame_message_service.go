package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
)

// GalgameMessageService exposes user-facing reads of the galgame message stream
// (GET /messages/mine), the admin queue (GET /admin/galgame/messages), and
// the kungal-local read-state cursor. The actual messages are owned by
// galgame — kungal proxies them through with the user's Bearer attached.
type GalgameMessageService struct {
	galgameClient *client.GalgameClient
	repo          *repository.GalgameMessageRepository
}

func NewGalgameMessageService(
	galgameClient *client.GalgameClient,
	repo *repository.GalgameMessageRepository,
) *GalgameMessageService {
	return &GalgameMessageService{galgameClient: galgameClient, repo: repo}
}

// MessagesMine proxies the user's notification feed from galgame. galgame resolves
// the Bearer's JWT.userID and returns messages where target_user_id matches.
func (s *GalgameMessageService) MessagesMine(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.galgameClient.GetWithToken(ctx, "/galgame/messages/mine", token, query)
}

// AdminMessages proxies the admin moderation queue. The handler must have
// already checked RequireModerator() — this is a thin pass-through and does
// not re-validate the caller's roles.
func (s *GalgameMessageService) AdminMessages(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.galgameClient.GetWithToken(ctx, "/admin/galgame/messages", token, query)
}

// GetReadState returns the user's last-read marker (0 if never set).
func (s *GalgameMessageService) GetReadState(userID int) (int64, error) {
	row, err := s.repo.FindOrZero(userID)
	if err != nil {
		return 0, err
	}
	return row.LastReadMessageID, nil
}

// SetReadState advances the user's last-read marker. The repo applies a
// GREATEST() so concurrent calls / stale tabs can't rewind it.
func (s *GalgameMessageService) SetReadState(userID int, lastReadID int64) error {
	return s.repo.UpsertForward(userID, lastReadID)
}
