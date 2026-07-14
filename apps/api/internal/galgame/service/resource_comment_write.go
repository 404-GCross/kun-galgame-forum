package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

// createCtx carries the per-area data resolved once at create time (so the
// notification / counter / feed side effects don't re-query it).
type createCtx struct {
	galgameID    int // rating: the rated galgame (notification deep-link target)
	toolsetOwner int // toolset: the toolset owner (top-level "commented" receiver)
}

// ──────────────────────────────────────────
// CreateComment
// ──────────────────────────────────────────

// CreateComment resolves the resource's comments thread (get-or-create) and
// appends a post. For the FLAT rating area the caller passes targetUserID (the
// explicit "A → B" recipient, required — charter ruling 19); for the website /
// toolset trees the caller passes replyToPostID and the primitive derives the
// target from the parent. After the post lands it maintains each area's original
// side effects (notification parity, the website display counter, the activity
// feed row) best-effort — a failure there never fails the reply (content is
// already committed in the primitive).
func (s *ResourceCommentService) CreateComment(ctx context.Context, src CommentSource, resourceID, userID int, content string, replyToPostID, targetUserID *int64) (*CommunityPostItem, *errors.AppError) {
	cc, appErr := s.resolveCreateCtx(src, resourceID)
	if appErr != nil {
		return nil, appErr
	}

	thread, err := s.community.ResolveComments(ctx, communityclient.ResolveCommentsRequest{
		AnchorKind: communityclient.AnchorSiteResource, AnchorID: src.anchorID(resourceID), ContentRating: communityclient.RatingAll,
	})
	if err != nil {
		return nil, mapCommunityError(err)
	}

	req := communityclient.ReplyRequest{AuthorID: int64(userID), Body: content}
	if replyToPostID != nil {
		req.ReplyToPostID = *replyToPostID
	}
	if targetUserID != nil {
		req.TargetUserID = *targetUserID
	}
	post, err := s.community.Reply(ctx, thread.Thread.ID, req)
	if err != nil {
		return nil, mapCommunityError(err)
	}

	s.afterCreate(src, resourceID, cc, userID, content, post)

	// Render through the read pipeline (viewer = author, so a held post is
	// visible to it) for shape/pointer/author consistency with the list.
	items := s.renderPosts(ctx, userID, []communityclient.PostView{*post})
	if len(items) == 0 {
		return buildCommunityItem(*post, 0, s.userClient.Hydrate(ctx, []int{userID})[userID], 0, false), nil
	}
	return items[0], nil
}

// resolveCreateCtx does the per-area existence check + resolves the data the side
// effects need. rating: the rating must exist (its galgame id feeds the
// notification link, parity with the old ErrNotFound); toolset: the toolset must
// exist (its owner is the top-level "commented" receiver); website: no existence
// check (parity — the old website create never verified it).
func (s *ResourceCommentService) resolveCreateCtx(src CommentSource, resourceID int) (createCtx, *errors.AppError) {
	switch src.key {
	case sourceRating.key:
		gid := s.ratingGalgameID(resourceID)
		if gid == 0 {
			return createCtx{}, errors.ErrNotFound("未找到这个评分")
		}
		return createCtx{galgameID: gid}, nil
	case sourceToolset.key:
		owner := s.toolsetOwner(resourceID)
		if owner == 0 {
			return createCtx{}, errors.ErrNotFound("未找到该工具")
		}
		return createCtx{toolsetOwner: owner}, nil
	default: // website
		return createCtx{}, nil
	}
}

// notifyPlan is the who + what of a resource comment notification.
type notifyPlan struct {
	receiver int
	msgType  string
}

