package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/redis/go-redis/v9"
)

// GalgameContributorSync projects the editing engine's work revisions onto
// kungal's contributor table (migration 069) — the forward half of the answer
// the detail page's contributor strip has been missing since the wiki froze.
//
// It is shaped like the claim-event cron next door: a durable Redis cursor, a
// batch per tick, at-least-once delivery made safe by an idempotent upsert, and
// a transient failure that holds the cursor rather than skipping rows.
//
// It differs from both existing crons in ONE deliberate way — see the cursor
// note in Run: this one replays from 0.
type GalgameContributorSync struct {
	catalog *catalogclient.Client
	repo    *repository.GalgameContributorRepository
	rdb     *redis.Client
	// batch is the feed page size. A field rather than a bare constant so the
	// paging tests can drive several pages without a 1000-row fixture.
	batch int
}

func NewGalgameContributorSync(
	catalog *catalogclient.Client,
	repo *repository.GalgameContributorRepository,
	rdb *redis.Client,
) *GalgameContributorSync {
	return &GalgameContributorSync{
		catalog: catalog, repo: repo, rdb: rdb, batch: contributorFeedBatch,
	}
}

const (
	// contributorCursorKey is the durable cursor: the largest revision id whose
	// contributions are in the table.
	contributorCursorKey = "catalog:contrib:cron:since"
	// contributorSite is the tenant whose revisions kungal accounts for.
	contributorSite = client.ClaimSiteKungal
	// contributorMaxPerGalgame caps the strip the detail page renders. A galgame
	// with hundreds of editors is a list nobody reads to the end of, and the
	// order is by contribution, so the cap keeps the people who did the work.
	contributorMaxPerGalgame = 50

	contributorFeedBatch    = 1000
	contributorMaxPagesRun  = 50
	contributorFeedPageWait = 10 * time.Minute
)

// Run executes one sync cycle.
func (s *GalgameContributorSync) Run() {
	if s.catalog == nil || !s.catalog.Configured() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), contributorFeedPageWait)
	defer cancel()

	// A MISSING cursor starts at 0 and replays the whole feed, which is the
	// opposite of what the claim and revision crons do on their first run — and
	// the difference is the point, not an oversight.
	//
	// Those two seed at the head because their history is already reflected
	// locally: the stubs exist, the timeline rows were delivered by the feed
	// they replaced. This table starts EMPTY, and the history it needs is
	// exactly what the engine holds — the wiki-era edits, rekeyed onto catalog
	// works during the re-anchoring. Seeding at the head would leave the strip
	// showing only who has edited a game since deploy day, with the frozen
	// ledger's seed rows sitting at revision_count 0 forever.
	//
	// Replaying is affordable: the first run walks ~12.6k revisions once. An
	// unreadable cursor skips the tick (below) rather than reading as 0, so a
	// Redis hiccup cannot restart it. A cursor that is genuinely GONE (a flushed
	// Redis) does replay and does inflate revision_count — which is an ordering
	// number, never an invented or lost contributor.
	since, err := s.readCursor(ctx)
	if err != nil {
		slog.Warn("读取贡献者游标失败 (本轮跳过)", "error", err)
		return
	}

	startedFrom, maxSeen := since, since
	for pages := 1; pages <= contributorMaxPagesRun; pages++ {
		page, err := s.catalog.WorkRevisionsAfter(ctx, maxSeen, s.batch, contributorSite)
		if err != nil {
			slog.Warn("catalog 修订 feed 拉取失败 (贡献者)", "error", err, "after", maxSeen)
			break
		}
		if len(page.Items) == 0 {
			break
		}
		touches, gids := contributorTouches(page.Items)
		if err := s.repo.UpsertRevisionTouches(touches); err != nil {
			// Hold the cursor: the whole page is re-applied next tick.
			slog.Warn("贡献者入库失败, 持有游标重试", "after", maxSeen, "error", err)
			break
		}
		if err := s.repo.RefreshContributorCounts(gids); err != nil {
			// The counts are derived, so a failure here costs freshness on a
			// display number, never a contribution. Advancing past the page is
			// correct — the next page that touches these gids recomputes them.
			slog.Warn("贡献者计数刷新失败", "galgames", len(gids), "error", err)
		}
		maxSeen = maxRevisionID(page.Items, maxSeen)
		if len(page.Items) < s.batch {
			break
		}
	}

	if maxSeen > startedFrom {
		s.writeCursor(ctx, maxSeen)
		slog.Info("贡献者同步完成", "from", startedFrom, "to", maxSeen)
	}
}

