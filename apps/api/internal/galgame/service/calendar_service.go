package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// CalendarService proxies the wiki release-calendar endpoints
// (GET /galgame/calendar[/pending|/tba]) and enriches each entry with
// forum-local data (view/like/platform + IsOnForum) via GalgameEnricher — the
// same overlay the entity detail pages use, so calendar cards match them,
// including the 未收录 marker for games the forum has never ingested (which is
// most upcoming releases). See docs/galgame_wiki/01-galgame.md §Galgame 发售月历.
type CalendarService struct {
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewCalendarService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *CalendarService {
	return &CalendarService{wikiClient: wikiClient, enricher: enricher}
}

// wikiCalendarMeta mirrors the wiki month-calendar meta (snake_case wire).
type wikiCalendarMeta struct {
	PrevMonth string `json:"prev_month"`
	NextMonth string `json:"next_month"`
	HasPrev   bool   `json:"has_prev"`
	HasNext   bool   `json:"has_next"`
	MinMonth  string `json:"min_month"`
	MaxMonth  string `json:"max_month"`
	Count     int    `json:"count"`
}

// wikiCalendarMonthResp mirrors the month envelope. Items parse into
// WikiGalgameItem (+ release_precision); wiki preloads covers/official too but
// the card only needs the scalar fields the enricher reads.
type wikiCalendarMonthResp struct {
	Month string                `json:"month"`
	Today string                `json:"today"`
	Items []dto.WikiGalgameItem `json:"items"`
	Meta  wikiCalendarMeta      `json:"meta"`
}

// wikiCalendarBucketResp covers both the pending (year) and TBA buckets — each
// is just {year?, items, meta:{count}}.
type wikiCalendarBucketResp struct {
	Year  string                `json:"year"`
	Items []dto.WikiGalgameItem `json:"items"`
	Meta  struct {
		Count int `json:"count"`
	} `json:"meta"`
}

// GetMonth returns one ISO month's release list. An empty `month` query lets
// wiki default to the current JST month. content_limit is pinned per the SFW
// cookie (sfw / all) via the shared withSFWFilter helper.
func (s *CalendarService) GetMonth(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.CalendarMonthPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/calendar", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiCalendarMonthResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	return &dto.CalendarMonthPage{
		Month: parsed.Month,
		Today: parsed.Today,
		Items: s.enricher.ToCards(ctx, parsed.Items),
		Meta: dto.CalendarMeta{
			PrevMonth: parsed.Meta.PrevMonth,
			NextMonth: parsed.Meta.NextMonth,
			HasPrev:   parsed.Meta.HasPrev,
			HasNext:   parsed.Meta.HasNext,
			MinMonth:  parsed.Meta.MinMonth,
			MaxMonth:  parsed.Meta.MaxMonth,
			Count:     parsed.Meta.Count,
		},
	}, nil
}

// GetPending returns the "year known, month undecided" bucket for a year
// (empty `year` → current JST year, server-side).
func (s *CalendarService) GetPending(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.CalendarPendingPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/calendar/pending", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiCalendarBucketResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	return &dto.CalendarPendingPage{
		Year:  parsed.Year,
		Items: s.enricher.ToCards(ctx, parsed.Items),
		Count: parsed.Meta.Count,
	}, nil
}

// GetTBA returns the global "release date to be announced" bucket.
func (s *CalendarService) GetTBA(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.CalendarTBAPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/calendar/tba", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiCalendarBucketResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	return &dto.CalendarTBAPage{
		Items: s.enricher.ToCards(ctx, parsed.Items),
		Count: parsed.Meta.Count,
	}, nil
}
