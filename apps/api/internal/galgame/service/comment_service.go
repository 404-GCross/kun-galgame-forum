package service

import (
	"context"
	"fmt"
	"time"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/moemoepoint"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	stateRepo   *userRepo.StateRepository
	userClient  *userclient.Client
	helpers     InteractionHelpers
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	stateRepo *userRepo.StateRepository,
	userClient *userclient.Client,
) *CommentService {
	return &CommentService{commentRepo: commentRepo, stateRepo: stateRepo, userClient: userClient}
}

// ──────────────────────────────────────────
// Response types
// ──────────────────────────────────────────

type UserObj struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// CommentItem is the wire shape. Roots have Replies populated with the
// full descendant subtree; the frontend visually caps recursion at
// depth 3 and opens the drawer for deeper threads. ReplyCount counts
// only direct + transitive descendants of THIS node (not the whole
// thread under the root), so a mid-tree "查看更多" button can use it.
type CommentItem struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
	// ContentHtml: server-rendered HTML via the project's goldmark
	// pipeline (apps/api/internal/infrastructure/markdown). Same
	// pipeline as topic / galgame intro / toolset / doc, so KunContent
	// + DOMPurify on the frontend handles spoilers, code-blocks,
	// MathJax, tables, and KaTeX consistently with the rest of the
	// site. Keeps Content as the raw source for the edit-mode
	// textarea to round-trip cleanly.
	ContentHtml     string   `json:"contentHtml"`
	GalgameID       int      `json:"galgameId"`
	User            UserObj  `json:"user"`
	TargetUser      *UserObj `json:"targetUser"`
	ParentCommentID *int     `json:"parentCommentId"`
	RootCommentID   *int     `json:"rootCommentId"`
	LikeCount       int      `json:"likeCount"`
	// IsLiked is per-viewer: set by GetComments/GetCommentThread from a
	// batch query against galgame_comment_like. Anonymous callers see
	// false. The FE Like component initialises its toggled state from
	// this flag — without it the heart icon would always render off
	// even after the user has already clicked it.
	IsLiked bool   `json:"isLiked"`
	Created string `json:"created"`
	// Edited: ISO timestamp when the author last rewrote this comment;
	// null if untouched. Drives the "已编辑" badge in the UI.
	Edited     *string        `json:"edited"`
	Replies    []*CommentItem `json:"replies"`
	ReplyCount int            `json:"replyCount"`
}

type CommentListResult struct {
	Items []*CommentItem
	Total int64
}

// ──────────────────────────────────────────
// GetComments
// ──────────────────────────────────────────

// inlineRepliesPerRoot caps how many descendants are shipped inline
// per root. The frontend's visual model has exactly TWO tiers (root +
// "reply"); these N replies appear as a flat list under each root
// regardless of their actual DB depth. Anything older surfaces via the
// "查看更多 N 条回复" lazy-loaded drawer.
const inlineRepliesPerRoot = 3

// GetComments returns paginated ROOT comments. Each root carries up to
// `inlineRepliesPerRoot` of its most recent descendants (any depth)
// FLATTENED into a single list, plus a ReplyCount = total descendants.
// The drawer (GET /comment/thread/:rootId) fetches the whole thread
// on demand for the "查看更多" path.
func (s *CommentService) GetComments(ctx context.Context, galgameID, page, limit, currentUserID int, sortOrder string) *CommentListResult {
	total := s.commentRepo.CountRootsByGalgame(galgameID)
	roots := s.commentRepo.FindRootsPaginated(galgameID, page, limit, sortOrder)
	if len(roots) == 0 {
		return &CommentListResult{Items: []*CommentItem{}, Total: total}
	}

	rootIDs := make([]int, 0, len(roots))
	for _, r := range roots {
		rootIDs = append(rootIDs, r.ID)
	}

	descendants := s.commentRepo.FindLatestDescendantsByRoots(rootIDs, inlineRepliesPerRoot)
	descendantCounts := s.commentRepo.CountDescendantsByRoots(rootIDs)

	allRows := make([]repository.CommentRow, 0, len(roots)+len(descendants))
	allRows = append(allRows, roots...)
	allRows = append(allRows, descendants...)

	userMap := s.hydrateUsers(ctx, allRows)

	rootItems := make([]*CommentItem, 0, len(roots))
	rootByID := make(map[int]*CommentItem, len(roots))
	for _, r := range roots {
		item := s.buildSingleItem(r, userMap)
		if item == nil {
			continue
		}
		item.ReplyCount = descendantCounts[r.ID]
		rootItems = append(rootItems, item)
		rootByID[r.ID] = item
	}

	// Flat-attach each descendant directly to its root (skipping the
	// intermediate parent chain). DB structure stays normalized; the
	// view collapses to a single nested tier.
	for _, d := range descendants {
		item := s.buildSingleItem(d, userMap)
		if item == nil || d.RootCommentID == nil {
			continue
		}
		if root, ok := rootByID[*d.RootCommentID]; ok {
			root.Replies = append(root.Replies, item)
		}
	}

	// Per-viewer `isLiked` flag — batched in one query then stamped on
	// every item across the roots-and-flattened-replies graph. Anonymous
	// callers (currentUserID=0) skip the DB hit entirely (FindLikedSet
	// returns the empty set).
	allIDs := make([]int, 0, len(rootItems)*(1+inlineRepliesPerRoot))
	for _, item := range rootItems {
		allIDs = append(allIDs, item.ID)
		for _, reply := range item.Replies {
			allIDs = append(allIDs, reply.ID)
		}
	}
	likedSet := s.commentRepo.FindLikedSet(currentUserID, allIDs)
	for _, item := range rootItems {
		item.IsLiked = likedSet[item.ID]
		for _, reply := range item.Replies {
			reply.IsLiked = likedSet[reply.ID]
		}
	}

	return &CommentListResult{Items: rootItems, Total: total}
}

