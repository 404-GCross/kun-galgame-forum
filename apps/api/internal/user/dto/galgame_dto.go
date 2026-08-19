package dto

type KunLanguage struct {
	EnUs string `json:"en-us"`
	JaJp string `json:"ja-jp"`
	ZhCn string `json:"zh-cn"`
	ZhTw string `json:"zh-tw"`
}

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type UserGalgameCard struct {
	ID                  int         `json:"id"`
	Name                KunLanguage `json:"name"`
	User                UserBrief   `json:"user"`
	ContentLimit        string      `json:"content_limit"`
	View                int         `json:"view"`
	LikeCount           int         `json:"like_count"`
	ResourceUpdateTime  string      `json:"resource_update_time"`
	Platform            []string    `json:"platform"`
	Language            []string    `json:"language"`
	ReleaseDate         *string     `json:"release_date"`
	ReleaseDateTBA      bool        `json:"release_date_tba"`
	EffectiveBannerHash string      `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL  string      `json:"effective_banner_url,omitempty"`
}

type UserGalgameComment struct {
	ID          int64     `json:"id"`
	GalgameID   int       `json:"galgame_id"`
	Content     string    `json:"content"`
	ContentHtml string    `json:"content_html"`
	User        UserBrief `json:"user"`
	Created     string    `json:"created"`
	Deleted     bool      `json:"deleted"`
}

type UserGalgameCommentsRequest struct {
	Type  string `query:"type" validate:"required"`
	After string `query:"after"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}
