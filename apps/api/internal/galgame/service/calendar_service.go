package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// upcomingMonthCap bounds how many months forward the "未发售" aggregation
// fans out (current month inclusive) — a backstop against a stray far-future
// max_month. Real announced dates rarely exceed ~1–2 years out.
const upcomingMonthCap = 24

// upcomingFetchConcurrency caps concurrent galgame calendar requests during the
// 未发售 fan-out.
const upcomingFetchConcurrency = 8

// CalendarService proxies the galgame release-calendar endpoints
// (GET /galgame/calendar[/pending|/tba]) and enriches each entry with
// forum-local data (view/like/platform + IsOnForum) via GalgameEnricher — the
// same overlay the entity detail pages use, so calendar cards match them,
// including the 未收录 marker for games the forum has never ingested (which is
// most upcoming releases). See docs/galgame_wiki/01-galgame.md §Galgame 发售月历.
type CalendarService struct {
	galgameClient *client.GalgameClient
	enricher      *GalgameEnricher
}

func NewCalendarService(galgameClient *client.GalgameClient, enricher *GalgameEnricher) *CalendarService {
	return &CalendarService{galgameClient: galgameClient, enricher: enricher}
}

// nextMoeCalendarMeta mirrors the galgame month-calendar meta (snake_case wire).
type nextMoeCalendarMeta struct {
	PrevMonth string `json:"prev_month"`
	NextMonth string `json:"next_month"`
	HasPrev   bool   `json:"has_prev"`
	HasNext   bool   `json:"has_next"`
	MinMonth  string `json:"min_month"`
	MaxMonth  string `json:"max_month"`
	Count     int    `json:"count"`
}

// nextMoeCalendarMonthResp mirrors the month envelope. Items parse into
// NextMoeGalgameItem (+ release_precision); galgame preloads covers/official too but
// the card only needs the scalar fields the enricher reads.
type nextMoeCalendarMonthResp struct {
	Month string                   `json:"month"`
	Today string                   `json:"today"`
	Items []dto.NextMoeGalgameItem `json:"items"`
	Meta  nextMoeCalendarMeta      `json:"meta"`
}

// nextMoeCalendarBucketResp covers both the pending (year) and TBA buckets — each
// is just {year?, items, meta:{count}}.
type nextMoeCalendarBucketResp struct {
	Year  string                   `json:"year"`
	Items []dto.NextMoeGalgameItem `json:"items"`
	Meta  struct {
		Count int `json:"count"`
	} `json:"meta"`
}

// fetchMonthRaw fetches + parses a single month's galgame calendar response. An
// empty `month` lets galgame default to the current JST month. content_limit is
// pinned per the SFW cookie (sfw / all) via the shared withSFWFilter helper.
func (s *CalendarService) fetchMonthRaw(
	ctx context.Context,
	month string,
	isSFW bool,
) (*nextMoeCalendarMonthResp, *errors.AppError) {
	q := url.Values{}
	if month != "" {
		q.Set("month", month)
	}
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/calendar", withSFWFilter(q, isSFW))
	if appErr != nil {
		return nil, appErr
	}

	var parsed nextMoeCalendarMonthResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}
	return &parsed, nil
}

// GetMonth returns one ISO month's release list.
func (s *CalendarService) GetMonth(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.CalendarMonthPage, *errors.AppError) {
	parsed, appErr := s.fetchMonthRaw(ctx, rawQuery.Get("month"), isSFW)
	if appErr != nil {
		return nil, appErr
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
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/calendar/pending", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}

	var parsed nextMoeCalendarBucketResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
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
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/calendar/tba", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}

	var parsed nextMoeCalendarBucketResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	return &dto.CalendarTBAPage{
		Items: s.enricher.ToCards(ctx, parsed.Items),
		Count: parsed.Meta.Count,
	}, nil
}

// GetUpcoming aggregates the not-yet-released schedule: every dated entry
// (day/month precision) with release_date >= today, from the current month up
// to the data's max month, grouped by month. The current month is fetched
// first (it carries `today` + `max_month`); the remaining months fan out in
// parallel. Enrichment runs once over the union (one DB/OAuth batch) and the
// cards are then regrouped by month.
func (s *CalendarService) GetUpcoming(
	ctx context.Context,
	isSFW bool,
) (*dto.CalendarUpcomingPage, *errors.AppError) {
	base, appErr := s.fetchMonthRaw(ctx, "", isSFW)
	if appErr != nil {
		return nil, appErr
	}
	today := base.Today
	months := monthRange(base.Month, base.Meta.MaxMonth, upcomingMonthCap)

	// rawByMonth[0] is the already-fetched current month; the rest fan out.
	rawByMonth := make([][]dto.NextMoeGalgameItem, len(months))
	rawByMonth[0] = base.Items
	if len(months) > 1 {
		var wg sync.WaitGroup
		sem := make(chan struct{}, upcomingFetchConcurrency)
		for i := 1; i < len(months); i++ {
			wg.Add(1)
			go func(idx int, m string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				// A single month failing must not sink the whole view — skip it.
				if resp, err := s.fetchMonthRaw(ctx, m, isSFW); err == nil {
					rawByMonth[idx] = resp.Items
				}
			}(i, months[i])
		}
		wg.Wait()
	}

	// Flatten in month order, keeping only the not-yet-released entries
	// (release_date >= today; lexicographic compare is valid on YYYY-MM-DD).
	var flat []dto.NextMoeGalgameItem
	for _, items := range rawByMonth {
		for _, it := range items {
			if it.ReleaseDate != nil && *it.ReleaseDate >= today {
				flat = append(flat, it)
			}
		}
	}

	// Enrich once, then regroup by the card's own month (release_date[:7]).
	cards := s.enricher.ToCards(ctx, flat)
	byMonth := make(map[string][]dto.GalgameCard, len(months))
	for _, c := range cards {
		if c.ReleaseDate == nil || len(*c.ReleaseDate) < 7 {
			continue
		}
		key := (*c.ReleaseDate)[:7]
		byMonth[key] = append(byMonth[key], c)
	}

	out := make([]dto.CalendarUpcomingMonth, 0, len(months))
	total := 0
	for _, m := range months {
		if g := byMonth[m]; len(g) > 0 {
			out = append(out, dto.CalendarUpcomingMonth{Month: m, Items: g})
			total += len(g)
		}
	}
	return &dto.CalendarUpcomingPage{Today: today, Months: out, Count: total}, nil
}

// monthRange returns the inclusive list of "YYYY-MM" months from start to end,
// capped at `cap` entries. If end precedes start (no future data), only the
// start month is returned.
func monthRange(start, end string, cap int) []string {
	sy, sm := parseYM(start)
	ey, em := parseYM(end)
	if ey < sy || (ey == sy && em < sm) {
		return []string{formatYM(sy, sm)}
	}
	out := make([]string, 0, cap)
	y, m := sy, sm
	for len(out) < cap {
		out = append(out, formatYM(y, m))
		if y == ey && m == em {
			break
		}
		if m++; m > 12 {
			m = 1
			y++
		}
	}
	return out
}

func parseYM(s string) (int, int) {
	if len(s) < 7 {
		return 0, 1
	}
	y, _ := strconv.Atoi(s[:4])
	m, _ := strconv.Atoi(s[5:7])
	if m < 1 || m > 12 {
		m = 1
	}
	return y, m
}

func formatYM(y, m int) string {
	return fmt.Sprintf("%04d-%02d", y, m)
}
