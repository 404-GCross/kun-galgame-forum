package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	userRepo "kun-galgame-api/internal/user/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// GalgameMessageSync drives the periodic ingestion of admin-triggered events
// (approved / declined / banned / unbanned) from galgame's /messages/feed.
// See docs/galgame_wiki/07-submission.md §调用方 cron 同步本地 status for
// the design rationale.
//
// Side effects per type (matches the kungal/wiki integration decision matrix):
//
//	approved    Creates the local galgame stub (approved is the moment a
//	            galgame becomes publicly visible and needs a row to anchor the
//	            list query), then awards +3 moemoepoint to TargetUserID.
//	banned      Delete the local galgame stub (CASCADE cleans interactions).
//	            Lazy-loaded stubs from prior likes/comments become orphan
//	            references after a ban; this cron is what tidies them up.
//	declined    No kungal-side action (the user gets a /messages/mine
//	            notification and can PATCH to re-submit).
//	unbanned    No kungal-side moemoe re-grant (the +3 was given at the
//	            original approval and would double-count on
//	            ban-then-unban).
//
// galgame=null (hard-deleted) → treated like banned (DELETE stub).
//
// Delivery semantics (at-least-once + idempotent = effectively exactly-once):
// the durable cursor only advances PAST a message once its side effects have
// landed. applyMessage signals a TRANSIENT failure (DB / OAuth network) by
// returning retry=true, which holds the cursor so the next tick re-attempts —
// safe because every side effect is idempotent (OnConflict stub + the STABLE
// per-message OAuth award key dedups replays). A PERMANENT OAuth rejection is
// logged and skipped (returns retry=false) so one poison message can't wedge
// the whole feed. This replaces the old "Redis SETNX before award" guard, which
// made the award at-MOST-once (a crash between guard-set and award lost the +3).
type GalgameMessageSync struct {
	galgameClient *client.GalgameClient
	galgameRepo   *repository.GalgameRepository
	stateRepo     *userRepo.StateRepository
	rdb           *redis.Client
}

func NewGalgameMessageSync(
	galgameClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	stateRepo *userRepo.StateRepository,
	rdb *redis.Client,
) *GalgameMessageSync {
	return &GalgameMessageSync{
		galgameClient: galgameClient,
		galgameRepo:   galgameRepo,
		stateRepo:     stateRepo,
		rdb:           rdb,
	}
}

const (
	// cronCursorKey is the Redis key holding the last-processed message id.
	// Single global cursor (server-wide), so storing in Redis rather than
	// a kungal SQL row keeps the cron stateless across restarts.
	cronCursorKey = "wiki:msg:cron:since"

	// messagesFeedBatch is the max number of messages requested per page.
	// Matches the galgame's stated 1000 limit.
	messagesFeedBatch = 1000

	// maxPagesPerRun guards against a runaway feed (galgame misbehaving and
	// always returning has_more=true). The cursor still advances within
	// the cap, so the next run picks up where this one stopped.
	maxPagesPerRun = 50
)

// Run executes one sync cycle. Safe to call concurrently — events use
// SETNX guards so duplicate processing only re-writes the cursor.
// Designed to be cheap when there's nothing new (one HTTP GET that
// returns has_more=false and items=[]).
func (s *GalgameMessageSync) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sinceID := s.readCursor(ctx)
	maxSeen := sinceID
	startedFrom := sinceID
	pages := 0

	for {
		pages++
		if pages > maxPagesPerRun {
			slog.Warn("galgame message sync 翻页超过上限, 中断本轮",
				"pages", pages, "since_id", maxSeen)
			break
		}

		feed, appErr := s.galgameClient.MessagesFeed(ctx, sinceID, messagesFeedBatch)
		if appErr != nil {
			slog.Warn("galgame message feed 拉取失败", "error", appErr, "since_id", sinceID)
			return
		}

		holding := false
		for _, msg := range feed.Items {
			if s.applyMessage(ctx, msg) {
				// Transient failure: hold the cursor BEFORE this message so the
				// next tick re-fetches from here and retries (idempotent).
				holding = true
				break
			}
			if msg.ID > maxSeen {
				maxSeen = msg.ID
			}
		}
		if holding {
			break
		}

		if !feed.HasMore || len(feed.Items) == 0 {
			break
		}
		sinceID = maxSeen
	}

	if maxSeen > startedFrom {
		s.writeCursor(ctx, maxSeen)
		slog.Info("galgame 消息同步完成",
			"from", startedFrom, "to", maxSeen, "pages", pages)
	}
}

