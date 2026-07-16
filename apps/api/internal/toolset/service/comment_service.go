package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	toolsetRepo *repository.ToolsetRepository
	userClient  *userclient.Client
	// community serves the detail-preview read off the primitive (charter step
	// 06a); the frozen galgame_toolset_comment reads it replaced are retired.
	community *communityclient.Client
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	toolsetRepo *repository.ToolsetRepository,
	userClient *userclient.Client,
	community *communityclient.Client,
) *CommentService {
	return &CommentService{commentRepo: commentRepo, toolsetRepo: toolsetRepo, userClient: userClient, community: community}
}

// ──────────────────────────────────────────
// GetComments — GET /toolset/:id/comment/all
// Returns the camelCase response shape consumed by the frontend
// (commentData + total, with `targetUser` filled in for replies).
// ──────────────────────────────────────────

func (s *CommentService) GetComments(
	ctx context.Context,
	toolsetID int,
	req *dto.CommentListRequest,
) *dto.ToolsetCommentListResponse {
	all := s.commentRepo.FindAllByToolset(toolsetID) // oldest-first
	if len(all) == 0 {
		return &dto.ToolsetCommentListResponse{CommentData: []dto.ToolsetCommentItem{}, Total: 0}
	}

	// Index every comment, then group non-roots under their thread ROOT by
	// walking parent_id (replies are shallow). `all` is oldest-first, so each
	// root's flattened replies come out oldest-first too.
	byID := make(map[int]model.GalgameToolsetComment, len(all))
	for _, c := range all {
		byID[c.ID] = c
	}
	rootOf := func(c model.GalgameToolsetComment) int {
		cur := c
		for cur.ParentID != nil {
			parent, ok := byID[*cur.ParentID]
			if !ok {
				break
			}
			cur = parent
		}
		return cur.ID
	}

	roots := make([]model.GalgameToolsetComment, 0)
	repliesByRoot := make(map[int][]model.GalgameToolsetComment)
	for _, c := range all {
		if c.ParentID == nil {
			roots = append(roots, c)
			continue
		}
		repliesByRoot[rootOf(c)] = append(repliesByRoot[rootOf(c)], c)
	}

	// Roots honor the sort toggle (default newest-first); replies stay
	// oldest-first regardless.
	if req.SortOrder != "asc" {
		for i, j := 0, len(roots)-1; i < j; i, j = i+1, j-1 {
			roots[i], roots[j] = roots[j], roots[i]
		}
	}

	// Paginate ROOTS.
	total := int64(len(roots))
	start := min(max((req.Page-1)*req.Limit, 0), len(roots))
	end := min(start+req.Limit, len(roots))
	pageRoots := roots[start:end]

	// Hydrate authors of the page's roots + their replies + each reply's parent
	// author (for the "A → B" targetUser), in one batch.
	userIDSet := map[int]struct{}{}
	for _, r := range pageRoots {
		userIDSet[r.UserID] = struct{}{}
		for _, rep := range repliesByRoot[r.ID] {
			userIDSet[rep.UserID] = struct{}{}
			if rep.ParentID != nil {
				if p, ok := byID[*rep.ParentID]; ok {
					userIDSet[p.UserID] = struct{}{}
				}
			}
		}
	}
	uids := make([]int, 0, len(userIDSet))
	for id := range userIDSet {
		uids = append(uids, id)
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	build := func(c model.GalgameToolsetComment) dto.ToolsetCommentItem {
		// Banned author: keep the node (a root may still have non-banned
		// replies that would otherwise be orphaned) but tombstone it —
		// placeholder identity + suppressed content, so nothing the banned
		// user wrote is shown. Banned replies are dropped by the caller.
		author := userMap[c.UserID]
		content := c.Content
		if !userclient.IsRenderable(author) {
			author = userclient.Placeholder(c.UserID)
			content = ""
		}
		item := dto.ToolsetCommentItem{
			ID:        c.ID,
			ToolsetID: c.ToolsetID,
			Content:   content,
			Created:   c.CreatedAt,
			Edited:    c.Edited,
			ParentID:  c.ParentID,
			UserID:    c.UserID,
			Reply:     []dto.ToolsetCommentItem{},
			User:      userBriefFromClient(author),
		}
		if c.ParentID != nil {
			if p, ok := byID[*c.ParentID]; ok {
				pu := userBriefFromClient(userMap[p.UserID])
				item.TargetUser = &pu
			}
		}
		return item
	}

	items := make([]dto.ToolsetCommentItem, 0, len(pageRoots))
	for _, r := range pageRoots {
		root := build(r)
		reps := repliesByRoot[r.ID]
		root.Reply = make([]dto.ToolsetCommentItem, 0, len(reps))
		for _, rep := range reps {
			// Drop replies authored by a banned user (leaves — no orphans).
			if !userclient.IsRenderable(userMap[rep.UserID]) {
				continue
			}
			root.Reply = append(root.Reply, build(rep))
		}
		root.ReplyCount = len(root.Reply)
		items = append(items, root)
	}

	return &dto.ToolsetCommentListResponse{CommentData: items, Total: total}
}

// GetLatestForDetail returns the first ≤limit toolset comments for the detail
// preview, now sourced from the community primitive (charter step 06a). It
// resolves the toolset's comments thread (get-or-create), skips held/tombstoned
// and banned-author posts, and maps onto the existing CommentDetailItem shape.
// BEST-EFFORT: any S2S error (or unconfigured client) yields an empty slice —
// the detail page must always render.
func (s *CommentService) GetLatestForDetail(ctx context.Context, toolsetID, limit int) []dto.CommentDetailItem {
	items := []dto.CommentDetailItem{}
	thread, err := s.community.ResolveComments(ctx, communityclient.ResolveCommentsRequest{
		AnchorKind:    communityclient.AnchorSiteResource,
		AnchorID:      "toolset:" + strconv.Itoa(toolsetID),
		ContentRating: communityclient.RatingAll,
	})
	if err != nil {
		slog.Warn("toolset detail: community resolve failed (best-effort)", "toolset_id", toolsetID, "error", err)
		return items
	}

	uids := make([]int, 0, len(thread.Posts))
	for _, p := range thread.Posts {
		uids = append(uids, int(p.AuthorID))
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	for _, p := range thread.Posts {
		if len(items) >= limit {
			break
		}
		if p.Status != communityclient.PostVisible {
			continue
		}
		u := userMap[int(p.AuthorID)]
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, toolsetDetailItemFromPost(p, toolsetID, userBriefFromClient(u)))
	}
	return items
}

// toolsetDetailItemFromPost maps a community post onto the toolset detail-preview
// item shape. Content is the raw markdown (the FE renders it), parent_id mirrors
// the community reply pointer, and the timestamps parse from the RFC3339 wire.
func toolsetDetailItemFromPost(p communityclient.PostView, toolsetID int, user userModel.UserBrief) dto.CommentDetailItem {
	var parentID *int
	if p.ReplyToPostID != 0 {
		pid := int(p.ReplyToPostID)
		parentID = &pid
	}
	created := parseCommunityTime(p.CreatedAt)
	updated := created
	if edited := parseCommunityTime(p.EditedAt); !edited.IsZero() {
		updated = edited
	}
	return dto.CommentDetailItem{
		ID:        int(p.ID),
		Content:   p.ContentRaw,
		UserID:    int(p.AuthorID),
		ToolsetID: toolsetID,
		ParentID:  parentID,
		Edited:    parseCommunityTimePtr(p.EditedAt),
		Created:   created,
		Updated:   updated,
		User:      user,
	}
}

// parseCommunityTime parses an RFC3339 community timestamp, returning the zero
// time on any error (the wire format is RFC3339, but a robust fallback keeps a
// preview from breaking on an odd row).
func parseCommunityTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// parseCommunityTimePtr returns *time.Time (nil when empty/invalid) for the
// nullable edited_at.
func parseCommunityTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}

