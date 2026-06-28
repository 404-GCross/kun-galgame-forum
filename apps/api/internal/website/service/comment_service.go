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
	rows := s.commentRepo.FindByWebsite(websiteID) // newest-first

	uids := userclient.CollectIDs(rows, func(r repository.CommentRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	// Index rows; group non-roots under their thread ROOT by walking parent_id.
	// The view is two flat tiers (root + replies) regardless of DB depth.
	rowByID := make(map[int]repository.CommentRow, len(rows))
	for _, r := range rows {
		rowByID[r.ID] = r
	}
	rootOf := func(r repository.CommentRow) int {
		cur := r
		for cur.ParentID != nil {
			parent, ok := rowByID[*cur.ParentID]
			if !ok {
				break
			}
			cur = parent
		}
		return cur.ID
	}
	build := func(r repository.CommentRow) *dto.CommentItem {
		u := userMap[r.UserID]
		item := &dto.CommentItem{
			ID:         r.ID,
			Content:    r.Content,
			ParentID:   r.ParentID,
			UserID:     r.UserID,
			WebsiteID:  websiteID,
			Created:    r.Created,
			Edited:     r.Edited,
			Reply:      []*dto.CommentItem{},
			User:       dto.CommentUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			TargetUser: nil,
		}
		// targetUser = the replied-to comment's author ("A → B").
		if r.ParentID != nil {
			if p, ok := rowByID[*r.ParentID]; ok {
				if pu := userMap[p.UserID]; userclient.IsRenderable(pu) {
					item.TargetUser = dto.CommentUser{ID: pu.ID, Name: pu.Name, Avatar: pu.Avatar}
				}
			}
		}
		return item
	}

	// rows is newest-first; walk it in reverse so roots and each root's
	// flattened replies come out oldest-first.
	rootItems := make(map[int]*dto.CommentItem, len(rows))
	roots := make([]*dto.CommentItem, 0)
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		if r.ParentID != nil || !userclient.IsRenderable(userMap[r.UserID]) {
			continue
		}
		item := build(r)
		rootItems[r.ID] = item
		roots = append(roots, item)
	}
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		if r.ParentID == nil || !userclient.IsRenderable(userMap[r.UserID]) {
			continue
		}
		root, ok := rootItems[rootOf(r)]
		if !ok {
			// Root was dropped (banned author) — the reply has no visible
			// thread to attach to; drop it (as the prior tree did for orphans).
			continue
		}
		root.Reply = append(root.Reply, build(r))
	}
	for _, root := range roots {
		root.ReplyCount = len(root.Reply)
	}
	return roots
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

func (s *CommentService) DeleteComment(userID int, canModerate bool, commentID int) *errors.AppError {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != userID && !canModerate {
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