// applyMessage applies one galgame feed message's kungal-side side effects.
// retry=true means a TRANSIENT failure (DB / OAuth network) — the caller holds
// the cursor and re-attempts next tick; this is safe because every side effect
// here is idempotent (OnConflict stub, idempotent DeleteLocalStub, and the
// STABLE per-message OAuth award key). retry=false means done (success, no-op,
// or a PERMANENT failure already logged — never wedge the feed on poison).
func (s *GalgameMessageSync) applyMessage(_ context.Context, msg client.NextMoeMessage) (retry bool) {
	// Hard-deleted galgame: clean any orphan stub. Idempotent.
	if msg.Galgame == nil {
		s.galgameRepo.DeleteLocalStub(msg.GalgameID)
		return false
	}

	switch msg.Type {
	case "approved":
		// approved is the moment a galgame becomes publicly visible — seed the
		// kungal stub (idempotent) so the list query has an anchor.
		if err := s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			return s.galgameRepo.CreateLocalStub(tx, msg.GalgameID)
		}); err != nil {
			slog.Warn("approved: 创建本地 stub 失败, 将重试",
				"msg_id", msg.ID, "gid", msg.GalgameID, "error", err)
			return true // transient DB — hold cursor
		}
		if msg.TargetUserID == nil || *msg.TargetUserID <= 0 {
			slog.Warn("approved 消息缺少 target_user_id, 仅创建 stub 跳过奖励",
				"msg_id", msg.ID, "gid", msg.GalgameID)
			return false
		}
		// Award +3 via OAuth SYNCHRONOUSLY so we know it landed before the
		// cursor advances. STABLE per-message key dedups cron replays (a retry
		// is a safe no-op). Transient failure → hold cursor; permanent OAuth
		// rejection → log + skip so one poison message can't wedge the feed.
		if err := moemoepoint.AwardSync(*msg.TargetUserID, constants.RewardCreateGalgame,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame", msg.GalgameID),
			moemoepoint.Key("wiki_approved", strconv.FormatInt(msg.ID, 10))); err != nil {
			if moemoepoint.IsPermanentAwardError(err) {
				slog.Error("approved: 发奖被 OAuth 永久拒绝, 跳过该消息",
					"msg_id", msg.ID, "target", *msg.TargetUserID, "error", err)
				return false
			}
			slog.Warn("approved: 发奖瞬时失败, 将重试",
				"msg_id", msg.ID, "target", *msg.TargetUserID, "error", err)
			return true
		}
		return false

	case "banned":
		s.galgameRepo.DeleteLocalStub(msg.GalgameID)
		return false

	case "declined", "unbanned":
		// No kungal-side action — see type-doc above.
		return false
	default:
		slog.Warn("收到未识别的 galgame 消息类型, 跳过",
			"type", msg.Type, "msg_id", msg.ID)
		return false
	}
}

func (s *GalgameMessageSync) readCursor(ctx context.Context) int64 {
	v, err := s.rdb.Get(ctx, cronCursorKey).Int64()
	if err == redis.Nil {
		return 0
	}
	if err != nil {
		slog.Warn("读取 galgame 消息游标失败 (从 0 开始)", "error", err)
		return 0
	}
	return v
}

func (s *GalgameMessageSync) writeCursor(ctx context.Context, id int64) {
	if err := s.rdb.Set(ctx, cronCursorKey, id, 0).Err(); err != nil {
		slog.Warn("写入 galgame 消息游标失败", "id", id, "error", err)
	}
}
