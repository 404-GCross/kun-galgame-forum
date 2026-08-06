package service

// The read half of the best-cover vote facet on the detail page (wave 176).
//
// It is BEST-EFFORT by construction: an advisory count is never worth failing
// a page over, so every failure here leaves the covers exactly as the public
// face rendered them — no ids, no counts — and the FE simply shows no vote
// control. The page it decorates is the same page it would have been.

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/galgame/dto"
)

// hydrateCoverVotes joins the catalog's cover row ids + vote tallies onto the
// covers already projected from the public face, matching on image hash (the
// one key both faces publish — see catalogclient.WorkCoverVotes).
//
// viewerUID 0 = anonymous: the counts are still filled in, the voted flags are
// not.
func (s *GalgameService) hydrateCoverVotes(ctx context.Context, gid, viewerUID int, covers []dto.GalgameCover) {
	if s.catalog == nil || len(covers) == 0 || !s.catalog.Configured() {
		return
	}
	ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})
	if appErr != nil {
		return
	}
	workID, ok := ids[gid]
	if !ok {
		return
	}
	tallies, err := s.catalog.WorkCoverVotes(ctx, workID, int64(viewerUID))
	if err != nil {
		slog.Warn("galgame detail: cover vote tallies unavailable", "gid", gid, "error", err)
		return
	}
	byHash := make(map[string]int, len(tallies))
	for i, t := range tallies {
		if t.ImageHash != "" {
			byHash[t.ImageHash] = i
		}
	}
	for i := range covers {
		t, ok := byHash[covers[i].ImageHash]
		if !ok {
			continue
		}
		covers[i].ID = tallies[t].ID
		covers[i].VoteCount = tallies[t].VoteCount
		covers[i].Voted = tallies[t].Voted
	}
}