// contributorTouches folds one feed page into per-(galgame, user) upserts, plus
// the gids whose counts the page invalidates.
//
// Both editing identities count as contribution and are credited separately —
// the filer and, when a reviewer shaped the change on its way in, the amender.
// When they are the SAME person the revision is one contribution, not two:
// amending one's own proposal is how the engine records a rebase, and paying it
// twice would rank self-amenders above everyone else.
//
// A revision with no product anchor is skipped rather than guessed at: the work
// id is not a gid (doc 106 R3).
func contributorTouches(items []catalogclient.WorkRevisionFeedItem) ([]repository.ContributorTouch, []int64) {
	type key struct{ gid, uid int64 }
	index := map[key]int{}
	touches := make([]repository.ContributorTouch, 0, len(items))
	gids := make([]int64, 0, len(items))
	seenGID := map[int64]bool{}

	for i := range items {
		it := &items[i]
		if it.ProductWorkID == nil || *it.ProductWorkID <= 0 {
			continue
		}
		gid := *it.ProductWorkID
		if !seenGID[gid] {
			seenGID[gid] = true
			gids = append(gids, gid)
		}
		for _, uid := range contributorUIDs(it) {
			k := key{gid, uid}
			if at, ok := index[k]; ok {
				t := &touches[at]
				t.Count++
				if it.CreatedAt.Before(t.FirstAt) {
					t.FirstAt = it.CreatedAt
				}
				if it.CreatedAt.After(t.LastAt) {
					t.LastAt = it.CreatedAt
				}
				continue
			}
			index[k] = len(touches)
			touches = append(touches, repository.ContributorTouch{
				GalgameID: gid, UserID: uid, Count: 1,
				FirstAt: it.CreatedAt, LastAt: it.CreatedAt,
			})
		}
	}
	return touches, gids
}

// contributorUIDs returns the distinct people a revision credits.
func contributorUIDs(it *catalogclient.WorkRevisionFeedItem) []int64 {
	uids := make([]int64, 0, 2)
	if it.ActorUID > 0 {
		uids = append(uids, it.ActorUID)
	}
	if it.AmenderUID != nil && *it.AmenderUID > 0 && *it.AmenderUID != it.ActorUID {
		uids = append(uids, *it.AmenderUID)
	}
	return uids
}

// maxRevisionID advances the cursor over a page. The feed is ascending, but the
// cursor is taken as the largest id SEEN rather than the last row's, so a page
// that arrives out of order can never rewind it.
func maxRevisionID(items []catalogclient.WorkRevisionFeedItem, current int64) int64 {
	for i := range items {
		if items[i].ID > current {
			current = items[i].ID
		}
	}
	return current
}

// readCursor returns the stored cursor. A missing key is 0 — a deliberate full
// replay — while an unreadable Redis is an error the caller skips the tick on:
// conflating the two would restart the replay on every Redis hiccup.
func (s *GalgameContributorSync) readCursor(ctx context.Context) (int64, error) {
	v, err := s.rdb.Get(ctx, contributorCursorKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (s *GalgameContributorSync) writeCursor(ctx context.Context, id int64) {
	if err := s.rdb.Set(ctx, contributorCursorKey, id, 0).Err(); err != nil {
		slog.Warn("写入贡献者游标失败", "id", id, "error", err)
	}
}