// resourceNotifyPlan is the PURE per-area notification decision (unit-tested for
// parity, charter ruling 20): rating → the explicit "A → B" target ("commented");
// website → the parent author ONLY on a reply ("commented"), a top-level comment
// notifies nobody; toolset → the parent author on a reply ("replied") else the
// toolset owner ("commented"). ok=false means notify nobody — a suppressed
// self-notification, a missing receiver, or a top-level website comment. The
// derived reply target is the primitive-completed post.TargetUserID.
func resourceNotifyPlan(src CommentSource, senderID int, post *communityclient.PostView, toolsetOwner int) (notifyPlan, bool) {
	var p notifyPlan
	switch src.key {
	case sourceRating.key:
		p = notifyPlan{receiver: int(post.TargetUserID), msgType: "commented"}
	case sourceWebsite.key:
		if post.ReplyToPostID == 0 || post.TargetUserID == 0 {
			return notifyPlan{}, false // a top-level website comment notifies nobody
		}
		p = notifyPlan{receiver: int(post.TargetUserID), msgType: "commented"}
	case sourceToolset.key:
		if post.ReplyToPostID != 0 && post.TargetUserID != 0 {
			p = notifyPlan{receiver: int(post.TargetUserID), msgType: "replied"}
		} else {
			p = notifyPlan{receiver: toolsetOwner, msgType: "commented"}
		}
	}
	if p.receiver <= 0 || p.receiver == senderID {
		return notifyPlan{}, false // self / missing receiver — suppressed (parity)
	}
	return p, true
}

// afterCreate replays each area's original create side effects onto the community
// post. Best-effort throughout (charter ruling 11 — the display counter tolerates
// drift; a notification / feed failure never fails the reply).
func (s *ResourceCommentService) afterCreate(src CommentSource, resourceID int, cc createCtx, userID int, content string, post *communityclient.PostView) {
	plan, notify := resourceNotifyPlan(src, userID, post, cc.toolsetOwner)
	switch src.key {
	case sourceRating.key:
		// Reused galgame helper: dedup on (sender,receiver,type,link=/galgame/<gid>).
		if notify {
			s.helpers.CreateGalgameMessageWithContent(s.db, userID, plan.receiver, plan.msgType, truncate(content, constants.TextPreviewLength), cc.galgameID)
		}
		s.feedUpsert(src.feedType, post.ID, userID, content, fmt.Sprintf("/galgame-rating/%d", resourceID), false, post.CreatedAt)

	case sourceWebsite.key:
		s.bumpWebsiteCommentCount(resourceID, 1) // charter ruling 21 — website counter is maintained
		slug, nsfw := s.websiteMeta(resourceID)
		if notify {
			s.notifyWebsiteReply(userID, plan.receiver, content, slug)
		}
		s.feedUpsert(src.feedType, post.ID, userID, content, "/website/"+slug, nsfw, post.CreatedAt)

	case sourceToolset.key:
		if notify {
			s.notifyToolset(userID, plan.receiver, plan.msgType, content, resourceID)
		}
		s.feedUpsert(src.feedType, post.ID, userID, content, fmt.Sprintf("/toolset/%d", resourceID), false, post.CreatedAt)
	}
}

// ──────────────────────────────────────────
// DeleteComment (region-aware)
// ──────────────────────────────────────────

// DeleteComment tombstones a post from a resource area. The resource id is pinned
// by the route path, so authority is decided SERVER-SIDE (never trusted from the
// client): author self-delete (the primitive checks author_id), a moderator, or —
// for rating (the rated galgame's owner) / toolset (the toolset owner) — the
// resource owner. Owner elevation additionally verifies the post actually belongs
// to THIS resource's thread before bypassing the author check, so a resource
// owner cannot delete an unrelated post by naming their own resource. On success
// it applies the website counter −1 (single-post tombstone semantics, charter
// ruling 11 — no subtree cascade) and removes the activity-feed row (parity with
// the old DELETE trigger).
func (s *ResourceCommentService) DeleteComment(ctx context.Context, src CommentSource, resourceID, userID int, canModerate bool, postID int64) *errors.AppError {
	elevated := canModerate
	if !elevated {
		if owner := s.resourceOwner(src, resourceID); owner != 0 && owner == userID {
			ok, verr := s.postInResourceThread(ctx, src, resourceID, postID)
			if verr != nil {
				return mapCommunityError(verr)
			}
			if !ok {
				return errors.ErrForbidden("您没有权限删除此评论")
			}
			elevated = true
		}
	}

	if err := s.community.DeletePost(ctx, postID, int64(userID), elevated); err != nil {
		return mapCommunityError(err)
	}

	if src.key == sourceWebsite.key {
		s.bumpWebsiteCommentCount(resourceID, -1)
	}
	s.feedDelete(src.feedType, postID)
	return nil
}

