package dto

type GalgameRankingRequest struct {
	Page           int    `query:"page" validate:"min=1"`
	Limit          int    `query:"limit" validate:"min=1,max=50"`
	SortField      string `query:"sort_field" validate:"required,oneof=view like favorite resource rating"`
	SortOrder      string `query:"sort_order" validate:"required,oneof=asc desc"`
	ShowNoResource bool   `query:"show_no_resource"`
}

type TopicRankingRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=50"`
	SortField string `query:"sort_field" validate:"required,oneof=view upvote like reply comment favorite"`
	SortOrder string `query:"sort_order" validate:"required,oneof=asc desc"`
}

type UserRankingRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=50"`
	SortField string `query:"sort_field" validate:"required,oneof=moemoepoint topic reply_created comment_created galgame_resource"`
	SortOrder string `query:"sort_order" validate:"required,oneof=asc desc"`
}

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type GalgameRankingItem struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	User                UserBrief `json:"user"`
	Value               float64   `json:"value"`
	SortField           string    `json:"sort_field"`
	EffectiveBannerHash string    `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL  string    `json:"effective_banner_url,omitempty"`
}

type TopicRankingItem struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	User      UserBrief `json:"user"`
	Value     int       `json:"value"`
	SortField string    `json:"sort_field"`
}

type UserRankingItem struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Bio       string `json:"bio"`
	Value     int    `json:"value"`
	SortField string `json:"sort_field"`
}
