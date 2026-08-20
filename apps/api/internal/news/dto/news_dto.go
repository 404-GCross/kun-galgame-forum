package dto

import "time"

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// NewsSource is sent once per page and named from each item by SourceKey. The
// upstream inlines the whole block on every item instead, because a consumer
// that renders one item in isolation must still be able to show the partner's
// attribution and link back — the two conditions 月幕 and Galgame 批评 attached
// to republication. Collapsing to a map keeps that true here: an item is not
// renderable without a lookup that necessarily yields both.
type NewsSource struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	HomepageURL string     `json:"homepage_url"`
	ColumnURL   string     `json:"column_url"`
	Attribution string     `json:"attribution"`
	Publisher   *UserBrief `json:"publisher"`
}

// NewsItem carries no article body on purpose: the partners authorised an
// index, so preview plus banner is the whole of it and SourceURL is the only
// route to the full text.
type NewsItem struct {
	ID          int64     `json:"id"`
	SourceKey   string    `json:"source_key"`
	Lane        string    `json:"lane"`
	Title       string    `json:"title"`
	Preview     string    `json:"preview"`
	SourceURL   string    `json:"source_url"`
	BannerURL   string    `json:"banner_url"`
	PublishedAt time.Time `json:"published_at"`
}

type NewsFeed struct {
	Items      []NewsItem            `json:"items"`
	Sources    map[string]NewsSource `json:"sources"`
	Count      int64                 `json:"count"`
	NextCursor string                `json:"next_cursor"`
}
