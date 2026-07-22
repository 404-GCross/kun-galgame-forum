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
// Taxonomy revision history (/galgame-{tag,official,engine,series}/:id/revisions)
// ──────────────────────────────────────────

// GalgameRevision is a single edit history entry. Since E3b it serves the
// taxonomy revision reads only — the galgame history/PR list shapes retired
// with the old wire (kungal reads the editing engine now).
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
