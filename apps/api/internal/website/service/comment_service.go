package service

import (
	"context"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	websiteRepo *repository.WebsiteRepository
	notifier    msgService.Notifier
	userClient  *userclient.Client
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	websiteRepo *repository.WebsiteRepository,
	notifier msgService.Notifier,
	userClient *userclient.Client,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		websiteRepo: websiteRepo,
		notifier:    notifier,
		userClient:  userClient,
	}
}

// ──────────────────────────────────────────
// GetComments — GET /website/:domain/comment
// ──────────────────────────────────────────

// GetComments returns the nested comment tree for a website. Identity is
// hydrated from OAuth via userclient since the repo no longer joins on the
// user table; rows authored by banned users are dropped.
func (s *CommentService) GetComments(ctx context.Context, websiteID int) []*dto.CommentItem {
	rows := s.commentRepo.FindByWebsite(websiteID)

	uids := userclient.CollectIDs(rows, func(r repository.CommentRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	flat := make([]*dto.CommentItem, 0, len(rows))
	idMap := make(map[int]*dto.CommentItem, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		item := &dto.CommentItem{
			ID:        r.ID,
			Content:   r.Content,
			ParentID:  r.ParentID,
			UserID:    r.UserID,
			WebsiteID: websiteID,
			Created:   r.Created,
			Edited:    r.Edited,
			Reply:     []*dto.CommentItem{},
			User: dto.CommentUser{
				ID: u.ID, Name: u.Name, Avatar: u.Avatar,
			},
			TargetUser: nil,
		}
		flat = append(flat, item)
		idMap[r.ID] = item
	}

	var nested []*dto.CommentItem
	for _, item := range flat {
		if item.ParentID != nil {
			parent, ok := idMap[*item.ParentID]
			if !ok {
				// Parent was dropped (banned author) — its reply has no
				// visible thread to attach to and TargetUser would be
				// nil, so the FE would render an orphan as a top-level
				// comment with a dangling "回复 (deleted)" affordance.
				// Drop the reply to keep the website comment tree
				// consistent with the galgame side, which also discards
				// descendants whose root row was filtered.
				continue
			}
			item.TargetUser = parent.User
			parent.Reply = append(parent.Reply, item)
			continue
		}
		nested = append(nested, item)
	}

	if nested == nil {
		nested = []*dto.CommentItem{}
	}
	return nested
}

// ──────────────────────────────────────────
// CreateComment — POST /website/:domain/comment
// ──────────────────────────────────────────

func (s *CommentService) CreateComment(
	ctx context.Context,
	userID int,
	req *dto.CreateCommentRequest,
) (*dto.CommentItem, *errors.AppError) {
	comment := model.GalgameWebsiteComment{
		Content:   req.Content,
		WebsiteID: req.WebsiteID,
		UserID:    userID,
		ParentID:  req.ParentID,
	}
	if err := s.commentRepo.Create(&comment); err != nil {
		return nil, errors.ErrInternal("发表评论失败")
	}

	s.websiteRepo.AdjustCommentCount(req.WebsiteID, 1)

	// Notify the parent-comment author (nitro legacy: only when replying to an
	// existing comment, using the website.url slug as the link key). Capture the
	// parent author so the response can fill targetUser too.
	parentUserID := 0
	if req.ParentID != nil {
		if parent, err := s.commentRepo.FindByID(*req.ParentID); err == nil {
			parentUserID = parent.UserID
			url := s.websiteRepo.GetURL(req.WebsiteID)
			_ = s.notifier.Emit(nil, msgService.Spec{
				SenderID:   userID,
				ReceiverID: parent.UserID,
				Kind:       msgService.NotifyCommented,
				Content:    markdown.ToPlainText(req.Content, 233),
				WebsiteURL: url,
			})
		}
	}

	// Return the SAME enriched shape as the list (CommentItem) with the author —
	// and the parent author for a reply — hydrated, so the (reactive) frontend
	// can render the freshly-posted comment directly. Returning the raw model
	// left `user` empty and crashed the list on comment.user.name.
	uids := []int{userID}
	if parentUserID != 0 {
		uids = append(uids, parentUserID)
	}
	userMap := s.userClient.Hydrate(ctx, uids)
	u := userMap[userID]

	item := &dto.CommentItem{
		ID:         comment.ID,
		Content:    comment.Content,
		ParentID:   comment.ParentID,
		UserID:     comment.UserID,
		WebsiteID:  comment.WebsiteID,
		Created:    comment.CreatedAt.Format(time.RFC3339),
		Edited:     nil,
		Reply:      []*dto.CommentItem{},
		User:       dto.CommentUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
		TargetUser: nil,
	}
	if parentUserID != 0 {
		pu := userMap[parentUserID]
		item.TargetUser = dto.CommentUser{ID: pu.ID, Name: pu.Name, Avatar: pu.Avatar}
	}
	return item, nil
}

// ──────────────────────────────────────────
// DeleteComment — DELETE /website/:domain/comment
// ──────────────────────────────────────────

func (s *CommentService) DeleteComment(userID, userRole, commentID int) *errors.AppError {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != userID && userRole < 2 {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	// Count the subtree BEFORE deletion. The DB FK on parent_id is
	// `ON DELETE CASCADE` (legacy Prisma schema) so deleting a parent
	// also wipes all replies; the website's denormalized comment_count
	// must drop by the same amount or it stays inflated forever.
	subtreeSize := max(s.commentRepo.CountSubtree(commentID), 1)

	s.commentRepo.Delete(comment)
	s.websiteRepo.AdjustCommentCount(comment.WebsiteID, -int(subtreeSize))
	return nil
}
