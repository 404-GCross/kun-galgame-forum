package dto

// UserContentStats is the per-type breakdown of a user's kungal content,
// used both to preview a purge (GET) and to report what was deleted (DELETE).
type UserContentStats struct {
	Topics           int64 `json:"topics"`
	Replies          int64 `json:"replies"`
	TopicComments    int64 `json:"topic_comments"`
	GalgameComments  int64 `json:"galgame_comments"`
	Ratings          int64 `json:"ratings"`
	RatingComments   int64 `json:"rating_comments"`
	Resources        int64 `json:"resources"`
	Websites         int64 `json:"websites"`
	WebsiteComments  int64 `json:"website_comments"`
	Toolsets         int64 `json:"toolsets"`
	ToolsetResources int64 `json:"toolset_resources"`
	ToolsetComments  int64 `json:"toolset_comments"`
	ChatMessages     int64 `json:"chat_messages"`
	Messages         int64 `json:"messages"`
	Interactions     int64 `json:"interactions"`
	Total            int64 `json:"total"`
}
