package dto

type UserRatingGalgame struct {
	ID           int         `json:"id"`
	Name         KunLanguage `json:"name"`
	ContentLimit string      `json:"content_limit"`
}

type UserRatingItem struct {
	ID           int               `json:"id"`
	User         UserBrief         `json:"user"`
	Recommend    string            `json:"recommend"`
	Overall      int               `json:"overall"`
	View         int               `json:"view"`
	GalgameType  []string          `json:"galgame_type"`
	PlayStatus   string            `json:"play_status"`
	ShortSummary string            `json:"short_summary"`
	Art          int               `json:"art"`
	Story        int               `json:"story"`
	Music        int               `json:"music"`
	Character    int               `json:"character"`
	Route        int               `json:"route"`
	System       int               `json:"system"`
	Voice        int               `json:"voice"`
	ReplayValue  int               `json:"replay_value"`
	SpoilerLevel string            `json:"spoiler_level"`
	LikeCount    int               `json:"like_count"`
	Created      string            `json:"created"`
	Updated      string            `json:"updated"`
	Galgame      UserRatingGalgame `json:"galgame"`
}

type UserRatingsResponse struct {
	RatingData []UserRatingItem `json:"rating_data"`
	Total      int64            `json:"total"`
}
