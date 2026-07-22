package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// GalgameRevisionSync mirrors the galgame's merged-revision (edit) events into the
// local galgame_activity table so the forum activity timeline can render galgame
// edits without querying the remote galgame at page-render time.
//
// Same design as GalgameMessageSync: a durable Redis cursor advances only PAST
// events whose local upsert succeeded; the upsert is idempotent (ON CONFLICT on
// wiki_revision_id), so at-least-once delivery is effectively exactly-once. A
// transient failure (galgame down / DB error) holds the cursor so the next tick
// re-attempts from the same point.
//
// Why a separate sync from GalgameMessageSync: different feed (/revisions/recent vs
// /messages/feed), different cursor, different target table. They share only the
// internal-face galgame client (X-API-Key).
type GalgameRevisionSync struct {
	galgameClient *client.GalgameClient
	db            *gorm.DB
	rdb           *redis.Client
}

func NewGalgameRevisionSync(
	galgameClient *client.GalgameClient,
	db *gorm.DB,
	rdb *redis.Client,
) *GalgameRevisionSync {
	return &GalgameRevisionSync{galgameClient: galgameClient, db: db, rdb: rdb}
}

const (
	// revisionCursorKey holds the last-processed revision id (Redis, server-wide).
	revisionCursorKey = "wiki:rev:cron:since"
	// revisionFeedBatch is the max rows requested per page (galgame caps at 5000).
	revisionFeedBatch = 1000
	// revisionMaxPages guards against a runaway feed (always has_more=true).
	revisionMaxPages = 50
)

// Run executes one sync cycle. Cheap when there's nothing new (one GET that
// returns has_more=false and items=[]).
func (s *GalgameRevisionSync) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sinceID := s.readCursor(ctx)
	maxSeen := sinceID
	startedFrom := sinceID
	pages := 0

	for {
		pages++
		if pages > revisionMaxPages {
			slog.Warn("galgame revision sync 翻页超过上限, 中断本轮",
				"pages", pages, "since_id", maxSeen)
			break
		}

		feed, appErr := s.galgameClient.RecentRevisions(ctx, sinceID, revisionFeedBatch)
		if appErr != nil {
			// Don't advance the cursor — next tick retries from sinceID.
			slog.Warn("galgame revision feed 拉取失败", "error", appErr, "since_id", sinceID)
			return
		}

		holding := false
		for _, rev := range feed.Items {
			if err := s.upsert(rev); err != nil {
				// Transient DB failure: hold the cursor BEFORE this id so the
				// next tick re-attempts (idempotent ON CONFLICT).
				slog.Warn("galgame revision 入库失败, 持有游标重试",
					"rev_id", rev.ID, "error", err)
				holding = true
				break
			}
			if rev.ID > maxSeen {
				maxSeen = rev.ID
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
		slog.Info("galgame 修订同步完成", "from", startedFrom, "to", maxSeen, "pages", pages)
	}
}

// upsert writes one merged-revision event into galgame_activity. Idempotent via
// the unique wiki_revision_id, so cron re-runs / retries are safe no-ops.
func (s *GalgameRevisionSync) upsert(rev client.NextMoeRevision) error {
	created, err := time.Parse(time.RFC3339, rev.Created)
	if err != nil {
		created = time.Now()
	}
	return s.db.Exec(`
		INSERT INTO galgame_activity (wiki_revision_id, wiki_revision_number, galgame_id, user_id, type, created)
		VALUES (?, ?, ?, ?, 'GALGAME_EDIT', ?)
		ON CONFLICT (wiki_revision_id) DO NOTHING
	`, rev.ID, rev.Revision, rev.GalgameID, rev.UserID, created).Error
}

func (s *GalgameRevisionSync) readCursor(ctx context.Context) int64 {
	v, err := s.rdb.Get(ctx, revisionCursorKey).Int64()
	if err == redis.Nil {
		return 0
	}
	if err != nil {
		slog.Warn("读取 galgame 修订游标失败 (从 0 开始)", "error", err)
		return 0
	}
	return v
}

func (s *GalgameRevisionSync) writeCursor(ctx context.Context, id int64) {
	if err := s.rdb.Set(ctx, revisionCursorKey, id, 0).Err(); err != nil {
		slog.Warn("写入 galgame 修订游标失败", "id", id, "error", err)
	}
}