// resourceOwner returns the user who may delete any comment in this area beyond
// its author: rating → the rated galgame's owner; toolset → the toolset owner;
// website → none (charter ruling 20). 0 = no owner branch / resource missing.
func (s *ResourceCommentService) resourceOwner(src CommentSource, resourceID int) int {
	switch src.key {
	case sourceRating.key:
		gid := s.ratingGalgameID(resourceID)
		if gid == 0 {
			return 0
		}
		return s.galgameOwner(gid)
	case sourceToolset.key:
		return s.toolsetOwner(resourceID)
	default: // website — no owner branch
		return 0
	}
}

// postInResourceThread reports whether postID belongs to the resource's comments
// thread. Used only on the rare owner-elevated delete path (anti-escalation). The
// resource threads are small, so a bounded keyset scan is cheap.
func (s *ResourceCommentService) postInResourceThread(ctx context.Context, src CommentSource, resourceID int, postID int64) (bool, error) {
	thread, err := s.community.ResolveComments(ctx, communityclient.ResolveCommentsRequest{
		AnchorKind: communityclient.AnchorSiteResource, AnchorID: src.anchorID(resourceID), ContentRating: communityclient.RatingAll,
	})
	if err != nil {
		return false, err
	}
	if containsPost(thread.Posts, postID) {
		return true, nil
	}
	cursor := thread.NextCursor
	for cursor != "" {
		page, perr := s.community.ListPosts(ctx, thread.Thread.ID, cursor, "50")
		if perr != nil {
			return false, perr
		}
		if containsPost(page.Posts, postID) {
			return true, nil
		}
		cursor = page.NextCursor
	}
	return false, nil
}

func containsPost(posts []communityclient.PostView, postID int64) bool {
	for _, p := range posts {
		if p.ID == postID {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────
// Per-area lookups (raw, to avoid cross-domain repo coupling)
// ──────────────────────────────────────────

func (s *ResourceCommentService) ratingGalgameID(ratingID int) int {
	var gid int
	s.db.Table("galgame_rating").Select("galgame_id").Where("id = ?", ratingID).Scan(&gid)
	return gid
}

func (s *ResourceCommentService) galgameOwner(galgameID int) int {
	var uid int
	s.db.Table("galgame").Select("user_id").Where("id = ?", galgameID).Scan(&uid)
	return uid
}

func (s *ResourceCommentService) toolsetOwner(toolsetID int) int {
	var uid int
	s.db.Table("galgame_toolset").Select("user_id").Where("id = ?", toolsetID).Scan(&uid)
	return uid
}

// websiteMeta resolves the website's url slug + nsfw flag, mirroring the feed
// trigger's projection EXACTLY: a present row is nsfw when age_limit <> 'all'; a
// missing row yields an empty slug (link "/website/") + not-nsfw (the trigger's
// COALESCE(v_nsfw, false)). Existence is read from RowsAffected so an empty
// age_limit on a present row is still treated as nsfw, like `” <> 'all'`.
func (s *ResourceCommentService) websiteMeta(websiteID int) (slug string, nsfw bool) {
	var row struct {
		URL      string `gorm:"column:url"`
		AgeLimit string `gorm:"column:age_limit"`
	}
	res := s.db.Table("galgame_website").Select("url, age_limit").Where("id = ?", websiteID).Limit(1).Find(&row)
	if res.Error != nil || res.RowsAffected == 0 {
		return "", false
	}
	return row.URL, row.AgeLimit != "all"
}

// bumpWebsiteCommentCount adjusts galgame_website.comment_count, floored at 0 (a
// tolerated display counter). Best-effort.
func (s *ResourceCommentService) bumpWebsiteCommentCount(websiteID, delta int) {
	if err := s.db.Table("galgame_website").Where("id = ?", websiteID).
		Update("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error; err != nil {
		slog.Warn("website comment counter adjust failed (best-effort)", "website_id", websiteID, "delta", delta, "error", err)
	}
}

// ──────────────────────────────────────────
// Notifications (per-area parity)
// ──────────────────────────────────────────

// notifyWebsiteReply replays the website reply notification: "commented" → parent
// author, link /website/<slug>, content ToPlainText(233), deduped on the full
// (sender,receiver,type,content,link) tuple (parity with the message notifier).
// The receiver is already validated (non-self, >0) by resourceNotifyPlan.
func (s *ResourceCommentService) notifyWebsiteReply(senderID, receiverID int, content, slug string) {
	link := "/website/" + slug
	preview := markdown.ToPlainText(content, constants.TextPreviewLength)
	var count int64
	s.db.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND content = ? AND link = ?",
			senderID, receiverID, "commented", preview, link).
		Count(&count)
	if count > 0 {
		return
	}
	s.db.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "commented", Content: preview, Link: link, Status: "unread",
	})
}

