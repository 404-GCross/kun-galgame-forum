package dto

// ──────────────────────────────────────────
// Galgame Links (/galgame/:gid/link/all)
// ──────────────────────────────────────────

// GalgameLink is an external link attached to a galgame, with the creator user
// resolved from the local DB.
type GalgameLink struct {
	ID        int       `json:"id"`
	User      UserBrief `json:"user"`
	GalgameID int       `json:"galgame_id"`
	Name      string    `json:"name"`
	Link      string    `json:"link"`
}

// ──────────────────────────────────────────
// Galgame History (/galgame/:gid/history/all)
// ──────────────────────────────────────────

// GalgameRevision is a single edit history entry.
type GalgameRevision struct {
	ID       int       `json:"id"`
	Revision int       `json:"revision"`
	Action   string    `json:"action"`
	Note     string    `json:"note"`
	User     UserBrief `json:"user"`
	IsMinor  bool      `json:"is_minor"`
	Created  string    `json:"created"`
}

type GalgameRevisionListPage struct {
	Items []GalgameRevision `json:"items"`
	Total int64             `json:"total"`
}

// ──────────────────────────────────────────
// Galgame PRs (/galgame/:gid/pr/all)
// ──────────────────────────────────────────

// GalgamePR is a pending/completed pull request on a galgame.
type GalgamePR struct {
	ID            int       `json:"id"`
	GalgameID     int       `json:"galgame_id"`
	Status        int       `json:"status"`
	Note          string    `json:"note"`
	BaseRevision  int       `json:"base_revision"`
	User          UserBrief `json:"user"`
	CompletedTime *string   `json:"completed_time"`
	Created       string    `json:"created"`
}

type GalgamePRListPage struct {
	Items []GalgamePR `json:"items"`
	Total int64       `json:"total"`
}