// ──────────────────────────────────────────
// GetThread
// ──────────────────────────────────────────

// GetThread returns root + all descendants flattened under root.Replies
// (ASC by created). Mirrors the inline flat-tier shape, with no per-
// root cap.
func (s *CommentService) GetThread(ctx context.Context, rootID, currentUserID int) (*CommentItem, *errors.AppError) {
	rows := s.commentRepo.FindThreadByRoot(rootID)
	if len(rows) == 0 {
		return nil, errors.ErrNotFound("评论线程不存在")
	}
	userMap := s.hydrateUsers(ctx, rows)
	// Batch-fetch the viewer's like state across the whole thread; same
	// rationale as GetComments above. Empty for anonymous callers.
	commentIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		commentIDs = append(commentIDs, r.ID)
	}
	likedSet := s.commentRepo.FindLikedSet(currentUserID, commentIDs)

	var rootItem *CommentItem
	descendants := make([]*CommentItem, 0, len(rows)-1)
	for _, r := range rows {
		item := s.buildSingleItem(r, userMap)
		if item == nil {
			continue
		}
		item.IsLiked = likedSet[item.ID]
		if r.ID == rootID {
			rootItem = item
		} else {
			descendants = append(descendants, item)
		}
	}
	if rootItem == nil {
		return nil, errors.ErrNotFound("评论线程不存在")
	}
	rootItem.Replies = descendants
	rootItem.ReplyCount = len(descendants)
	return rootItem, nil
}

// ──────────────────────────────────────────
// Row → item assembly
// ──────────────────────────────────────────

func (s *CommentService) hydrateUsers(ctx context.Context, rows []repository.CommentRow) map[int]userclient.User {
	uidSet := make(map[int]struct{}, len(rows)*2)
	for _, r := range rows {
		uidSet[r.UserID] = struct{}{}
		if r.TargetUserID != nil && *r.TargetUserID > 0 {
			uidSet[*r.TargetUserID] = struct{}{}
		}
	}
	uids := make([]int, 0, len(uidSet))
	for id := range uidSet {
		uids = append(uids, id)
	}
	return s.userClient.Hydrate(ctx, uids)
}

