package service

import (
	"context"
	"slices"

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

func (s *MessageService) localMutedFor(userID int) []string {
	state, err := s.stateRepo.FindByID(userID)
	if err != nil || state == nil {
		return nil
	}
	local, _ := SplitMuted(state.MutedNotificationTypes)
	return local
}

func (s *MessageService) hydrateMessageRows(ctx context.Context, rows []repository.MessageRow) []dto.MessageResponse {
	uids := userclient.CollectIDs(rows, func(r repository.MessageRow) int { return r.SenderID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.MessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.SenderID]
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
	return messages
}

func (s *MessageService) GetMessages(
	ctx context.Context,
	userID int,
	req *dto.ListMessagesRequest,
) (*dto.MessageListResponse, *errors.AppError) {
	rows, total, err := s.messageRepo.FindMessages(
		userID, s.localMutedFor(userID), nil, req.SortOrder, req.Page, req.Limit,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取消息列表失败")
	}
	return &dto.MessageListResponse{Messages: s.hydrateMessageRows(ctx, rows), Total: total}, nil
}

func (s *MessageService) GetMutedMessages(
	ctx context.Context,
	userID int,
	req *dto.ListMessagesRequest,
) (*dto.MessageListResponse, *errors.AppError) {
	empty := &dto.MessageListResponse{Messages: []dto.MessageResponse{}, Total: 0}

	localMuted := s.localMutedFor(userID)
	if len(localMuted) == 0 {
		return empty, nil
	}

	onlyTypes := localMuted
	if req.Type != "" {
		if !slices.Contains(localMuted, req.Type) {
			return empty, nil
		}
		onlyTypes = []string{req.Type}
	}

	rows, total, err := s.messageRepo.FindMessages(
		userID, nil, onlyTypes, req.SortOrder, req.Page, req.Limit,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取消息列表失败")
	}
	return &dto.MessageListResponse{Messages: s.hydrateMessageRows(ctx, rows), Total: total}, nil
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

	cursor, _ := s.messageRepo.GetSystemReadCursor(userID)

	uids := userclient.CollectIDs(rows, func(r repository.SystemMessageRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.SystemMessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		messages = append(messages, dto.SystemMessageResponse{
			ID:      r.ID,
			IsRead:  int64(r.ID) <= cursor,
			Content: r.Content,
			Admin:   dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Created: r.CreatedAt,
		})
	}
	return messages, nil
}

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
	result, err := s.messageRepo.GetNavSummary(userID, s.localMutedFor(userID))
	if err != nil {
		return nil, errors.ErrInternal("获取消息概要失败")
	}
	return result, nil
}
