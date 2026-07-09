package service

import (
	"context"

	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type MessageService struct {
	messageRepo *repository.MessageRepository
	stateRepo   *userRepo.StateRepository
	userClient  *userclient.Client
}

func NewMessageService(
	messageRepo *repository.MessageRepository,
	stateRepo *userRepo.StateRepository,
	userClient *userclient.Client,
) *MessageService {
	return &MessageService{messageRepo: messageRepo, stateRepo: stateRepo, userClient: userClient}
}

func (s *MessageService) GetMessages(
	ctx context.Context,
	userID int,
	req *dto.ListMessagesRequest,
) (*dto.MessageListResponse, *errors.AppError) {
	rows, total, err := s.messageRepo.FindMessages(
		userID, req.Type, req.SortOrder, req.Page, req.Limit,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取消息列表失败")
	}

	uids := userclient.CollectIDs(rows, func(r repository.MessageRow) int { return r.SenderID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.MessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.SenderID]
		// Hide messages whose sender is banned/deleted.
		if !userclient.IsRenderable(u) {
			continue
		}
		messages = append(messages, dto.MessageResponse{
			ID:         r.ID,
			Sender:     dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ReceiverID: r.ReceiverID,
			Link:       r.Link,
			Content:    r.Content,
			Status:     r.Status,
			Type:       r.Type,
			Created:    r.CreatedAt,
		})
	}

	return &dto.MessageListResponse{Messages: messages, Total: total}, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, userID, messageID int) *errors.AppError {
	if err := s.messageRepo.DeleteByIDAndReceiver(messageID, userID); err != nil {
		return errors.ErrInternal("删除消息失败")
	}
	return nil
}

func (s *MessageService) MarkAllRead(ctx context.Context, userID int) *errors.AppError {
	if err := s.messageRepo.MarkAllRead(userID); err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	return nil
}

func (s *MessageService) GetSystemMessages(ctx context.Context, userID int) ([]dto.SystemMessageResponse, *errors.AppError) {
	rows, err := s.messageRepo.FindSystemMessages()
	if err != nil {
		return nil, errors.ErrInternal("获取系统消息失败")
	}

	// Per-user HWM cursor: every row whose id <= cursor has been read by
	// this user. A missing cursor row returns 0 (everything unread),
	// which matches "I just signed up, show me the backlog."
	cursor, _ := s.messageRepo.GetSystemReadCursor(userID)

	uids := userclient.CollectIDs(rows, func(r repository.SystemMessageRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.SystemMessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		messages = append(messages, dto.SystemMessageResponse{
			ID:     r.ID,
			IsRead: int64(r.ID) <= cursor,
			Content: map[string]string{
				"en-us": r.ContentEnUS,
				"ja-jp": r.ContentJaJP,
				"zh-cn": r.ContentZhCN,
				"zh-tw": r.ContentZhTW,
			},
			Admin:   dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Created: r.CreatedAt,
		})
	}
	return messages, nil
}

// MarkAllSystemRead advances the caller's HWM cursor to MAX(id) so every
// existing broadcast becomes "read" for this user only. Broadcasts
// posted later still appear as unread (id > cursor) until the next call.
func (s *MessageService) MarkAllSystemRead(ctx context.Context, userID int) *errors.AppError {
	maxID, err := s.messageRepo.GetMaxSystemMessageID()
	if err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	if err := s.messageRepo.UpsertSystemReadCursorForward(userID, maxID); err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	return nil
}

func (s *MessageService) GetNavSummary(ctx context.Context, userID int) ([]map[string]any, *errors.AppError) {
	// The user's muted set (migration 053) excludes muted categories from the
	// unread badges so they stay in sync with the top-bar red dot. Only the
	// local message.type mutes matter here (system is non-mutable; chat isn't
	// part of this summary).
	var localMuted []string
	if state, err := s.stateRepo.FindByID(userID); err == nil && state != nil {
		localMuted, _ = SplitMuted(state.MutedNotificationTypes)
	}

	result, err := s.messageRepo.GetNavSummary(userID, localMuted)
	if err != nil {
		return nil, errors.ErrInternal("获取消息概要失败")
	}
	return result, nil
}
