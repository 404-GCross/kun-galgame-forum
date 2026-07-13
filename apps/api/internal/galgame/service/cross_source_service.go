package service

import (
	"context"
	"encoding/json"
	stderrors "errors"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

// CrossSourceService is the BFF read face for the galgame detail page's
// cross-source data (steps 34 + 36): the three-source scores + site stats
// proxied from the wiki's public read face, and the catalog credits / entity
// reverse-lookups proxied from the infra catalog service (S2S Basic). Every
// method is a verbatim GET passthrough — no local enrichment, no DB — so the
// wiki / catalog contract IS the forum's exit contract, and the 37 frontend
// reads the infra shape (docs/integration/galgame_wiki/10 + docs/catalog).
type CrossSourceService struct {
	wikiClient    *client.GalgameClient
	catalogClient *catalogclient.Client
}

func NewCrossSourceService(wikiClient *client.GalgameClient, catalogClient *catalogclient.Client) *CrossSourceService {
	return &CrossSourceService{wikiClient: wikiClient, catalogClient: catalogClient}
}

// ──────────────────────────────────────────
// Scores / stats (wiki passthrough)
// ──────────────────────────────────────────

// Scores proxies GET /galgame/:gid/scores (three-source rating snapshot).
func (s *CrossSourceService) Scores(ctx context.Context, gid int) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.Scores(ctx, gid)
}

// Stats proxies GET /galgame/stats, preserving the ETag / 304 conditional cache.
func (s *CrossSourceService) Stats(ctx context.Context, ifNoneMatch string) (*client.StatsResult, *errors.AppError) {
	return s.wikiClient.Stats(ctx, ifNoneMatch)
}

// ──────────────────────────────────────────
// Catalog read face (S2S passthrough)
// ──────────────────────────────────────────

// WorkCredits proxies the catalog work credits, mapping catalog transport
// failures to a graceful AppError (an unreachable catalog must not 500 the page).
func (s *CrossSourceService) WorkCredits(ctx context.Context, workID int64) (json.RawMessage, *errors.AppError) {
	data, err := s.catalogClient.WorkCredits(ctx, workID)
	if err != nil {
		return nil, mapCatalogErr(err)
	}
	return data, nil
}

// NameWorks proxies the catalog name→works reverse lookup.
func (s *CrossSourceService) NameWorks(ctx context.Context, nameID int64, limit, offset int) (json.RawMessage, *errors.AppError) {
	data, err := s.catalogClient.NameWorks(ctx, nameID, limit, offset)
	if err != nil {
		return nil, mapCatalogErr(err)
	}
	return data, nil
}

// CharacterWorks proxies the catalog character→works reverse lookup.
func (s *CrossSourceService) CharacterWorks(ctx context.Context, charID int64, limit, offset int) (json.RawMessage, *errors.AppError) {
	data, err := s.catalogClient.CharacterWorks(ctx, charID, limit, offset)
	if err != nil {
		return nil, mapCatalogErr(err)
	}
	return data, nil
}

// mapCatalogErr turns a catalogclient sentinel error into a kungal AppError. A
// missing entity is a real 404; unconfigured / unreachable / upstream errors
// degrade to a clear 503 rather than a page-wide 500 — the credits block is a
// side panel, so a down catalog must not take the whole detail page with it.
func mapCatalogErr(err error) *errors.AppError {
	switch {
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return errors.ErrNotFound("未找到该条目")
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return errors.New(errors.CodeBiz, "资料库服务暂未启用", 503)
	default:
		return errors.New(errors.CodeBiz, "资料库服务暂不可用", 503)
	}
}
