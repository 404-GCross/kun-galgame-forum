package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
)

type GalgameUserStats struct {
	Published      int64 `json:"published"`
	PublishedToday int   `json:"published_today"`
	Contributed    int   `json:"contributed"`
	MergedEdits    int64 `json:"merged_edits"`
}

type GalgameUserStatsService struct {
	catalog       *catalogclient.Client
	galgameClient *client.GalgameClient
}

func NewGalgameUserStatsService(
	catalog *catalogclient.Client,
	galgameClient *client.GalgameClient,
) *GalgameUserStatsService {
	return &GalgameUserStatsService{catalog: catalog, galgameClient: galgameClient}
}

const (
	dailyClaimScan  = 100
	contributedScan = 200
)

func (s *GalgameUserStatsService) Stats(ctx context.Context, uid int64) GalgameUserStats {
	var out GalgameUserStats
	if s.catalog == nil || !s.catalog.Configured() {
		return out
	}

	if page, err := s.catalog.UserClaims(ctx, uid, catalogclient.UserClaimFilter{
		Site: submissionSite, ClaimStates: []string{catalogclient.ClaimStateLive}, Limit: 1,
	}); err != nil {
		slog.Warn("user stats: 读取已发布计数失败", "uid", uid, "error", err)
	} else {
		out.Published = page.Total
	}

	if page, err := s.catalog.UserClaims(ctx, uid, catalogclient.UserClaimFilter{
		Site: submissionSite, Limit: dailyClaimScan,
	}); err != nil {
		slog.Warn("user stats: 读取今日投稿数失败", "uid", uid, "error", err)
	} else {
		out.PublishedToday = countToday(page.Items)
	}

	if total, err := s.catalog.CountEditProposals(ctx, catalogclient.EditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, ProposerUID: uid, Status: "merged",
	}); err != nil {
		slog.Warn("user stats: 读取合并提案计数失败", "uid", uid, "error", err)
	} else {
		out.MergedEdits = total
	}

	if items, err := s.catalog.ListEditProposals(ctx, catalogclient.EditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, ProposerUID: uid,
		Status: "merged", Limit: contributedScan,
	}); err != nil {
		slog.Warn("user stats: 读取贡献条目失败", "uid", uid, "error", err)
	} else {
		out.Contributed = len(distinctEntityIDs(items))
	}
	return out
}

func countToday(items []catalogclient.UserClaimItem) int {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	count := 0
	for i := range items {
		if items[i].FirstActedAt.After(midnight) {
			count++
		}
	}
	return count
}

func distinctEntityIDs(items []catalogclient.EditProposal) []int64 {
	seen := make(map[int64]struct{}, len(items))
	out := make([]int64, 0, len(items))
	for i := range items {
		id := items[i].EntityID
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

const maxClaimPageWalk = 20

func (s *GalgameUserStatsService) PublishedGIDs(
	ctx context.Context, uid int64, page, limit int,
) ([]int, int64, error) {
	return s.claimGIDs(ctx, uid, []string{catalogclient.ClaimStateLive}, page, limit)
}

func (s *GalgameUserStatsService) claimGIDs(
	ctx context.Context, uid int64, states []string, page, limit int,
) ([]int, int64, error) {
	if page < 1 {
		page = 1
	}
	if page > maxClaimPageWalk {
		return []int{}, 0, nil
	}
	var (
		before int64
		total  int64
		items  []catalogclient.UserClaimItem
	)
	for i := 0; i < page; i++ {
		p, err := s.catalog.UserClaims(ctx, uid, catalogclient.UserClaimFilter{
			Site: submissionSite, ClaimStates: states, Before: before, Limit: limit,
		})
		if err != nil {
			return nil, 0, err
		}
		total, items, before = p.Total, p.Items, p.NextBefore
		if before == 0 {
			if i < page-1 {
				return []int{}, total, nil
			}
			break
		}
	}
	gids := make([]int, 0, len(items))
	for i := range items {
		if id := items[i].ProductWorkID; id != nil && *id > 0 {
			gids = append(gids, int(*id))
		}
	}
	return gids, total, nil
}

func (s *GalgameUserStatsService) ContributedGIDs(ctx context.Context, uid int64) ([]int, error) {
	items, err := s.catalog.ListEditProposals(ctx, catalogclient.EditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, ProposerUID: uid,
		Status: "merged", Limit: contributedScan,
	})
	if err != nil {
		return nil, err
	}
	workIDs := distinctEntityIDs(items)
	gidByWork, appErr := s.galgameClient.GIDsByCatalogIDs(ctx, workIDs)
	if appErr != nil {
		return nil, appErr
	}
	gids := make([]int, 0, len(workIDs))
	for _, id := range workIDs {
		if gid, ok := gidByWork[id]; ok {
			gids = append(gids, gid)
		}
	}
	return gids, nil
}
