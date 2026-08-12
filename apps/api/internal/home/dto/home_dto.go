package dto

import "time"

type LocaleName struct {
	EnUS string `json:"en-us"`
	JaJP string `json:"ja-jp"`
	ZhCN string `json:"zh-cn"`
	ZhTW string `json:"zh-tw"`
}

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type HomeGalgame struct {
	ID                  int        `json:"id"`
	Name                LocaleName `json:"name"`
	Banner              string     `json:"banner"`
	User                UserBrief  `json:"user"`
	ContentLimit        string     `json:"content_limit"`
	View                int        `json:"view"`
	LikeCount           int        `json:"like_count"`
	ResourceUpdateTime  string     `json:"resource_update_time"`
	Platform            []string   `json:"platform"`
	Language            []string   `json:"language"`
	EffectiveBannerHash string     `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL  string     `json:"effective_banner_url,omitempty"`
}

type HomeTopic struct {
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	View             int        `json:"view"`
	LikeCount        int        `json:"like_count"`
	ReplyCount       int        `json:"reply_count"`
	CommentCount     int        `json:"comment_count"`
	HasBestAnswer    bool       `json:"has_best_answer"`
	IsPollTopic      bool       `json:"is_poll_topic"`
	IsNSFWTopic      bool       `json:"is_nsfw_topic"`
	Section          []string   `json:"section"`
	User             UserBrief  `json:"user"`
	Status           int        `json:"status"`
	UpvoteTime       *time.Time `json:"upvote_time"`
	StatusUpdateTime time.Time  `json:"status_update_time"`
}

type HomeResponse struct {
	Galgames []HomeGalgame `json:"galgames"`
	Topics   []HomeTopic   `json:"topics"`
}