// ──────────────────────────────────────────
// CreateComment — POST /toolset/:id/comment
// ──────────────────────────────────────────

func (s *CommentService) CreateComment(
	ctx context.Context,
	userID, toolsetID int,
	req *dto.CreateCommentRequest,
) (*dto.ToolsetCommentItem, *errors.AppError) {
	// Verify toolset exists
	toolset, err := s.toolsetRepo.FindByID(toolsetID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	comment := model.GalgameToolsetComment{
		Content:   req.Content,
		UserID:    userID,
		ToolsetID: toolsetID,
		ParentID:  req.ParentID,
	}
	if err := s.commentRepo.Create(&comment); err != nil {
		return nil, errors.ErrInternal("发表评论失败")
	}

	// Send notification to toolset owner or parent comment owner
	go s.notifyCommentReceiver(userID, toolsetID, toolset.UserID, req)

	// Return the SAME enriched shape as the list (ToolsetCommentItem) with the
	// author — and the parent author for a reply — hydrated, so the frontend can
	// render the freshly-posted comment directly. Returning the raw model left
	// `user` empty (and snake_case keys), which crashed the list on
	// `comment.user.name` once the list became reactive.
	uids := []int{userID}
	parentUserID := 0
	if comment.ParentID != nil {
		if parent, err := s.commentRepo.FindByID(*comment.ParentID); err == nil {
			parentUserID = parent.UserID
			uids = append(uids, parentUserID)
		}
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	item := &dto.ToolsetCommentItem{
		ID:        comment.ID,
		ToolsetID: comment.ToolsetID,
		Content:   comment.Content,
		Created:   comment.CreatedAt,
		Edited:    comment.Edited,
		ParentID:  comment.ParentID,
		UserID:    comment.UserID,
		Reply:     []dto.ToolsetCommentItem{},
		User:      userBriefFromClient(userMap[userID]),
	}
	if parentUserID != 0 {
		pu := userBriefFromClient(userMap[parentUserID])
		item.TargetUser = &pu
	}
	return item, nil
}

// notifyCommentReceiver sends a "commented" or "replied" notification.
// It runs in a goroutine (fire-and-forget) so the HTTP request isn't blocked.
func (s *CommentService) notifyCommentReceiver(
	senderID, toolsetID, toolsetOwnerID int,
	req *dto.CreateCommentRequest,
) {
	receiverID := toolsetOwnerID
	msgType := "commented"

	if req.ParentID != nil {
		if parent, err := s.commentRepo.FindByID(*req.ParentID); err == nil {
			receiverID = parent.UserID
			msgType = "replied"
		}
	}

	if receiverID == senderID || receiverID <= 0 {
		return
	}

	s.commentRepo.DB().Create(&msgModel.Message{
		Content:    markdown.ToPlainText(req.Content, 100),
		Link:       fmt.Sprintf("/toolset/%d", toolsetID),
		Type:       msgType,
		SenderID:   senderID,
		ReceiverID: receiverID,
	})
}

// ──────────────────────────────────────────
// UpdateComment — PUT /toolset/:id/comment
// ──────────────────────────────────────────

func (s *CommentService) UpdateComment(
	userID int,
	req *dto.UpdateCommentRequest,
) *errors.AppError {
	comment, err := s.commentRepo.FindByID(req.CommentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != userID {
		return errors.ErrForbidden("您只能编辑自己的评论")
	}

	now := time.Now()
	s.commentRepo.UpdateContent(comment, req.Content, now)
	return nil
}

// ──────────────────────────────────────────
// DeleteComment — DELETE /toolset/:id/comment
// ──────────────────────────────────────────

func (s *CommentService) DeleteComment(
	userID int, canModerate bool, toolsetID int,
	req *dto.DeleteCommentRequest,
) *errors.AppError {
	comment, err := s.commentRepo.FindByID(req.CommentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}

	// Load toolset (may or may not exist; we only need its owner for perms).
	var ownerID int
	if t, err := s.toolsetRepo.FindByID(toolsetID); err == nil {
		ownerID = t.UserID
	} else if err != gorm.ErrRecordNotFound {
		// If the lookup fails for some other reason, treat ownerID as 0.
		ownerID = 0
	}

	if comment.UserID != userID && ownerID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	s.commentRepo.Delete(comment)
	return nil
}
