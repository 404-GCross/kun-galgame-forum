package dto

type GetStatsRequest struct {
	Days int `query:"days" validate:"min=1,max=365"`
}

type OverviewItem struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type DailyStatRow map[string]any
