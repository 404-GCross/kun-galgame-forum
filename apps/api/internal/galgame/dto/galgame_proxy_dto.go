package dto

type GalgameLink struct {
	ID        int       `json:"id"`
	User      UserBrief `json:"user"`
	GalgameID int       `json:"galgame_id"`
	Name      string    `json:"name"`
	Link      string    `json:"link"`
}
