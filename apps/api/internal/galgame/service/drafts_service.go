package service

import (
	"context"
	"encoding/json"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// DraftsService proxies the wiki's unclaimed-VNDB-draft list
// (GET /galgame/drafts, status=2) and enriches each entry into the shared
// GalgameCard shape via GalgameEnricher — the same overlay the calendar /
// entity detail pages use. Drafts are never on the forum, so every card comes
// back IsOnForum=false + Status=2, which the frontend renders as a "未在论坛发布"
// claim card linking to the publish wizard (identical to the calendar's
// status=2 cards). See docs/galgame_wiki §drafts.
type DraftsService struct {
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewDraftsService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *DraftsService {
	return &DraftsService{wikiClient: wikiClient, enricher: enricher}
}

// wikiDraftsResp mirrors the wiki {items, total} draft envelope. Items parse
// into WikiGalgameItem; the enricher reads the scalar card fields (name /
// banner / status / content_limit) and fuses in local stats.
type wikiDraftsResp struct {
	Items []dto.WikiGalgameItem `json:"items"`
	Total int64                 `json:"total"`
}

// GetDrafts returns one page of unclaimed VNDB drafts as enriched cards.
// page/limit are pre-clamped by the handler (parseCollectionPage).
func (s *DraftsService) GetDrafts(
	ctx context.Context,
	page, limit int,
	isSFW bool,
) (*dto.DraftsPage, *errors.AppError) {
	data, appErr := s.wikiClient.Drafts(ctx, page, limit, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiDraftsResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	return &dto.DraftsPage{
		Items: s.enricher.ToCards(ctx, parsed.Items),
		Total: parsed.Total,
	}, nil
}
