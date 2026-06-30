package dto

// Release-calendar response shapes. These wrap the wiki calendar endpoints
// (GET /galgame/calendar[/pending|/tba]) after local enrichment, re-keying the
// wiki's snake_case meta to the camelCase the frontend consumes elsewhere.
// See docs/galgame_wiki/01-galgame.md §Galgame 发售月历.

// CalendarMeta is the month-navigation envelope. HasPrev/HasNext are
// data-boundary clamps (no day/month-precision data before MinMonth or after
// MaxMonth, per the current content_limit), so the FE disables paging at the
// edges instead of walking into empty months forever.
type CalendarMeta struct {
	PrevMonth string `json:"prevMonth"`
	NextMonth string `json:"nextMonth"`
	HasPrev   bool   `json:"hasPrev"`
	HasNext   bool   `json:"hasNext"`
	MinMonth  string `json:"minMonth"`
	MaxMonth  string `json:"maxMonth"`
	Count     int    `json:"count"`
}

// CalendarMonthPage is one ISO month's release list (day + month precision),
// already sorted by the wiki (date asc; within a day, exact-day entries before
// the "日未定" month-precision tail). Today is JST, for the 今日 marker.
type CalendarMonthPage struct {
	Month string        `json:"month"`
	Today string        `json:"today"`
	Items []GalgameCard `json:"items"`
	Meta  CalendarMeta  `json:"meta"`
}

// CalendarPendingPage is the "year known, month undecided" bucket
// (release_precision='year') for a given year.
type CalendarPendingPage struct {
	Year  string        `json:"year"`
	Items []GalgameCard `json:"items"`
	Count int           `json:"count"`
}

// CalendarTBAPage is the global "release date to be announced" bucket
// (release_precision='tba').
type CalendarTBAPage struct {
	Items []GalgameCard `json:"items"`
	Count int           `json:"count"`
}