// buildSingleItem maps one row to a CommentItem with empty Replies.
// Returns nil if the author is no longer renderable (deleted user),
// letting callers skip the row.
func (s *CommentService) buildSingleItem(r repository.CommentRow, userMap map[int]userclient.User) *CommentItem {
	u := userMap[r.UserID]
	if !userclient.IsRenderable(u) {
		return nil
	}
	item := &CommentItem{
		ID:              r.ID,
		Content:         r.Content,
		ContentHtml:     markdown.Render(r.Content),
		GalgameID:       r.GalgameID,
		User:            UserObj{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
		ParentCommentID: r.ParentCommentID,
		RootCommentID:   r.RootCommentID,
		LikeCount:       r.LikeCount,
		Created:         r.CreatedAt,
		Replies:         []*CommentItem{},
	}
	if r.Edited != nil {
		s := r.Edited.Format("2006-01-02T15:04:05Z")
		item.Edited = &s
	}
	if r.TargetUserID != nil {
		if t, ok := userMap[*r.TargetUserID]; ok {
			item.TargetUser = &UserObj{ID: t.ID, Name: t.Name, Avatar: t.Avatar}
		}
	}
	return item
}

// ──────────────────────────────────────────
// CreateComment
// ──────────────────────────────────────────

// CreateComment creates a top-level or reply comment.
//   - parentCommentID == nil → root comment.
//   - parentCommentID != nil → reply; we look up the parent, verify
//     it belongs to the same galgame, and derive root_comment_id as
//     parent.RootCommentID (if parent is itself a reply) or parent.ID
//     (if parent is a root).
func (s *CommentService) CreateComment(
	ctx context.Context,
	userID, galgameID int,
	content string,
	targetUserID *int,
	parentCommentID *int,
) (*CommentItem, *errors.AppError) {
	var (
		parentID *int
		rootID   *int
	)
	if parentCommentID != nil {
		parent, err := s.commentRepo.FindByID(*parentCommentID)
		if err != nil || parent.GalgameID != galgameID {
			return nil, errors.ErrBadRequest("回复的评论不存在")
		}
		pid := parent.ID
		parentID = &pid
		if parent.RootCommentID != nil {
			rid := *parent.RootCommentID
			rootID = &rid
		} else {
			rid := parent.ID
			rootID = &rid
		}
	}

	comment := model.GalgameComment{
		Content:         content,
		GalgameID:       galgameID,
		UserID:          userID,
		TargetUserID:    targetUserID,
		ParentCommentID: parentID,
		RootCommentID:   rootID,
	}

	txErr := s.commentRepo.DB().Transaction(func(tx *gorm.DB) error {
		// Lazy-create stub: a pending submission has no kungal stub yet
		// (decision 2 in the wiki integration plan), so the very first
		// interaction must INSERT one or the comment_count UPDATE below
		// silently affects 0 rows.
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.GalgameLocal{ID: galgameID}).Error; err != nil {
			return err
		}
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
			Update("comment_count", gorm.Expr("comment_count + 1")).Error; err != nil {
			return err
		}

		if targetUserID != nil && *targetUserID != userID {
			// Award via OAuth (no local +=). Comment engagement → content_approved.
			moemoepoint.Award(*targetUserID, 1, moemoepoint.ReasonContentApproved,
				moemoepoint.Ref("galgame_comment", galgameID),
				moemoepoint.KeyNonce(moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_comment", galgameID)))

			link := fmt.Sprintf("/galgame/%d", galgameID)
			if err := tx.Create(&msgModel.Message{
				SenderID: userID, ReceiverID: *targetUserID,
				Type: "commented", Content: truncate(content, 233),
				Link: link, Status: "unread",
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("发表评论失败")
	}

	creator, _, _ := s.userClient.User(ctx, userID)
	resp := &CommentItem{
		ID:              comment.ID,
		Content:         comment.Content,
		ContentHtml:     markdown.Render(comment.Content),
		GalgameID:       comment.GalgameID,
		User:            UserObj{ID: creator.ID, Name: creator.Name, Avatar: creator.Avatar},
		ParentCommentID: comment.ParentCommentID,
		RootCommentID:   comment.RootCommentID,
		LikeCount:       0,
		Created:         comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Replies:         []*CommentItem{},
	}
	if targetUserID != nil {
		t, _, _ := s.userClient.User(ctx, *targetUserID)
		resp.TargetUser = &UserObj{ID: t.ID, Name: t.Name, Avatar: t.Avatar}
	}

	return resp, nil
}

// ──────────────────────────────────────────
// DeleteComment
// ──────────────────────────────────────────

// DeleteComment removes a comment. Cascading FK deletes child rows in
// galgame_comment (parent_comment_id ON DELETE CASCADE), so the
// comment_count adjustment subtracts the size of the destroyed subtree.
func (s *CommentService) DeleteComment(userID int, canModerate bool, commentID int) *errors.AppError {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	// Count this comment + every descendant reachable through
	// parent_comment_id so we can decrement galgame.comment_count by the
	// full subtree size (Postgres cascades the actual row deletes via
	// the parent_comment_id FK). The old query
	// `id = ? OR root_comment_id = ?` was only correct when commentID
	// happened to be a thread root — deleting a mid-tree reply would
	// under-count its grandchildren, since root_comment_id always points
	// to the top of the thread, not the immediate parent. Recursive CTE
	// is independent of the depth-of-current-row assumption.
	var subtreeSize int64
	s.commentRepo.DB().Raw(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM galgame_comment WHERE id = ?
			UNION ALL
			SELECT c.id
			FROM galgame_comment c
			JOIN subtree s ON c.parent_comment_id = s.id
		)
		SELECT COUNT(*) FROM subtree
	`, commentID).Scan(&subtreeSize)
	if subtreeSize < 1 {
		subtreeSize = 1
	}

	txErr := s.commentRepo.DB().Transaction(func(tx *gorm.DB) error {
		// Likes on descendant comments are cleared by their cascade
		// chain through galgame_comment → galgame_comment_like.
		tx.Where("galgame_comment_id = ?", commentID).Delete(&model.GalgameCommentLike{})
		tx.Delete(&comment)
		tx.Model(&model.GalgameLocal{}).Where("id = ?", comment.GalgameID).
			Update("comment_count", gorm.Expr("comment_count - ?", subtreeSize))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除评论失败")
	}

	return nil
}

// ──────────────────────────────────────────
// UpdateComment
// ──────────────────────────────────────────

// UpdateComment rewrites the content of an existing comment and stamps
// the `edited` column. Author-only: moderators (CanModerate) can also
// edit, matching the same authority used by DeleteComment.
//
// Returns the patched comment so the frontend can drop the response
// straight into its tree without re-fetching.
func (s *CommentService) UpdateComment(
	ctx context.Context,
	userID int, canModerate bool, commentID int,
	content string,
) (*CommentItem, *errors.AppError) {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != userID && !canModerate {
		return nil, errors.ErrForbidden("您没有权限编辑此评论")
	}

	now := time.Now()
	if err := s.commentRepo.UpdateContent(commentID, content, now); err != nil {
		return nil, errors.ErrInternal("更新评论失败")
	}

	// Re-read so the response reflects the canonical row (covers the
	// rare case where a concurrent write updated other fields).
	updated, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return nil, errors.ErrInternal("更新评论失败")
	}

	creator, _, _ := s.userClient.User(ctx, updated.UserID)
	editedStr := now.Format("2006-01-02T15:04:05Z")
	resp := &CommentItem{
		ID:              updated.ID,
		Content:         updated.Content,
		ContentHtml:     markdown.Render(updated.Content),
		GalgameID:       updated.GalgameID,
		User:            UserObj{ID: creator.ID, Name: creator.Name, Avatar: creator.Avatar},
		ParentCommentID: updated.ParentCommentID,
		RootCommentID:   updated.RootCommentID,
		LikeCount:       updated.LikeCount,
		Created:         updated.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Edited:          &editedStr,
		Replies:         []*CommentItem{},
	}
	if updated.TargetUserID != nil {
		t, _, _ := s.userClient.User(ctx, *updated.TargetUserID)
		resp.TargetUser = &UserObj{ID: t.ID, Name: t.Name, Avatar: t.Avatar}
	}
	return resp, nil
}

// ──────────────────────────────────────────
// ToggleCommentLike
// ──────────────────────────────────────────

func (s *CommentService) ToggleCommentLike(userID, commentID int) *errors.AppError {
	txErr := s.commentRepo.DB().Transaction(func(tx *gorm.DB) error {
		var comment model.GalgameComment
		if err := tx.First(&comment, commentID).Error; err != nil {
			return err // not-found → reported as 404 below; no like on a ghost comment
		}

		var existing model.GalgameCommentLike
		result := tx.Where("user_id = ? AND galgame_comment_id = ?", userID, commentID).First(&existing)
		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			return result.Error
		}

		if result.Error == gorm.ErrRecordNotFound {
			if err := tx.Create(&model.GalgameCommentLike{UserID: userID, CommentID: commentID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.GalgameComment{}).Where("id = ?", commentID).
				Update("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			if comment.UserID != userID {
				moemoepoint.Award(comment.UserID, 1, moemoepoint.ReasonLiked,
					moemoepoint.Ref("galgame_comment", commentID),
					moemoepoint.KeyNonce(moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_comment", commentID)))
				s.helpers.CreateGalgameMessageWithContent(
					tx, userID, comment.UserID, "liked",
					truncate(comment.Content, 233),
					comment.GalgameID,
				)
			}
		} else {
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.GalgameComment{}).Where("id = ?", commentID).
				Update("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
				return err
			}
			if comment.UserID != userID {
				moemoepoint.Award(comment.UserID, -1, moemoepoint.ReasonLiked,
					moemoepoint.Ref("galgame_comment", commentID),
					moemoepoint.KeyNonce(moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_comment", commentID)))
			}
		}
		return nil
	})
	if txErr != nil {
		if txErr == gorm.ErrRecordNotFound {
			return errors.ErrNotFound("评论不存在")
		}
		return errors.ErrInternal("操作失败")
	}

	return nil
}

// ──────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
