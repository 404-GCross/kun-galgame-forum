package handler

import (
	"context"
	stderrors "errors"
	"log/slog"

	"kun-galgame-api/internal/news/dto"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/newsclient"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

const defaultLimit = 20

type NewsHandler struct {
	news *newsclient.Client
	user *userclient.Client
}

func NewNewsHandler(news *newsclient.Client, user *userclient.Client) *NewsHandler {
	return &NewsHandler{news: news, user: user}
}

type feedRequest struct {
	Lane   string `query:"lane" validate:"omitempty,oneof=news column"`
	Source string `query:"source"`
	Cursor string `query:"cursor"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=50"`
}

func (h *NewsHandler) GetFeed(c fiber.Ctx) error {
	var req feedRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Limit == 0 {
		req.Limit = defaultLimit
	}

	feed, err := h.news.Feed(c.Context(), newsclient.FeedQuery{
		Lane:   req.Lane,
		Source: req.Source,
		Cursor: req.Cursor,
		Limit:  req.Limit,
	})
	if err != nil {
		return response.Error(c, feedError(err))
	}
	return response.OK(c, h.mapFeed(c.Context(), feed))
}

func (h *NewsHandler) mapFeed(ctx context.Context, feed *newsclient.Feed) dto.NewsFeed {
	upstream := make(map[string]newsclient.Source)
	items := make([]dto.NewsItem, 0, len(feed.Items))
	for _, it := range feed.Items {
		items = append(items, dto.NewsItem{
			ID:          it.ID,
			SourceKey:   it.Source.Key,
			Lane:        it.Lane,
			Title:       it.Title,
			Preview:     it.Preview,
			SourceURL:   it.SourceURL,
			BannerURL:   it.BannerURL,
			PublishedAt: it.PublishedAt,
		})
		if _, ok := upstream[it.Source.Key]; !ok {
			upstream[it.Source.Key] = it.Source
		}
	}

	publishers := h.publishers(ctx, upstream)
	sources := make(map[string]dto.NewsSource, len(upstream))
	for key, src := range upstream {
		sources[key] = dto.NewsSource{
			Key:         key,
			Name:        src.DisplayName,
			HomepageURL: src.HomepageURL,
			ColumnURL:   src.ColumnURL,
			Attribution: src.Attribution,
			Publisher:   publishers[src.PublisherUID],
		}
	}

	return dto.NewsFeed{
		Items:      items,
		Sources:    sources,
		Count:      feed.Count,
		NextCursor: feed.NextCursor,
	}
}

// publishers hydrates the forum account each partner publishes under, which is
// what news_source.publisher_uid names. A miss leaves the source without one
// rather than failing the page: the display name and the attribution text stand
// on their own, and an unreachable OAuth is not worth losing the feed over.
func (h *NewsHandler) publishers(ctx context.Context, sources map[string]newsclient.Source) map[int64]*dto.UserBrief {
	out := make(map[int64]*dto.UserBrief, len(sources))
	if h.user == nil {
		return out
	}
	ids := make([]int, 0, len(sources))
	for _, src := range sources {
		if src.PublisherUID > 0 {
			ids = append(ids, int(src.PublisherUID))
		}
	}
	if len(ids) == 0 {
		return out
	}
	users, err := h.user.Users(ctx, ids)
	if err != nil {
		slog.Warn("news: 获取情报发布者信息失败", "error", err)
		return out
	}
	for id, u := range users {
		if !userclient.IsRenderable(u) {
			continue
		}
		out[int64(id)] = &dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return out
}

func feedError(err error) *errors.AppError {
	switch {
	case stderrors.Is(err, newsclient.ErrNotConfigured):
		return errors.New(errors.CodeBiz, "情报服务未配置", fiber.StatusServiceUnavailable)
	case stderrors.Is(err, newsclient.ErrBadRequest):
		return errors.ErrBadRequest("情报服务拒绝了该查询")
	}
	slog.Error("news: 情报服务请求失败", "error", err)
	return errors.New(errors.CodeBiz, "情报服务暂时不可用", fiber.StatusBadGateway)
}