// notifyToolset replays the toolset notification: "replied"/"commented", link
// /toolset/<id>, content ToPlainText(100), NO dedup (parity — the old toolset
// notify created unconditionally). The receiver is already validated (non-self,
// >0) by resourceNotifyPlan.
func (s *ResourceCommentService) notifyToolset(senderID, receiverID int, msgType, content string, toolsetID int) {
	s.db.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type:    msgType,
		Content: markdown.ToPlainText(content, 100),
		Link:    fmt.Sprintf("/toolset/%d", toolsetID),
		Status:  "unread",
	})
}

// ──────────────────────────────────────────
// Activity feed parity (charter ruling 22)
// ──────────────────────────────────────────

// feedUpsert projects a community-backed comment into the materialized feed the
// old table's AFTER-INSERT trigger used to maintain (it can't: community posts
// live in another database). It reuses the SAME feed_upsert() SQL function the
// triggers call, so the row is byte-for-byte the projection the old comment would
// have produced — except source_id is the community post id. source_id is an
// int32 column; a post id beyond int32 is skipped + logged (see the execution
// report — a latent cap, not a live risk). Best-effort.
func (s *ResourceCommentService) feedUpsert(feedType string, postID int64, userID int, content, link string, nsfw bool, createdAt string) {
	if postID > math.MaxInt32 {
		slog.Warn("feed_activity source_id overflow — community post id exceeds int32; feed parity row skipped", "post_id", postID, "type", feedType)
		return
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		created = time.Now()
	}
	if err := s.db.Exec("SELECT feed_upsert(?, ?, ?, ?, ?, ?, ?, ?)",
		feedType, postID, userID, 0, truncate(content, 100), link, nsfw, created).Error; err != nil {
		slog.Warn("feed_activity parity upsert failed (best-effort)", "post_id", postID, "type", feedType, "error", err)
	}
}

// feedDelete removes a comment's feed row on tombstone (parity with the old
// DELETE trigger). Best-effort.
func (s *ResourceCommentService) feedDelete(feedType string, postID int64) {
	if postID > math.MaxInt32 {
		return
	}
	if err := s.db.Exec("SELECT feed_delete(?, ?)", feedType, postID).Error; err != nil {
		slog.Warn("feed_activity parity delete failed (best-effort)", "post_id", postID, "type", feedType, "error", err)
	}
}
