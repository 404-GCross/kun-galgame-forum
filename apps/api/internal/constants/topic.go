package constants

var TopicSectionConsume = map[string]bool{
	"g-seeking": true,
	"g-other":   true,
	"t-help":    true,
}

var ValidTopicCategories = []string{"galgame", "technique", "others"}

var ValidTopicSortFields = map[string]string{
	"created":            "created",
	"view":               "view",
	"view_7d":            "view_7d",
	"view_30d":           "view_30d",
	"status_update_time": "status_update_time",
}

var ValidTopicCountSortFields = map[string]string{
	"like":     "like_count",
	"favorite": "favorite_count",
	"upvote":   "upvote_count",
}

const (
	MaxPollsPerTopic = 30
	MaxTagsPerTopic  = 7
	MaxTagLength     = 17
)
