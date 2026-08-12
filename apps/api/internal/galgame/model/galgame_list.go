package model

type GalgameListFilter struct {
	Type                 string
	Language             string
	Platform             string
	GameType             string
	SortField            string
	SortOrder            string
	IncludeProviders     []string
	ExcludeOnlyProviders []string
	ReleasedFrom         string
	ReleasedTo           string
	ReleasedMonths       []int
	MinRatingCount       int
	MinRating            float64
	ShowNoResource       bool
	RestrictIDs          []int
	Page                 int
	Limit                int
}

type GalgameResourceMeta struct {
	GalgameID int    `gorm:"column:galgame_id"`
	Platform  string `gorm:"column:platform"`
	Language  string `gorm:"column:language"`
}
