package dto

type UpsertPracticalityRequest struct {
	Rate int `json:"rate" validate:"required,min=1,max=5"`
}

type PracticalityResponse struct {
	Counts map[int]int64 `json:"counts"`
	Avg    float64       `json:"avg"`
	Mine   *int          `json:"mine"`
}

type PracticalitySummary struct {
	Counts map[int]int64 `json:"counts"`
	Avg    float64       `json:"avg"`
}
